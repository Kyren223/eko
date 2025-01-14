package core

import (
	"crypto/ed25519"
	"fmt"
	"log"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/core/chat"
	"github.com/kyren223/eko/internal/client/ui/core/frequencycreation"
	"github.com/kyren223/eko/internal/client/ui/core/frequencylist"
	"github.com/kyren223/eko/internal/client/ui/core/frequencyupdate"
	"github.com/kyren223/eko/internal/client/ui/core/networkcreation"
	"github.com/kyren223/eko/internal/client/ui/core/networkjoin"
	"github.com/kyren223/eko/internal/client/ui/core/networklist"
	"github.com/kyren223/eko/internal/client/ui/core/networkupdate"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/client/ui/loadscreen"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

var (
	connectingToServer = "Connecting to server.."
	connectionFailed   = "Connection failed - retrying in %d sec..."
	connectionTimeout  = 5 * time.Second
	initialTimeout     = 3750 * time.Millisecond
	timerInterval      = 50 * time.Millisecond
)

const (
	FocusNetworkList = iota
	FocusFrequencyList
	FocusChat
	FocusMax
)

type Model struct {
	name    string
	privKey ed25519.PrivateKey

	loading   loadscreen.Model
	timer     timer.Model
	timeout   time.Duration
	connected bool

	networkCreationPopup   *networkcreation.Model
	networkUpdatePopup     *networkupdate.Model
	networkJoinPopup       *networkjoin.Model
	frequencyCreationPopup *frequencycreation.Model
	frequencyUpdatePopup   *frequencyupdate.Model
	networkList            networklist.Model
	frequencyList          frequencylist.Model
	chat                   chat.Model
	focus                  int
}

func New(privKey ed25519.PrivateKey, name string) Model {
	m := Model{
		name:                   name,
		privKey:                privKey,
		loading:                loadscreen.New(connectingToServer),
		timer:                  newTimer(initialTimeout),
		timeout:                initialTimeout,
		connected:              false,
		networkCreationPopup:   nil,
		networkUpdatePopup:     nil,
		networkJoinPopup:       nil,
		frequencyCreationPopup: nil,
		frequencyUpdatePopup:   nil,
		networkList:            networklist.New(),
		frequencyList:          frequencylist.New(),
		chat:                   chat.New(70),
		focus:                  FocusNetworkList,
	}
	m.move(0) // Update focus

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(gateway.Connect(m.privKey, connectionTimeout), m.loading.Init())
}

func (m Model) View() string {
	if !m.connected {
		return m.loading.View()
	}

	networkList := m.networkList.View()
	frequencyList := m.frequencyList.View()
	chat := m.chat.View()
	result := lipgloss.JoinHorizontal(lipgloss.Top, networkList, frequencyList, chat)

	result = lipgloss.Place(
		ui.Width, ui.Height,
		lipgloss.Left, lipgloss.Top,
		result,
	)

	var popup string
	if m.networkCreationPopup != nil {
		popup = m.networkCreationPopup.View()
	} else if m.networkUpdatePopup != nil {
		popup = m.networkUpdatePopup.View()
	} else if m.frequencyCreationPopup != nil {
		popup = m.frequencyCreationPopup.View()
	} else if m.frequencyUpdatePopup != nil {
		popup = m.frequencyUpdatePopup.View()
	} else if m.networkJoinPopup != nil {
		popup = m.networkJoinPopup.View()
	}
	if popup != "" {
		x := (ui.Width - lipgloss.Width(popup)) / 2
		y := (ui.Height - lipgloss.Height(popup)) / 2
		result = ui.PlaceOverlay(x, y, popup, result)
	}

	return result
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.connected {
		cmd := m.updateConnected(msg)
		return m, cmd
	} else {
		cmd := m.updateNotConnected(msg)
		return m, cmd
	}
}

func (m *Model) updateNotConnected(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case gateway.ConnectionEstablished:
		state.UserID = (*snowflake.ID)(&msg)
		m.connected = true
		m.timeout = initialTimeout
		return tea.Batch(m.timer.Stop(), gateway.Send(&packet.GetUserData{}))

	case gateway.ConnectionFailed:
		log.Println("failed to connect:", msg)
		m.timer = newTimer(m.timeout)
		m.updateLoadScreenContent()
		return m.timer.Start()

	case timer.TimeoutMsg:
		m.timeout = min(m.timeout*2, time.Minute)
		m.loading.SetContent(connectingToServer)
		return gateway.Connect(m.privKey, connectionTimeout)

	case timer.StartStopMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return cmd

	case timer.TickMsg:
		m.updateLoadScreenContent()
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return cmd

	case spinner.TickMsg:
		var loadscreenCmd tea.Cmd
		m.loading, loadscreenCmd = m.loading.Update(msg)
		return loadscreenCmd

	default:
		return nil
	}
}

func (m *Model) updateConnected(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case ui.QuitMsg:
		gateway.Disconnect()

	case gateway.ConnectionLost:
		state.UserID = nil
		m.connected = false
		m.timeout = initialTimeout
		return gateway.Connect(m.privKey, connectionTimeout)

	case *packet.Error:
		err := "new connection from another location, closing this one"
		if msg.PktType == packet.PacketError && err == msg.Error {
			return ui.Transition(ui.NewAuth())
		}

	case *packet.SetUserData:
		state.FromJsonUserData(msg.Data)

	case *packet.NetworksInfo:
		state.UpdateNetworks(msg)
		networkId := state.NetworkId(m.networkList.Index())
		if networkId == nil && m.networkList.Index() != networklist.PeersIndex {
			m.networkList.SetIndex(m.networkList.Index() - 1)
			m.frequencyList.SetNetworkIndex(m.networkList.Index())
			m.chat.SetFrequency(m.networkList.Index(), m.frequencyList.Index())
		}

	case *packet.MembersInfo:
		state.UpdateMembers(msg)

	case *packet.FrequenciesInfo:
		state.UpdateFrequencies(msg)
		networkId := state.NetworkId(m.networkList.Index())
		if networkId == nil {
			return nil
		}
		index := m.frequencyList.Index()
		length := len(state.State.Frequencies[*networkId])
		if index >= length {
			m.frequencyList.SetIndex(index - 1)
			m.chat.SetFrequency(m.networkList.Index(), m.frequencyList.Index())
		}

	case *packet.MessagesInfo:
		state.UpdateMessages(msg)

	case tea.KeyMsg:
		switch msg.String() {
		case "n":
			switch m.focus {
			case FocusNetworkList:
				if !m.HasPopup() {
					popup := networkcreation.New()
					m.networkCreationPopup = &popup
				} else {
					cmd := m.updatePopups(msg)
					if cmd != nil {
						return cmd
					}
				}
			case FocusFrequencyList:
				if !m.HasPopup() {
					networkId := state.NetworkId(m.networkList.Index())
					if networkId == nil {
						return nil
					}
					member := state.State.Members[*networkId][*state.UserID]
					if !member.IsAdmin {
						return nil
					}
					popup := frequencycreation.New(*networkId)
					m.frequencyCreationPopup = &popup
				} else {
					cmd := m.updatePopups(msg)
					if cmd != nil {
						return cmd
					}
				}

			default:
				cmd := m.updatePopups(msg)
				if cmd != nil {
					return cmd
				}
			}

		case "a":
			if m.focus == FocusNetworkList && !m.HasPopup() {
				popup := networkjoin.New()
				m.networkJoinPopup = &popup
			} else {
				m.updatePopups(msg)
			}

		case "i":
			index := m.networkList.Index()
			networkFocus := m.focus == FocusNetworkList
			if !m.HasPopup() && networkFocus && index != networklist.PeersIndex {
				networkId := state.NetworkId(index)
				if networkId != nil {
					_ = clipboard.WriteAll(networkId.String())
				}
			} else {
				m.updatePopups(msg)
			}

		case "u":
			index := m.networkList.Index()
			networkFocus := m.focus == FocusNetworkList
			frequencyFocus := m.focus == FocusFrequencyList
			if !m.HasPopup() && networkFocus && index != networklist.PeersIndex {
				networkId := state.NetworkId(index)
				if networkId != nil {
					popup := networkupdate.New(*networkId)
					m.networkUpdatePopup = &popup
				}
			} else if !m.HasPopup() && frequencyFocus {
				networkId := state.NetworkId(index)
				if networkId != nil {
					index := m.frequencyList.Index()
					popup := frequencyupdate.New(*networkId, index)
					m.frequencyUpdatePopup = &popup
				}
			} else {
				cmd := m.updatePopups(msg)
				if cmd != nil {
					return cmd
				}
			}

		case "esc":
			if m.HasPopup() {
				m.networkCreationPopup = nil
				m.networkUpdatePopup = nil
				m.frequencyCreationPopup = nil
				m.frequencyUpdatePopup = nil
				m.networkJoinPopup = nil
			}

		case "enter":
			if m.networkCreationPopup != nil {
				cmd := m.networkCreationPopup.Select()
				if cmd != nil {
					m.networkCreationPopup = nil
				}
				return cmd
			} else if m.networkUpdatePopup != nil {
				cmd := m.networkUpdatePopup.Select()
				if cmd != nil {
					m.networkUpdatePopup = nil
				}
				return cmd
			} else if m.frequencyCreationPopup != nil {
				cmd := m.frequencyCreationPopup.Select()
				if cmd != nil {
					m.frequencyCreationPopup = nil
				}
				return cmd
			} else if m.frequencyUpdatePopup != nil {
				cmd := m.frequencyUpdatePopup.Select()
				if cmd != nil {
					m.frequencyUpdatePopup = nil
				}
				return cmd
			} else if m.networkJoinPopup != nil {
				cmd := m.networkJoinPopup.Select()
				if cmd != nil {
					m.networkJoinPopup = nil
				}
				return cmd
			}

		default:
			cmd := m.updatePopups(msg)
			if cmd != nil {
				return cmd
			}

			if m.focus != FocusChat || !m.chat.Locked() {
				left := msg.String() == "H"
				right := msg.String() == "L"
				direction := 0
				if left {
					direction = -1
				} else if right {
					direction = 1
				}
				m.move(direction)
			}
		}
	}

	if m.HasPopup() {
		return nil
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.networkList, cmd = m.networkList.Update(msg)
	cmds = append(cmds, cmd)

	m.frequencyList.SetNetworkIndex(m.networkList.Index())
	m.frequencyList, cmd = m.frequencyList.Update(msg)
	cmds = append(cmds, cmd)

	cmd = m.chat.SetFrequency(m.networkList.Index(), m.frequencyList.Index())
	cmds = append(cmds, cmd)
	m.chat, cmd = m.chat.Update(msg)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

func (m *Model) updateLoadScreenContent() {
	seconds := m.timer.Timeout.Round(time.Second) / time.Second
	m.loading.SetContent(fmt.Sprintf(connectionFailed, seconds))
}

func newTimer(timeout time.Duration) timer.Model {
	return timer.NewWithInterval(timeout.Truncate(time.Second)+(time.Second/2), timerInterval)
}

func (m *Model) move(direction int) {
	focus := m.focus + direction
	m.focus = max(0, min(FocusMax-1, focus))

	switch m.focus {
	case FocusNetworkList:
		m.frequencyList.Blur()
		m.chat.Blur()
		m.networkList.Focus()
	case FocusFrequencyList:
		m.networkList.Blur()
		m.chat.Blur()
		m.frequencyList.Focus()
	case FocusChat:
		m.networkList.Blur()
		m.frequencyList.Blur()
		m.chat.Focus()
	default:
		assert.Never("missing switch statement field in move", "focus", m.focus)
	}
}

func (m *Model) updatePopups(msg tea.Msg) tea.Cmd {
	if m.networkCreationPopup != nil {
		popup, cmd := m.networkCreationPopup.Update(msg)
		m.networkCreationPopup = &popup
		return cmd
	} else if m.networkUpdatePopup != nil {
		popup, cmd := m.networkUpdatePopup.Update(msg)
		m.networkUpdatePopup = &popup
		return cmd
	} else if m.frequencyCreationPopup != nil {
		popup, cmd := m.frequencyCreationPopup.Update(msg)
		m.frequencyCreationPopup = &popup
		return cmd
	} else if m.frequencyUpdatePopup != nil {
		popup, cmd := m.frequencyUpdatePopup.Update(msg)
		m.frequencyUpdatePopup = &popup
		return cmd
	} else if m.networkJoinPopup != nil {
		popup, cmd := m.networkJoinPopup.Update(msg)
		m.networkJoinPopup = &popup
		return cmd
	}
	return nil
}

func (m *Model) HasPopup() bool {
	return m.networkCreationPopup != nil ||
		m.networkUpdatePopup != nil ||
		m.frequencyCreationPopup != nil ||
		m.frequencyUpdatePopup != nil ||
		m.networkJoinPopup != nil
}

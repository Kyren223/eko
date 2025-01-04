package core

import (
	"crypto/ed25519"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/btree"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/core/chat"
	"github.com/kyren223/eko/internal/client/ui/core/frequencycreation"
	"github.com/kyren223/eko/internal/client/ui/core/frequencylist"
	"github.com/kyren223/eko/internal/client/ui/core/networkcreation"
	"github.com/kyren223/eko/internal/client/ui/core/networkjoin"
	"github.com/kyren223/eko/internal/client/ui/core/networklist"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/client/ui/loadscreen"
	"github.com/kyren223/eko/internal/data"
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
	networkJoinPopup       *networkjoin.Model
	frequencyCreationPopup *frequencycreation.Model
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
		networkJoinPopup:       nil,
		frequencyCreationPopup: nil,
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
	} else if m.frequencyCreationPopup != nil {
		popup = m.frequencyCreationPopup.View()
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
		state.State.UserID = (*snowflake.ID)(&msg)
		m.connected = true
		m.timeout = initialTimeout
		return m.timer.Stop()

	case gateway.ConnectionFailed:
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
		state.State.UserID = nil
		m.connected = false
		m.timeout = initialTimeout
		return tea.Batch(gateway.Connect(m.privKey, connectionTimeout), m.loading.Init())

	case *packet.NetworksInfo:
		if msg.Set {
			state.State.Networks = msg.Networks
		} else {
			networks := state.State.Networks

			for _, newNetwork := range msg.Networks {
				add := true
				for i, existingNetwork := range networks {
					if existingNetwork.ID == newNetwork.ID {
						add = false
						if newNetwork.Position == -1 {
							newNetwork.Position = existingNetwork.Position
						}
						networks[i] = newNetwork
						break
					}
				}
				if add {
					networks = append(networks, newNetwork)
				}
			}

			networks = slices.DeleteFunc(networks, func(network packet.FullNetwork) bool {
				return slices.Contains(msg.RemoveNetworks, network.ID)
			})
			slices.SortFunc(networks, func(a, b packet.FullNetwork) int {
				return a.Position - b.Position
			})
			log.Println(networks)
			state.State.Networks = networks
		}

	case *packet.FrequenciesInfo:
		var network *packet.FullNetwork
		for i, fullNetwork := range state.State.Networks {
			if fullNetwork.ID == msg.Network {
				network = &state.State.Networks[i]
			}
		}

		if msg.Set {
			network.Frequencies = msg.Frequencies
		} else {
			frequencies := network.Frequencies
			frequencies = append(frequencies, msg.Frequencies...)
			frequencies = slices.DeleteFunc(frequencies, func(frequency data.Frequency) bool {
				return slices.Contains(msg.RemoveFrequencies, frequency.ID)
			})
			slices.SortFunc(frequencies, func(a, b data.Frequency) int {
				return int(a.Position - b.Position)
			})
			log.Println(frequencies)
			network.Frequencies = frequencies
		}

	case *packet.MessagesInfo:
		for _, id := range msg.RemoveMessages {
			for _, btree := range state.State.Messages {
				btree.Delete(data.Message{ID: id})
			}
		}

		for _, message := range msg.Messages {
			msgSource := message.FrequencyID
			if msgSource == nil {
				msgSource = message.ReceiverID
			}
			bt := state.State.Messages[*msgSource]
			if bt == nil {
				bt = btree.NewG(2, func(a, b data.Message) bool {
					return a.ID < b.ID
				})
				state.State.Messages[*msgSource] = bt
			}
			bt.ReplaceOrInsert(message)

		}

	case tea.KeyMsg:
		switch msg.String() {
		case "n":
			switch m.focus {
			case FocusNetworkList:
				if m.networkCreationPopup == nil && m.networkJoinPopup == nil {
					popup := networkcreation.New()
					m.networkCreationPopup = &popup
				}
			case FocusFrequencyList:
				if m.frequencyCreationPopup == nil {
					index := m.networkList.Index()
					network := state.State.Networks[index]
					popup := frequencycreation.New(network.ID)
					m.frequencyCreationPopup = &popup
				}
			}

		case "i":
			if m.focus == FocusNetworkList && m.networkJoinPopup == nil && m.networkCreationPopup == nil {
				popup := networkjoin.New()
				m.networkJoinPopup = &popup
			}

		case "esc":
			if m.networkCreationPopup != nil {
				m.networkCreationPopup = nil
			} else if m.frequencyCreationPopup != nil {
				m.frequencyCreationPopup = nil
			} else if m.networkJoinPopup != nil {
				m.networkJoinPopup = nil
			}

		case "enter":
			if m.networkCreationPopup != nil {
				cmd := m.networkCreationPopup.Select()
				if cmd != nil {
					m.networkCreationPopup = nil
				}
				return cmd
			} else if m.frequencyCreationPopup != nil {
				cmd := m.frequencyCreationPopup.Select()
				if cmd != nil {
					m.frequencyCreationPopup = nil
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
			if m.networkCreationPopup != nil {
				popup, cmd := m.networkCreationPopup.Update(msg)
				m.networkCreationPopup = &popup
				return cmd
			} else if m.frequencyCreationPopup != nil {
				popup, cmd := m.frequencyCreationPopup.Update(msg)
				m.frequencyCreationPopup = &popup
				return cmd
			} else if m.networkJoinPopup != nil {
				popup, cmd := m.networkJoinPopup.Update(msg)
				m.networkJoinPopup = &popup
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

	isPopup := m.networkCreationPopup != nil ||
		m.frequencyCreationPopup != nil ||
		m.networkJoinPopup != nil
	if isPopup {
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

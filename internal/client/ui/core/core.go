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
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/core/frequencycreation"
	"github.com/kyren223/eko/internal/client/ui/core/network"
	"github.com/kyren223/eko/internal/client/ui/core/networkcreation"
	"github.com/kyren223/eko/internal/client/ui/core/networklist"
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
	FocusNetwork
	FocusMax
)

type Model struct {
	name    string
	privKey ed25519.PrivateKey
	id      snowflake.ID

	loading   loadscreen.Model
	timer     timer.Model
	timeout   time.Duration
	connected bool

	networkCreationPopup   *networkcreation.Model
	frequencyCreationPopup *frequencycreation.Model
	networkList            networklist.Model
	network                network.Model
	focus                  int
}

func New(privKey ed25519.PrivateKey, name string) Model {
	m := Model{
		name:        name,
		privKey:     privKey,
		networkList: networklist.New(),
		loading:     loadscreen.New(connectingToServer),
		timer:       newTimer(initialTimeout),
		timeout:     initialTimeout,
		connected:   false,
		focus:       FocusNetworkList,
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
	network := m.network.View()
	result := lipgloss.JoinHorizontal(lipgloss.Top, networkList, network)

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
		m.id = snowflake.ID(msg)
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
	case gateway.ConnectionLost:
		m.connected = false
		m.timeout = initialTimeout
		return tea.Batch(gateway.Connect(m.privKey, connectionTimeout), m.loading.Init())

	case *packet.NetworksInfo:
		if msg.Set {
			state.State.Networks = msg.Networks
		} else {
			// state.State.Networks = append(state.State.Networks, msg.Networks...)
			networks := state.State.Networks
			networks = append(networks, msg.Networks...)
			networks = slices.DeleteFunc(networks, func(network packet.FullNetwork) bool {
				return slices.Contains(msg.RemoveNetworks, network.ID)
			})
			slices.SortFunc(networks, func(a, b packet.FullNetwork) int {
				return a.Position - b.Position
			})
			log.Println(networks)
			state.State.Networks = networks
		}

	case ui.QuitMsg:
		gateway.Disconnect()

	case tea.KeyMsg:
		key := msg.Type
		switch key {
		case tea.KeyCtrlN:
			switch m.focus {
			case FocusNetworkList:
				if m.networkCreationPopup == nil {
					popup := networkcreation.New()
					m.networkCreationPopup = &popup
				}
			case FocusNetwork:
				if m.frequencyCreationPopup == nil {
					index := m.networkList.Index()
					network := state.State.Networks[index]
					popup := frequencycreation.New(network.ID)
					m.frequencyCreationPopup = &popup
				}
			}

		case tea.KeyEscape:
			if m.networkCreationPopup != nil {
				m.networkCreationPopup = nil
			} else if m.frequencyCreationPopup != nil {
				m.frequencyCreationPopup = nil
			}

		case tea.KeyEnter:
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
			}

			if msg.String() == "L" && m.focus == FocusNetworkList {
				m.move(1)
			} else if msg.String() == "H" && m.focus == FocusNetwork {
				m.move(-1)
			}

		}
	}

	inPopup := m.networkCreationPopup != nil || m.frequencyCreationPopup != nil
	if inPopup {
		return nil
	}

	var cmd tea.Cmd
	switch m.focus {
	case FocusNetworkList:
		m.networkList, cmd = m.networkList.Update(msg)
		m.network.Set(m.networkList.Index())
	case FocusNetwork:
		m.network, cmd = m.network.Update(msg)
	}
	return cmd
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

	m.networkList.Blur()
	m.network.Blur()
	switch m.focus {
	case FocusNetworkList:
		m.networkList.Focus()
	case FocusNetwork:
		m.network.Focus()
	default:
		assert.Never("missing switch statement field in move", "focus", m.focus)
	}
}

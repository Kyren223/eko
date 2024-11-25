package core

import (
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/core/networkcreation"
	"github.com/kyren223/eko/internal/client/ui/core/networks"
	"github.com/kyren223/eko/internal/client/ui/loadscreen"
	"github.com/kyren223/eko/pkg/snowflake"
)

var (
	connectingToServer = "Connecting to server.."
	connectionFailed   = "Connection failed - retrying in %d sec..."
	connectionTimeout  = 5 * time.Second
	initialTimeout     = 3750 * time.Millisecond
	timerInterval      = 50 * time.Millisecond
)

type Model struct {
	name    string
	privKey ed25519.PrivateKey

	networks  networks.Model
	loading   loadscreen.Model
	timer     timer.Model
	timeout   time.Duration
	connected bool

	id snowflake.ID

	networkCreationPopup *networkcreation.Model
}

func New(privKey ed25519.PrivateKey, name string) Model {
	return Model{
		name:      name,
		privKey:   privKey,
		networks:  networks.New(),
		loading:   loadscreen.New(connectingToServer),
		timer:     newTimer(initialTimeout),
		timeout:   initialTimeout,
		connected: false,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(gateway.Connect(m.privKey, connectionTimeout), m.loading.Init())
}

func (m Model) View() string {
	if !m.connected {
		return m.loading.View()
	}

	result := m.networks.View()

	result = lipgloss.Place(
		ui.Width, ui.Height,
		lipgloss.Left, lipgloss.Top,
		result,
	)

	if m.networkCreationPopup != nil {
		popup := m.networkCreationPopup.View()
		x := (ui.Width - lipgloss.Width(popup)) / 2
		y := (ui.Height - lipgloss.Height(popup)) / 2
		result = ui.PlaceOverlay(x, y, popup, result)
	}

	return result
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.connected {
		switch msg := msg.(type) {
		case gateway.ConnectionEstablished:
			m.id = snowflake.ID(msg)
			m.connected = true
			m.timeout = initialTimeout
			return m, m.timer.Stop()

		case gateway.ConnectionFailed:
			m.timer = newTimer(m.timeout)
			m.updateLoadScreenContent()
			return m, m.timer.Start()

		case timer.TimeoutMsg:
			m.timeout = min(m.timeout*2, time.Minute)
			m.loading.SetContent(connectingToServer)
			return m, gateway.Connect(m.privKey, connectionTimeout)

		case timer.StartStopMsg:
			var cmd tea.Cmd
			m.timer, cmd = m.timer.Update(msg)
			return m, cmd

		case timer.TickMsg:
			m.updateLoadScreenContent()
			var cmd tea.Cmd
			m.timer, cmd = m.timer.Update(msg)
			return m, cmd

		case spinner.TickMsg:
			var loadscreenCmd tea.Cmd
			m.loading, loadscreenCmd = m.loading.Update(msg)
			return m, loadscreenCmd

		default:
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case gateway.ConnectionLost:
		m.connected = false
		m.timeout = initialTimeout
		return m, tea.Batch(gateway.Connect(m.privKey, connectionTimeout), m.loading.Init())

	case ui.QuitMsg:
		gateway.Disconnect()

	case tea.KeyMsg:
		key := msg.Type
		switch key {
		case tea.KeyCtrlN:
			if m.networkCreationPopup == nil {
				popup := networkcreation.New()
				m.networkCreationPopup = &popup
			}

		case tea.KeyEscape:
			if m.networkCreationPopup != nil {
				m.networkCreationPopup = nil
			}

		case tea.KeyEnter:
			if m.networkCreationPopup != nil {
				return m, m.networkCreationPopup.Select()
			}

		default:
			if m.networkCreationPopup != nil {
				popup, cmd := m.networkCreationPopup.Update(msg)
				m.networkCreationPopup = &popup
				return m, cmd
			}
		}
	}

	return m, nil
}

func (m *Model) updateLoadScreenContent() {
	seconds := m.timer.Timeout.Round(time.Second) / time.Second
	m.loading.SetContent(fmt.Sprintf(connectionFailed, seconds))
}

func newTimer(timeout time.Duration) timer.Model {
	return timer.NewWithInterval(timeout.Truncate(time.Second)+(time.Second/2), timerInterval)
}

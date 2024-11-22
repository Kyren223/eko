package core

import (
	"crypto/ed25519"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui/loadscreen"
)

var (
	connectingToServer = "Connecting to server.."
	connectionFailed   = "Connection failed - retrying in %d sec..."
)

type Model struct {
	privKey ed25519.PrivateKey
	name    string

	loading   loadscreen.Model
	connected bool
}

func New(privKey ed25519.PrivateKey, name string) Model {
	return Model{
		privKey:   privKey,
		name:      name,
		loading:   loadscreen.New(connectingToServer),
		connected: false,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(gateway.Connect(m.privKey, 5*time.Second), m.loading.Init())
}

func (m Model) View() string {
	if !m.connected {
		return m.loading.View()
	}

	return ""
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.connected {
		var loadscreenCmd tea.Cmd
		m.loading, loadscreenCmd = m.loading.Update(msg)
		return m, loadscreenCmd
	}
	return m, nil
}

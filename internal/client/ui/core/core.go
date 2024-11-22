package core

import (
	"crypto/ed25519"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct{}

func New(privKey ed25519.PrivateKey, name string) Model {

	// Connectiong to server...
	// Connection failed - retrying in 3 sec...
	// Connection failed - retrying in 3 sec...
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	return fmt.Sprintf(
		"%s",
		"",
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	default:
		return m, nil
	}
}

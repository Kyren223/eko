package chat

import (
	"log"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/viminput"
)

type Model struct {
	vi  viminput.Model
}

func New() Model {
	vi := viminput.New(30, 3)
	vi.Placeholder = "Send a message..."
	vi.PlaceholderStyle = lipgloss.NewStyle().Foreground(colors.Gray)
	vi.LineDecoration = func(lnum int, line string, cursorLnum int) string {
		return "┃ "
	}
	// vi.ShowLineNumbers = false

	// vi.CharLimit = 280

	// vi.Focus()

	log.Println("Placeholder:", vi.Placeholder)

	return Model{
		vi:  vi,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, nil)
}

func (m Model) View() string {
	return m.vi.View()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

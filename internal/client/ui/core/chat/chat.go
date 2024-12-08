package chat

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/viminput"
)

type Model struct {
	vi    viminput.Model
	focus bool
}

func New() Model {
	vi := viminput.New(90, 3)
	vi.Placeholder = "Send a message..."
	vi.PlaceholderStyle = lipgloss.NewStyle().Foreground(colors.Gray)
	vi.LineDecoration = func(lnum int, m viminput.Model) string {
		// lineNumberDecor := viminput.LineNumberDecoration(lipgloss.NewStyle())
		// lineNumber := lineNumberDecor(lnum, m)
		// return lineNumber + " ┃ "
		return "┃ "
	}

	vi.SetLines([]rune("test"), []rune("best"))
	// ta.Cursor.SetChar()

	// vi.CharLimit = 280

	// vi.Focus()

	return Model{
		vi: vi,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	return m.vi.View()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focus {
		return m, nil
	}
	var cmd tea.Cmd
	m.vi, cmd = m.vi.Update(msg)
	return m, cmd
}

func (m *Model) Focus() {
	m.focus = true
	m.vi.Focus()
}

func (m *Model) Blur() {
	m.focus = false
	m.vi.Blur()
}

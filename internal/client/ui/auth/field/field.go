package field

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kyren223/eko/internal/client/ui"
)

type Model struct {
	Input        textinput.Model
	Style        lipgloss.Style
	FocusedStyle lipgloss.Style
	BlurredStyle lipgloss.Style
	ErrorStyle   lipgloss.Style
	Header       string
}

func New(width int) Model {
	t := textinput.New()
	t.Width = width
	t.CharLimit = width
	t.Prompt = ""

	m := Model{
		Input: t,
	}

	m.Blur()

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	style := m.Style
	error := ""
	if m.Input.Err != nil {
		error = m.Input.Err.Error()
		style = style.BorderForeground(m.ErrorStyle.GetForeground())
	}
	field := ui.AddBorderHeader(m.Header, 1, style, m.Input.View())

	error = m.ErrorStyle.MaxWidth(lipgloss.Width(field)).Render(error)

	// return lipgloss.JoinVertical(lipgloss.Left, borderTop, field, error)
	return lipgloss.JoinVertical(lipgloss.Left, field, error)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

func (m *Model) Focus() tea.Cmd {
	m.Input.PromptStyle = m.FocusedStyle
	m.Input.TextStyle = m.FocusedStyle
	return m.Input.Focus()
}

func (m *Model) Blur() {
	m.Input.Blur()
	m.Input.PromptStyle = m.BlurredStyle
	m.Input.TextStyle = m.BlurredStyle
}

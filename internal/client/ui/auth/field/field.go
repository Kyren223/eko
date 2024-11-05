package field

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)


type Model struct {
	Input textinput.Model
	Style lipgloss.Style
	FocusedStyle lipgloss.Style
	BlurredStyle lipgloss.Style
	ErrorStyle lipgloss.Style
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
	var builder strings.Builder

	builder.WriteString(m.Style.Render(m.Input.View()))
	builder.WriteRune('\n')
	if m.Input.Err != nil {
		builder.WriteString(m.ErrorStyle.Render(m.Input.Err.Error()))
	}

	return builder.String()
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

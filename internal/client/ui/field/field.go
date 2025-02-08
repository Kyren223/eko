package field

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kyren223/eko/internal/client/ui"
)

type Model struct {
	Input    textinput.Model
	Header   string
	Visisble bool
	width    int

	reveal      bool
	revealIcon  string
	concealIcon string
	icon        string

	FocusedStyle     lipgloss.Style
	BlurredStyle     lipgloss.Style
	FocusedTextStyle lipgloss.Style
	BlurredTextStyle lipgloss.Style
	ErrorStyle       lipgloss.Style
	HeaderStyle      lipgloss.Style
}

func New(width int) Model {
	input := textinput.New()
	input.Prompt = ""
	input.Width = width

	m := Model{
		width:    width,
		Visisble: true,
		Input:    input,
	}

	m.Blur()
	m.SetRevealed(true)

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	if !m.Visisble {
		return ""
	}

	style := m.BlurredStyle
	if m.Input.Focused() {
		style = m.FocusedStyle
	}

	header := m.HeaderStyle.Render(m.Header)
	if m.Input.Err != nil {
		error := m.Input.Err.Error()
		style = style.BorderForeground(m.ErrorStyle.GetForeground())
		header = lipgloss.NewStyle().
			MaxWidth(m.width).
			Render(header + m.ErrorStyle.Render(" - "+error))
	}

	input := m.Input.View() + m.icon

	field := ui.AddBorderHeader(header, 1, style, input)
	return field
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.Visisble {
		return m, nil
	}

	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

func (m *Model) Focus() tea.Cmd {
	m.Input.PromptStyle = m.FocusedTextStyle
	m.Input.TextStyle = m.FocusedTextStyle
	m.Input.Cursor.Style = m.FocusedTextStyle
	m.Input.Cursor.TextStyle = m.FocusedTextStyle
	return m.Input.Focus()
}

func (m *Model) Blur() {
	m.Input.PromptStyle = m.BlurredTextStyle
	m.Input.TextStyle = m.BlurredTextStyle
	m.Input.Cursor.Style = m.BlurredTextStyle
	m.Input.Cursor.TextStyle = m.BlurredTextStyle
	m.Input.Blur()
}

func (m *Model) SetWidth(width int) {
	m.width = width
	m.recalculateWidth()
}

func (m Model) Width() int {
	return m.width
}

func (m *Model) SetRevealed(revealed bool) {
	m.reveal = revealed
	if m.reveal {
		m.Input.EchoMode = textinput.EchoNormal
		m.icon = m.revealIcon
	} else {
		m.Input.EchoMode = textinput.EchoPassword
		m.icon = m.concealIcon
	}
	m.recalculateWidth()
}

func (m Model) Revealed() bool {
	return m.reveal
}

func (m *Model) SetRevealIcon(icon string) {
	m.revealIcon = icon
	if m.reveal {
		m.icon = icon
	}
	m.recalculateWidth()
}

func (m Model) RevealIcon() string {
	return m.revealIcon
}

func (m *Model) SetConcealIcon(icon string) {
	m.concealIcon = icon
	if !m.reveal {
		m.icon = icon
	}
	m.recalculateWidth()
}

func (m Model) ConcealIcon() string {
	return m.concealIcon
}

func (m *Model) recalculateWidth() {
	m.Input.Width = m.width - lipgloss.Width(m.icon)
}

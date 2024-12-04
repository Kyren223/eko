package viminput

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type (
	Keymap         struct{}
	LineDecoration = func(lnum int, line string, cursorLnum int) string
)

type Line []byte

type Model struct {
	keymap Keymap

	PlaceholderStyle lipgloss.Style
	PromptStyle      lipgloss.Style

	Placeholder    string
	LineDecoration LineDecoration

	lines []Line

	width  int
	height int
	focus  bool
}

func New(width, height int) Model {
	return Model{
		keymap:           Keymap{},
		PlaceholderStyle: lipgloss.NewStyle(),
		PromptStyle:      lipgloss.NewStyle(),
		Placeholder:      "",
		LineDecoration:   EmptyLineDecoration,
		width:            width,
		height:           height,
		focus:            false,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	placeholder := m.PlaceholderStyle.Render(m.Placeholder)
	lineDecoration := m.LineDecoration(1, "", 1)

	result := lipgloss.JoinHorizontal(lipgloss.Top, lineDecoration, placeholder)

	result = lipgloss.NewStyle().Width(m.width).Height(m.height).Render(result)

	return result
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focus {
		return m, nil
	}

	return m, nil
}

func (m *Model) SetWidth(width int) {
	m.width = width
}

func (m Model) Width() int {
	return m.width
}

func (m *Model) SetHeight(height int) {
	m.height = height
}

func (m Model) Height() int {
	return m.height
}

func (m *Model) Focus() {
	m.focus = true
}

func (m *Model) Blur() {
	m.focus = false
}

package choicepopup

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kyren223/eko/pkg/assert"
)

type Model struct {
	Width  int
	Height int

	Dialogue viewport.Model
	Cycle    bool

	choices []string
	index   int

	SelectedStyle   lipgloss.Style
	UnselectedStyle lipgloss.Style
	ChoicesStyle    lipgloss.Style
	Style           lipgloss.Style
}

func New(width, height int) Model {
	return Model{
		Width:    width,
		Height:   height,
		Dialogue: viewport.New(0, 0),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	var choices []string
	for i, choice := range m.choices {
		if i == m.index {
			choices = append(choices, m.SelectedStyle.Render(choice))
		} else {
			choices = append(choices, m.UnselectedStyle.Render(choice))
		}
	}
	styledChoices := m.ChoicesStyle.MaxWidth(m.Width).MaxHeight(m.Height).
		Render(lipgloss.JoinHorizontal(lipgloss.Center, choices...))

	m.Dialogue.Width = m.Width
	m.Dialogue.Height = m.Height - lipgloss.Height(styledChoices)

	popup := lipgloss.JoinVertical(lipgloss.Left, m.Dialogue.View(), styledChoices)
	return m.Style.Render(popup)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.Dialogue, cmd = m.Dialogue.Update(msg)
	return m, cmd
}

func (m *Model) SetChoices(choices ...string) {
	assert.Assert(len(choices) != 0, "choices must have at least 1 element")
	m.choices = choices
	m.index = 0
}

func (m *Model) ScrollLeft() {
	if m.index == 0 {
		if m.Cycle {
			m.index = len(m.choices) - 1
		}
		return
	}
	m.index--
}

func (m *Model) ScrollRight() {
	if m.index == len(m.choices)-1 {
		if m.Cycle {
			m.index = 0
		}
		return
	}
	m.index++
}

func (m Model) Select() (index int, choice string) {
	return m.index, m.choices[m.index]
}

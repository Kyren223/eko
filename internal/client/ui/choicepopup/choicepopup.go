package choicepopup

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kyren223/eko/pkg/assert"
)

type Model struct {
	Width  int
	Height int
	Cycle  bool

	content   string
	choices   []string
	index     int
	leftCount int

	SelectedStyle   lipgloss.Style
	UnselectedStyle lipgloss.Style
	ChoicesStyle    lipgloss.Style
	Style           lipgloss.Style
}

func New(width, height int) Model {
	return Model{
		Width:  width,
		Height: height,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	var leftChoices []string
	var rightChoices []string
	for i, choice := range m.choices {
		if i == m.index {
			choice = m.SelectedStyle.Render(choice)
		} else {
			choice = m.UnselectedStyle.Render(choice)
		}

		if i < m.leftCount {
			leftChoices = append(leftChoices, choice)
		} else {
			rightChoices = append(rightChoices, choice)
		}
	}

	left := lipgloss.JoinHorizontal(lipgloss.Center, leftChoices...)
	right := lipgloss.JoinHorizontal(lipgloss.Center, rightChoices...)
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)

	paddingSize := m.Width - leftWidth - rightWidth
	assert.Assert(paddingSize >= 0, "there should be enough space for all choices", "remaining", paddingSize)
	padding := strings.Repeat(" ", paddingSize)
	choices := lipgloss.JoinHorizontal(lipgloss.Center, left, padding, right)
	styledChoices := m.ChoicesStyle.MaxWidth(m.Width).MaxHeight(m.Height).Render(choices)

	maxContentHeight := m.Height - lipgloss.Height(styledChoices)
	content := lipgloss.NewStyle().MaxWidth(m.Width).MaxHeight(maxContentHeight).
		Render(m.content)

	popup := lipgloss.JoinVertical(lipgloss.Left, content, styledChoices)
	return m.Style.Render(popup)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m *Model) SetContent(content string) {
	m.content = content
}

func (m *Model) SetChoices(leftChoices, rightChoices []string) {
	assert.Assert(len(leftChoices)+len(rightChoices) != 0, "choices must have at least 1 element")
	m.index = 0
	m.leftCount = len(leftChoices)
	m.choices = nil
	m.choices = append(m.choices, leftChoices...)
	m.choices = append(m.choices, rightChoices...)
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

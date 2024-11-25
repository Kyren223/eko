package flex

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	Style     lipgloss.Style
	ItemStyle lipgloss.Style

	contents  []string

	Position  lipgloss.Position
	Gap       int
	Vertical  bool
}

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	contents := make([]string, len(m.contents))
	for i, content := range contents {
		content = m.ItemStyle.Render(content)
		if i != 0 {
			if m.Vertical {
				contents[i] = lipgloss.NewStyle().PaddingTop(m.Gap).Render(content)
			} else {
				contents[i] = lipgloss.NewStyle().PaddingLeft(m.Gap).Render(content)
			}
		}
	}

	var result string
	if m.Vertical {
		result = lipgloss.JoinVertical(m.Position, contents...)
	} else {
		result = lipgloss.JoinHorizontal(m.Position, contents...)
	}

	return m.Style.Render(result)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m *Model) SetContents(contents []string) {
	m.contents = contents
}

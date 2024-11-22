package networks

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	style        = lipgloss.NewStyle().Border(lipgloss.ThickBorder(), true, false)
	networkStyle = lipgloss.NewStyle().Width(6).Height(3).PaddingTop(1).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("#ffbf00")).Background(lipgloss.Color("#4a3d5c")).
			Border(lipgloss.ThickBorder(), false, true)
	networks = []string{
		"󰜈 ",
		" ",
		"Kr",
		" ",
		// "Test",
		// "██████\n██████\n██████",
		// " ▄▄▄▄ \n █  █ \n ▀▀▀▀ ",
		// "      \n  󰜈  \n     ",
		// "Hmm",
		// "Another",
	}
)

type Model struct {
	networks []string
}

func New() Model {
	return Model{
		networks: networks,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	border := lipgloss.ThickBorder()
	var builder strings.Builder

	top := strings.Repeat(border.Top, 6)
	builder.WriteString(fmt.Sprintf("%s%s%s\n", border.TopLeft, top, border.TopRight))
	for i, network := range m.networks {
		if i != 0 {
			middle := strings.Repeat(border.Top, 6)
			builder.WriteString(fmt.Sprintf("\n%s%s%s\n", border.MiddleLeft, middle, border.MiddleRight))
		}
		builder.WriteString(networkStyle.Render(network))
	}
	bottom := strings.Repeat(border.Bottom, 6)
	builder.WriteString(fmt.Sprintf("\n%s%s%s", border.BottomLeft, bottom, border.BottomRight))

	return builder.String()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

package networklist

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
)

var (
	style            = lipgloss.NewStyle().Border(lipgloss.ThickBorder(), true, false)
	partialIconStyle = lipgloss.NewStyle().Width(6).Height(3).PaddingTop(1).Margin(0, 1).
				Align(lipgloss.Center).
				Border(lipgloss.ThickBorder(), false, false)
	trustedUsersButton = IconStyle(lipgloss.Color(colors.Turquoise), lipgloss.Color(colors.DarkerCyan)).
			MarginBottom(1).Render(" ")
)

func IconStyle(fg, bg lipgloss.Color) lipgloss.Style {
	return partialIconStyle.Foreground(fg).Background(bg)
}

type Model struct{
	focus bool
}

func New() Model {
	return Model{
		focus: false,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	// border := lipgloss.ThickBorder()
	var builder strings.Builder

	// top := strings.Repeat(border.Top, 6)
	// builder.WriteString(fmt.Sprintf("%s%s%s\n", border.TopLeft, top, border.TopRight))
	builder.WriteString("\n")
	builder.WriteString(trustedUsersButton)
	builder.WriteString("\n")
	for _, network := range state.State.Networks {
		builder.WriteString(IconStyle(lipgloss.Color(network.FgHexColor), lipgloss.Color(network.BgHexColor)).
			MarginBottom(1).Render(network.Icon))
		builder.WriteString("\n")
	}
	// bottom := strings.Repeat(border.Bottom, 6)
	// builder.WriteString(fmt.Sprintf("\n%s%s%s", border.BottomLeft, bottom, border.BottomRight))

	result := builder.String()

	sep := lipgloss.NewStyle().Border(lipgloss.ThickBorder(), false, true, false, false).Height(ui.Height).Width(0)
	if m.focus {
		sep = sep.BorderForeground(colors.Focus)
	}
	result = lipgloss.JoinHorizontal(lipgloss.Top, result, sep.Render(""))

	return result
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m *Model) Focus() {
	m.focus = true
}

func (m *Model) Blur() {
	m.focus = false
}

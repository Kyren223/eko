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
	partialIconStyle = lipgloss.NewStyle().
				Width(6).Height(3).
				Align(lipgloss.Center).
				Border(lipgloss.ThickBorder(), false, false)

	selectedIndicator          = "|\n|\n|\n"
	trustedUsersIcon           = IconStyle(" ", lipgloss.Color(colors.Turquoise), lipgloss.Color(colors.DarkerCyan))
	trustedUsersButton         = trustedUsersIcon.Margin(0, 1, 1).String()
	trustedUsersButtonSelected = lipgloss.JoinHorizontal(ui.Center, selectedIndicator, trustedUsersIcon.Margin(0, 1, 1, 0).String())
)

func IconStyle(icon string, fg, bg lipgloss.Color) lipgloss.Style {
	return partialIconStyle.Foreground(fg).Background(bg).SetString("\n" + icon)
}

type Model struct {
	focus bool
	index int
}

func New() Model {
	return Model{
		focus: false,
		index: 0,
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
	if m.index == 0 {
		builder.WriteString(trustedUsersButtonSelected)
	} else {
		builder.WriteString(trustedUsersButton)
	}
	builder.WriteString("\n")
	for i, network := range state.State.Networks {
		icon := IconStyle(network.Icon, lipgloss.Color(network.FgHexColor), lipgloss.Color(network.BgHexColor))
		if m.index == i+1 {
			builder.WriteString(lipgloss.JoinHorizontal(ui.Center, selectedIndicator, icon.Margin(0, 1, 1, 0).String()))
		} else {
			builder.WriteString(icon.Margin(0, 1, 1).String())
		}
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "k":
			m.index = max(0, m.index-1)
		case "j":
			m.index = min(len(state.State.Networks), m.index+1)
		}
	}
	return m, nil
}

func (m *Model) Focus() {
	m.focus = true
}

func (m *Model) Blur() {
	m.focus = false
}

package banview

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/client/ui/layouts/flex"
	"github.com/kyren223/eko/pkg/snowflake"
)

var width = 48

const (
	BanReasonField = iota
	BanField
	FieldCount
)

type Model struct {
	banReason string
	name      string
	id        string
}

func New(userId, networkId snowflake.ID) Model {
	return Model{
		banReason: *state.State.Members[networkId][userId].BanReason,
		name:      state.State.Users[userId].Name,
		id:        strconv.FormatInt(int64(userId), 10),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	headerStyle := lipgloss.NewStyle().
		Width(width).
		Background(colors.Background).
		Foreground(colors.Focus)
	grayStyle := lipgloss.NewStyle().
		Width(width).
		Background(colors.Background).
		Foreground(colors.LightGray)

	nameHeader := headerStyle.Render("Banned Username:")
	name := m.name + grayStyle.Render(" ("+m.id+")")
	name = lipgloss.NewStyle().
		Width(width).
		Background(colors.Background).
		Render(name)

	banReasonHeader := headerStyle.Render("Ban Reason:")
	banReason := lipgloss.NewStyle().Width(width).Render(m.banReason)

	content := flex.NewVertical(nameHeader, name, banReasonHeader, banReason).WithGap(1).View()

	return lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		Padding(1, 4).
		Align(lipgloss.Center, lipgloss.Center).
		BorderBackground(colors.Background).
		BorderForeground(colors.White).
		Background(colors.Background).
		Foreground(colors.White).
		Render(content)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

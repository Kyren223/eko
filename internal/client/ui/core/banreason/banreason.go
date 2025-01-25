package banreason

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/client/ui/field"
	"github.com/kyren223/eko/internal/client/ui/layouts/flex"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

var (
	width = 48

	style = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		Padding(1, 4).
		Align(lipgloss.Center, lipgloss.Center)

	headerStyle = lipgloss.NewStyle().Foreground(colors.Turquoise)

	fieldBlurredStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colors.DarkCyan)
	fieldFocusedStyle = fieldBlurredStyle.
				BorderForeground(colors.Focus).
				Border(lipgloss.ThickBorder())

	blurredBanStyle = lipgloss.NewStyle().
			Background(colors.Gray).Padding(0, 1)
	focusedBanStyle = lipgloss.NewStyle().
			Background(colors.Blue).Padding(0, 1)
)

const (
	BanReasonField = iota
	BanField
	FieldCount
)

type Model struct {
	networkId snowflake.ID
	userId    snowflake.ID
	banReason field.Model
	banStyle  lipgloss.Style

	selected  int
	nameWidth int
}

func New(userId, networkId snowflake.ID) Model {
	banReason := field.New(width)
	banReason.Header = "Ban Reason"
	banReason.HeaderStyle = headerStyle
	banReason.FocusedStyle = fieldFocusedStyle
	banReason.BlurredStyle = fieldBlurredStyle
	banReason.Input.CharLimit = packet.MaxBanReasonBytes
	banReason.Focus()
	nameWidth := lipgloss.Width(banReason.View())

	return Model{
		networkId: networkId,
		userId:    userId,
		banReason: banReason,
		banStyle:  blurredBanStyle,
		selected:  0,
		nameWidth: nameWidth,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	name := m.banReason.View()
	ban := lipgloss.NewStyle().
		Width(m.nameWidth).Align(lipgloss.Center).
		Render(m.banStyle.Render("Ban", state.State.Users[m.userId].Name))

	content := flex.NewVertical(name, ban).WithGap(1).View()
	return style.Render(content)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.Type
		switch key {
		case tea.KeyTab:
			return m, m.cycle(1)
		case tea.KeyShiftTab:
			return m, m.cycle(-1)

		default:
			var cmd tea.Cmd
			switch m.selected {
			case BanReasonField:
				m.banReason, cmd = m.banReason.Update(msg)
			}
			return m, cmd
		}
	}

	return m, nil
}

func (m *Model) cycle(step int) tea.Cmd {
	m.selected += step
	if m.selected < 0 {
		m.selected = FieldCount - 1
	} else {
		m.selected %= FieldCount
	}
	return m.updateFocus()
}

func (m *Model) updateFocus() tea.Cmd {
	m.banReason.Blur()
	m.banStyle = blurredBanStyle
	switch m.selected {
	case BanReasonField:
		return m.banReason.Focus()
	case BanField:
		m.banStyle = focusedBanStyle
		return nil
	default:
		assert.Never("missing switch statement field in update focus", "selected", m.selected)
		return nil
	}
}

func (m *Model) Select() tea.Cmd {
	if m.selected != BanField {
		return nil
	}

	banReason := m.banReason.Input.Value()

	yes := true
	return gateway.Send(&packet.SetMember{
		Member:    nil,
		Admin:     nil,
		Muted:     nil,
		Banned:    &yes,
		BanReason: &banReason,
		Network:   m.networkId,
		User:      m.userId,
	})
}

package networklist

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/packet"
)

var (
	sepStyle = lipgloss.NewStyle().Width(0).
			Border(lipgloss.ThickBorder(), false, true, false, false)
	partialIconStyle = lipgloss.NewStyle().
				Width(6).Height(3).
				Align(lipgloss.Center).
				Border(lipgloss.ThickBorder(), false, false)

	selectedIndicator          = "🭀\n▌\n🭛"
	trustedUsersIcon           = IconStyle(" ", colors.Turquoise, colors.DarkerCyan)
	trustedUsersButton         = trustedUsersIcon.Margin(0, 1, 1).String()
	trustedUsersButtonSelected = lipgloss.JoinHorizontal(
		ui.Center,
		selectedIndicator,
		trustedUsersIcon.Margin(0, 1, 1, 0).String(),
	)
)

/*
🭊🭂██🭍🬿
██████
🭥🭓██🭞🭚

🭠🭘  🭣🭕

🭏🬽  🭈🭄
*/

func IconStyle(icon string, fg, bg lipgloss.Color) lipgloss.Style {
	if true {
		bgStyle := lipgloss.NewStyle().Background(bg).Foreground(colors.Background)
		top := bgStyle.Render("🭠🭘  🭣🭕")
		middle := lipgloss.NewStyle().Width(6).Align(lipgloss.Center).
			Background(bg).Foreground(fg).Render(icon)
		bgStyle2 := lipgloss.NewStyle().Foreground(bg)
		bottom := bgStyle2.Render("🭥🭓██🭞🭚")
		combined := lipgloss.JoinVertical(lipgloss.Left, top, middle, bottom)
		return lipgloss.NewStyle().SetString(combined)
	}
	if true {
		bgStyle := lipgloss.NewStyle().Background(bg).Foreground(colors.Background)
		top := bgStyle.Render("🭠🭘  🭣🭕")
		middle := lipgloss.NewStyle().Width(6).Align(lipgloss.Center).
			Background(bg).Foreground(fg).Render(icon)
		bottom := bgStyle.Render("🭏🬽  🭈🭄")
		combined := lipgloss.JoinVertical(lipgloss.Left, top, middle, bottom)
		return lipgloss.NewStyle().SetString(combined)
	}
	if true {
		return partialIconStyle.Foreground(fg).SetString("\n" + icon)
	}
	if true {
		bgStyle := lipgloss.NewStyle().Foreground(bg)
		top := bgStyle.Render("🭊🭂██🭍🬿")
		middle := lipgloss.NewStyle().Width(6).Align(lipgloss.Center).
			Background(bg).Foreground(fg).Render(icon)
		bottom := bgStyle.Render("🭥🭓██🭞🭚")
		combined := lipgloss.JoinVertical(lipgloss.Left, top, middle, bottom)
		return lipgloss.NewStyle().SetString(combined)
	}
	return partialIconStyle.Foreground(fg).Background(bg).SetString("\n" + icon)
}

type Model struct {
	history []func(m *Model)
	index   int
	focus   bool
}

func New() Model {
	return Model{
		focus: false,
		index: -1,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	var builder strings.Builder
	builder.WriteString("\n")
	if m.index == -1 {
		builder.WriteString(trustedUsersButtonSelected)
	} else {
		builder.WriteString(trustedUsersButton)
	}
	builder.WriteString("\n")
	for i, network := range state.State.Networks {
		icon := IconStyle(network.Icon,
			lipgloss.Color(network.FgHexColor),
			lipgloss.Color(network.BgHexColor),
		)
		if m.index == i {
			builder.WriteString(lipgloss.JoinHorizontal(
				ui.Center,
				selectedIndicator,
				icon.Margin(0, 1, 1, 0).String(),
			))
		} else {
			builder.WriteString(icon.Margin(0, 1, 1).String())
		}
		builder.WriteString("\n")
	}

	result := builder.String()

	sep := sepStyle.Height(ui.Height)
	if m.focus {
		sep = sep.BorderForeground(colors.Focus)
	}
	result = lipgloss.JoinHorizontal(lipgloss.Top, result, sep.String())

	return result
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *packet.SwapUserNetworks:
		// Pop first by shifting to the left
		copy(m.history, m.history[1:])
		m.history = m.history[:len(m.history)-1]
	case *packet.Error:
		if msg.PktType == packet.PacketSwapUserNetworks {
			// Server failed, revert!
			undo := m.history[len(m.history)-1]
			m.history = m.history[:len(m.history)-1]
			undo(&m)
		}

	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "K":
			if 0 < m.index {
				return m.Swap(-1)
			}
		case "J":
			if 0 <= m.index && m.index < len(state.State.Networks)-1 {
				return m.Swap(1)
			}
		case "k":
			m.index = max(-1, m.index-1)
		case "j":
			m.index = min(len(state.State.Networks)-1, m.index+1)
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

func (m Model) Swap(dir int) (Model, tea.Cmd) {
	cmd := gateway.Send(&packet.SwapUserNetworks{
		Pos1: m.index,
		Pos2: m.index + dir,
	})
	tmp := state.State.Networks[m.index]
	state.State.Networks[m.index] = state.State.Networks[m.index+dir]
	state.State.Networks[m.index+dir] = tmp
	m.index += dir
	m.history = append(m.history, func(m *Model) {
		m.index -= dir
		tmp := state.State.Networks[m.index]
		state.State.Networks[m.index] = state.State.Networks[m.index+dir]
		state.State.Networks[m.index+dir] = tmp
	})
	return m, cmd
}

func (m Model) Index() int {
	return m.index
}

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

	selectedIndicator   = "🭀\n▌\n🭛"
	trustedUsersIcon    = IconStyle(" ", colors.Turquoise, colors.DarkerCyan)
	peersButton         = trustedUsersIcon.Margin(0, 1, 1).String()
	peersButtonSelected = lipgloss.JoinHorizontal(
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
	bgStyle := lipgloss.NewStyle().Background(bg).Foreground(colors.Background)
	top := bgStyle.Render("🭠🭘  🭣🭕")
	middle := lipgloss.NewStyle().Width(6).Align(lipgloss.Center).
		Background(bg).Foreground(fg).Render(icon)
	bgStyle2 := lipgloss.NewStyle().Foreground(bg)
	bottom := bgStyle2.Render("🭥🭓██🭞🭚")
	combined := lipgloss.JoinVertical(lipgloss.Left, top, middle, bottom)
	return lipgloss.NewStyle().SetString(combined)
}

const PeersIndex = -1

type Model struct {
	index int
	focus bool
}

func New() Model {
	return Model{
		focus: false,
		index: PeersIndex,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	var builder strings.Builder
	builder.WriteString("\n")
	if m.index == PeersIndex {
		builder.WriteString(peersButtonSelected)
	} else {
		builder.WriteString(peersButton)
	}
	builder.WriteString("\n")
	for i, networkId := range state.Data.Networks {
		network := state.State.Networks[networkId]

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
	case tea.KeyMsg:
		if !m.focus {
			return m, nil
		}

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

		case "Q":
			if state.UserID == nil || m.index == PeersIndex {
				return m, nil
			}
			no := false
			return m, gateway.Send(&packet.SetMember{
				Member:    &no,
				Admin:     nil,
				Muted:     nil,
				Banned:    nil,
				BanReason: nil,
				Network:   *state.NetworkId(m.index),
				User:      *state.UserID,
			})

		case "D":
			if state.UserID == nil || m.index == PeersIndex {
				return m, nil
			}
			return m, gateway.Send(&packet.DeleteNetwork{
				Network: *state.NetworkId(m.index),
			})

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
	tmp := state.Data.Networks[m.index]
	state.Data.Networks[m.index] = state.Data.Networks[m.index+dir]
	state.Data.Networks[m.index+dir] = tmp
	m.index += dir
	return m, gateway.Send(&packet.SetUserData{
		Data: state.JsonUserData(),
	})
}

func (m Model) Index() int {
	return m.index
}

func (m *Model) SetIndex(index int) {
	m.index = index
}

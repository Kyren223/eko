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

const SignalsIndex = -1

type Model struct {
	base   int
	index  int
	height int
	focus  bool
}

func New() Model {
	return Model{
		base:   0,
		index:  SignalsIndex,
		height: 1,
		focus:  false,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	selectedIndicator := lipgloss.NewStyle().
		Foreground(colors.White).Render("🭀\n▌\n🭛")

	notification := lipgloss.NewStyle().
		Foreground(colors.White).Render(" \n◗\n ")

	var builder strings.Builder

	pings := 0
	for _, signal := range state.Data.Signals {
		pings += state.State.Notifications[signal]
	}
	var signalsIcon lipgloss.Style
	if pings == 0 {
		signalsIcon = ui.IconStyle(" ", colors.Turquoise, colors.DarkerCyan, colors.BackgroundDimmer)
	} else {
		signalsIcon = ui.IconStyleNotif(" ", colors.Turquoise, colors.DarkerCyan, colors.BackgroundDimmer, pings)
	}

	builder.WriteString("\n")
	if m.index == SignalsIndex {
		signalsButtonSelected := lipgloss.JoinHorizontal(
			ui.Center,
			selectedIndicator,
			signalsIcon.Background(colors.BackgroundDimmer).
				Padding(0, 1, 1, 0).String(),
		)
		builder.WriteString(signalsButtonSelected)
	} else {
		signalsButtonStyle := signalsIcon.
			Background(colors.BackgroundDimmer).Padding(0, 1, 1).String()
		builder.WriteString(signalsButtonStyle)
	}
	builder.WriteString("\n")

	networks := state.Data.Networks
	upper := min(m.base+m.height, len(networks))
	networks = networks[m.base:upper]
	for i, networkId := range networks {
		network := state.State.Networks[networkId]

		pings, ok := 0, false
		frequencies := state.State.Frequencies[networkId]
		for _, frequency := range frequencies {
			fpings, fok := state.State.Notifications[frequency.ID]
			pings += fpings
			ok = ok || fok
		}

		var icon lipgloss.Style
		if ok && pings != 0 {
			icon = ui.IconStyleNotif(network.Icon,
				lipgloss.Color(network.FgHexColor),
				lipgloss.Color(network.BgHexColor),
				colors.BackgroundDimmer, pings,
			)
		} else {
			icon = ui.IconStyle(network.Icon,
				lipgloss.Color(network.FgHexColor),
				lipgloss.Color(network.BgHexColor),
				colors.BackgroundDimmer,
			)
		}

		if m.index == m.base+i {
			builder.WriteString(lipgloss.JoinHorizontal(
				ui.Center,
				selectedIndicator,
				icon.Background(colors.BackgroundDimmer).Padding(0, 1, 1, 0).String(),
			))
		} else if ok {
			builder.WriteString(lipgloss.JoinHorizontal(
				ui.Center,
				notification,
				icon.Background(colors.BackgroundDimmer).Padding(0, 1, 1, 0).String(),
			))
		} else {
			builder.WriteString(icon.Background(colors.BackgroundDimmer).Padding(0, 1, 1).String())
		}
		builder.WriteString("\n")
	}

	result := builder.String()

	sep := lipgloss.NewStyle().Width(0).Height(ui.Height).
		BorderBackground(colors.BackgroundDimmer).BorderForeground(colors.White).
		Border(lipgloss.ThickBorder(), false, true, false, false)
	if m.focus {
		sep = sep.BorderForeground(colors.Focus)
	}

	result = lipgloss.JoinHorizontal(lipgloss.Top, result, sep.String())

	return lipgloss.NewStyle().Background(colors.BackgroundDimmer).Render(result)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focus {
		return m, nil
	}

	m.height = ui.Height
	m.height -= 1 // Inital top margin
	m.height /= 4 // 4 per icon
	m.height -= 1 // For signals icon
	m.SetIndex(m.index)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "K":
			if 0 < m.index {
				m.Swap(-1)
			}
		case "J":
			if 0 <= m.index && m.index < len(state.State.Networks)-1 {
				m.Swap(1)
			}
		case "k":
			m.SetIndex(m.index - 1)
		case "j":
			m.SetIndex(m.index + 1)

		case "Q":
			if state.UserID == nil || m.index == SignalsIndex {
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
			if state.UserID == nil || m.index == SignalsIndex {
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

func (m *Model) Swap(dir int) {
	tmp := state.Data.Networks[m.index]
	state.Data.Networks[m.index] = state.Data.Networks[m.index+dir]
	state.Data.Networks[m.index+dir] = tmp
	m.SetIndex(m.index + dir)
}

func (m Model) Index() int {
	return m.index
}

func (m *Model) SetIndex(index int) {
	m.index = min(max(index, SignalsIndex), len(state.State.Networks)-1)
	if m.index < m.base && m.index != SignalsIndex {
		m.base = m.index
	} else if m.index >= m.base+m.height {
		m.base = 1 + m.index - m.height
	}
}

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
	sepStyle = lipgloss.NewStyle().Width(0).BorderBackground(colors.BackgroundDimmer).
			Border(lipgloss.ThickBorder(), false, true, false, false)

	selectedIndicator   = "🭀\n▌\n🭛"
	peersIcon           = ui.IconStyle(" ", colors.Turquoise, colors.DarkerCyan, colors.BackgroundDimmer)
	peersButton         = peersIcon.Background(colors.BackgroundDimmer).Padding(0, 1, 1).String()
	peersButtonSelected = lipgloss.JoinHorizontal(
		ui.Center,
		selectedIndicator,
		peersIcon.Background(colors.BackgroundDimmer).Padding(0, 1, 1, 0).String(),
	)

	backgroundStyle = lipgloss.NewStyle().Background(colors.BackgroundDimmer)
)

const PeersIndex = -1

type Model struct {
	base   int
	index  int
	height int
	focus  bool
}

func New() Model {
	return Model{
		base:   0,
		index:  PeersIndex,
		height: 1,
		focus:  false,
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

	networks := state.Data.Networks
	upper := min(m.base+m.height, len(networks))
	networks = networks[m.base:upper]
	for i, networkId := range networks {
		network := state.State.Networks[networkId]

		icon := ui.IconStyle(network.Icon,
			lipgloss.Color(network.FgHexColor),
			lipgloss.Color(network.BgHexColor),
			colors.BackgroundDimmer,
		)
		if m.index == m.base+i {
			builder.WriteString(lipgloss.JoinHorizontal(
				ui.Center,
				selectedIndicator,
				icon.Background(colors.BackgroundDimmer).Padding(0, 1, 1, 0).String(),
			))
		} else {
			builder.WriteString(icon.Background(colors.BackgroundDimmer).Padding(0, 1, 1).String())
		}
		builder.WriteString("\n")
	}

	result := builder.String()

	sep := sepStyle.Height(ui.Height)
	if m.focus {
		sep = sep.BorderForeground(colors.Focus)
	}
	result = lipgloss.JoinHorizontal(lipgloss.Top, result, sep.String())

	return backgroundStyle.Render(result)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focus {
		return m, nil
	}

	m.height = ui.Height
	m.height -= 1 // Inital top margin
	m.height /= 4 // 4 per icon
	m.height -= 1 // For peers icon
	m.SetIndex(m.index)

	switch msg := msg.(type) {
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
			m.SetIndex(m.index - 1)
		case "j":
			m.SetIndex(m.index + 1)

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

func (m *Model) Swap(dir int) (Model, tea.Cmd) {
	tmp := state.Data.Networks[m.index]
	state.Data.Networks[m.index] = state.Data.Networks[m.index+dir]
	state.Data.Networks[m.index+dir] = tmp
	m.SetIndex(m.index + dir)

	data := state.JsonUserData()
	return *m, gateway.Send(&packet.SetUserData{
		Data: &data,
		User: nil,
	})
}

func (m Model) Index() int {
	return m.index
}

func (m *Model) SetIndex(index int) {
	m.index = min(max(index, PeersIndex), len(state.State.Networks)-1)
	if m.index < m.base && m.index != PeersIndex {
		m.base = m.index
	} else if m.index >= m.base+m.height {
		m.base = 1 + m.index - m.height
	}
}

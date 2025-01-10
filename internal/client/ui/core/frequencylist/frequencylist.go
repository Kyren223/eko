package frequencylist

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
)

var (
	width    = 24
	sepStyle = lipgloss.NewStyle().Width(0).
			Border(lipgloss.ThickBorder(), false, true, false, false)
	nameStyle = lipgloss.NewStyle().
			Margin(0, 0, 1).Padding(1).Width(width).Align(lipgloss.Center).
			Border(lipgloss.ThickBorder(), false, false, true)
	margin         = 2
	frequencyStyle = lipgloss.NewStyle().
			Margin(0, margin).Padding(0, 1).Width(width - (margin * 2)).
			Align(lipgloss.Left)
	symbol = "# "
)

type Model struct {
	history      []func(m *Model)
	networkIndex int
	index        int
	focus        bool
}

func New() Model {
	return Model{
		history:      []func(m *Model){},
		networkIndex: -1,
		index:        -1,
		focus:        false,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	if state.NetworkId(m.networkIndex) == nil {
		return ""
	}

	var builder strings.Builder

	bg := lipgloss.Color(m.Network().BgHexColor)
	fg := lipgloss.Color(m.Network().FgHexColor)
	nameStyle := nameStyle.Background(bg).Foreground(fg)
	if m.focus {
		nameStyle = nameStyle.BorderForeground(colors.Focus)
	}
	builder.WriteString(nameStyle.Render(m.Network().Name))

	builder.WriteString("\n")
	for i, frequency := range m.Frequencies() {
		frequencyStyle := frequencyStyle.Foreground(lipgloss.Color(frequency.HexColor))
		if m.index == i {
			frequencyStyle = frequencyStyle.Background(colors.BackgroundHighlight)
		}

		frequencyName := lipgloss.NewStyle().
			MaxWidth(width - (margin * 2) - 4).
			Render(frequency.Name)
		builder.WriteString(frequencyStyle.Render(symbol + frequencyName))
		builder.WriteString("\n")
	}

	sidebar := builder.String()
	sep := sepStyle.Height(ui.Height)
	if m.focus {
		sep = sep.BorderForeground(colors.Focus)
	}
	result := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, sep.String())

	return result
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *packet.SwapFrequencies:
		// Pop first by shifting to the left
		copy(m.history, m.history[1:])
		m.history = m.history[:len(m.history)-1]
	case *packet.Error:
		if msg.PktType == packet.PacketSwapFrequencies {
			// Server failed, revert!
			undo := m.history[len(m.history)-1]
			m.history = m.history[:len(m.history)-1]
			undo(&m)
		}

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
			if m.index < m.FrequenciesLength()-1 {
				return m.Swap(1)
			}
		case "k":
			m.index = max(0, m.index-1)
		case "j":
			m.index = min(m.FrequenciesLength()-1, m.index+1)

		case "x":
			frequenciesCount := len(m.Frequencies())
			if frequenciesCount == 1 {
				// Don't delete the last frequency!
				return m, nil
			}
			// TODO: consider adding a confirmation popup
			frequencyId := m.Frequencies()[m.index].ID
			if m.index == frequenciesCount-1 {
				m.index--
			}
			return m, gateway.Send(&packet.DeleteFrequency{
				Frequency: frequencyId,
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
	cmd := gateway.Send(&packet.SwapFrequencies{
		Network: m.Network().ID,
		Pos1:    m.index,
		Pos2:    m.index + dir,
	})
	tmp := m.Frequencies()[m.index]
	m.Frequencies()[m.index] = m.Frequencies()[m.index+dir]
	m.Frequencies()[m.index+dir] = tmp
	m.index += dir
	m.history = append(m.history, func(m *Model) {
		m.index -= dir
		tmp := m.Frequencies()[m.index]
		m.Frequencies()[m.index] = m.Frequencies()[m.index+dir]
		m.Frequencies()[m.index+dir] = tmp
	})
	return m, cmd
}

func (m *Model) SetNetworkIndex(networkIndex int) {
	if m.networkIndex == networkIndex {
		return
	}

	networkId := state.NetworkId(m.networkIndex)
	if networkId != nil {
		frequencies := state.State.Frequencies[*networkId]
		if 0 <= m.index && m.index < len(frequencies) {
			frequencyId := frequencies[m.index].ID
			state.State.LastFrequency[*networkId] = frequencyId
		}
	}

	if networkIndex == -1 {
		m.networkIndex = -1
		m.index = -1
		return
	}

	m.networkIndex = networkIndex
	m.index = 0

	// Try restoring last ID
	networkId = state.NetworkId(m.networkIndex)
	if networkId == nil {
		return
	}
	id, ok := state.State.LastFrequency[*networkId]
	if !ok {
		return
	}
	frequencies := state.State.Frequencies[*networkId]
	for i, frequency := range frequencies {
		if frequency.ID == id {
			m.index = i
			break
		}
	}
}

func (m Model) FrequenciesLength() int {
	networkId := state.NetworkId(m.networkIndex)
	if networkId == nil {
		return 0
	}
	return len(state.State.Frequencies[*networkId])
}

func (m Model) Network() *data.Network {
	networkId := state.NetworkId(m.networkIndex)
	if networkId == nil {
		return nil
	}
	network := state.State.Networks[*networkId]
	return &network
}

func (m Model) Frequencies() []data.Frequency {
	networkId := state.NetworkId(m.networkIndex)
	if networkId == nil {
		return nil
	}
	return state.State.Frequencies[*networkId]
}

func (m *Model) Index() int {
	return m.index
}

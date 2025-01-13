package frequencylist

import (
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
)

var (
	width    = 24
	sepStyle = lipgloss.NewStyle().Width(0).
			Border(lipgloss.ThickBorder(), false, true, false, false)
	nameStyle = lipgloss.NewStyle().
			Margin(0, 0, 1).Padding(1).Width(width).Align(lipgloss.Center).
			Border(lipgloss.ThickBorder(), false, false, true)
	xMargin        = 2
	frequencyStyle = lipgloss.NewStyle().
			Margin(0, xMargin).Padding(0, 1).Width(width - (xMargin * 2)).
			Align(lipgloss.Left)
	symbol = "# "
)

type Model struct {
	networkIndex int
	base         int
	index        int
	focus        bool
	height       int
}

func New() Model {
	return Model{
		networkIndex: -1,
		base:         -1,
		index:        -1,
		focus:        false,
		height:       1,
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

	builder.WriteString(m.renderNetworkName())

	builder.WriteString("\n")
	frequencies := m.Frequencies()
	upper := min(m.base+m.height, len(frequencies))
	frequencies = frequencies[m.base:upper]
	for i, frequency := range frequencies {
		frequencyStyle := frequencyStyle.Foreground(lipgloss.Color(frequency.HexColor))
		if m.index == m.base+i {
			frequencyStyle = frequencyStyle.Background(colors.BackgroundHighlight)
		}

		frequencyName := lipgloss.NewStyle().
			MaxWidth(width - (xMargin * 2) - 4).
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
	if state.NetworkId(m.networkIndex) != nil {
		// Calculate height for frequencies
		m.height = ui.Height
		m.height -= lipgloss.Height(m.renderNetworkName())
		m.height -= 1
	}

	switch msg := msg.(type) {
	case *packet.SwapFrequencies:
		tmp := m.Frequencies()[msg.Pos1]
		m.Frequencies()[msg.Pos1] = m.Frequencies()[msg.Pos2]
		m.Frequencies()[msg.Pos2] = tmp
		m.Frequencies()[msg.Pos1].Position = int64(msg.Pos1)
		m.Frequencies()[msg.Pos2].Position = int64(msg.Pos2)

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
			m.SetIndex(m.index - 1)
		case "j":
			m.SetIndex(m.index + 1)
		case "g":
			m.SetIndex(0)
		case "G":
			m.SetIndex(m.FrequenciesLength() - 1)
		case "ctrl+u":
			m.SetIndex(m.index - m.height/2)
		case "ctrl+d":
			m.SetIndex(m.index + m.height/2)

		case "x":
			frequenciesCount := len(m.Frequencies())
			if frequenciesCount == 1 {
				// Don't delete the last frequency!
				return m, nil
			}
			// TODO: consider adding a confirmation popup
			frequencyId := m.Frequencies()[m.index].ID
			if m.index == frequenciesCount-1 {
				m.SetIndex(m.index - 1)
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

func (m Model) Swap(dir int) (model Model, cmd tea.Cmd) {
	cmd = nil
	networkId := state.NetworkId(m.networkIndex)
	assert.NotNil(networkId, "if frequency can swap it must mean the network id is valid")
	member := state.State.Members[*networkId][*state.UserID]
	if member.IsAdmin {
		cmd = gateway.Send(&packet.SwapFrequencies{
			Network: m.Network().ID,
			Pos1:    m.index,
			Pos2:    m.index + dir,
		})
	}
	m.SetIndex(m.index + dir)
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
		m.base = -1
		return
	}

	m.networkIndex = networkIndex
	m.SetIndex(0)

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
			m.SetIndex(i)
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

func (m *Model) SetIndex(index int) {
	if m.base == -1 {
		m.base = 0
	}
	m.index = min(max(index, 0), m.FrequenciesLength()-1)
	if m.index < m.base {
		m.base = m.index
	} else if m.index >= m.base+m.height {
		m.base = 1 + m.index - m.height
	}
}

func (m Model) renderNetworkName() string {
	bg := lipgloss.Color(m.Network().BgHexColor)
	fg := lipgloss.Color(m.Network().FgHexColor)
	nameStyle := nameStyle.Background(bg).Foreground(fg)
	if m.focus {
		nameStyle = nameStyle.BorderForeground(colors.Focus)
	}
	return nameStyle.Render(m.Network().Name)
}

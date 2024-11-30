package network

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
	width    = 24
	sepStyle = lipgloss.NewStyle().Width(0).
			Border(lipgloss.ThickBorder(), false, true, false, false)
	nameStyle = lipgloss.NewStyle().
			Margin(0, 0, 1).Padding(1).Width(width).Align(lipgloss.Center).
			Border(lipgloss.ThickBorder(), false, false, true)
	margin         = 3
	frequencyStyle = lipgloss.NewStyle().
			Margin(0, margin).Padding(0, 1).Width(width - (margin * 2)).
			Align(lipgloss.Left)
	symbol = "# "
)

type Model struct {
	network *packet.FullNetwork
	history []func(m *Model)
	index   int
	focus   bool
}

func New() Model {
	return Model{
		focus:   false,
		index:   0,
		network: nil,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	if m.network == nil {
		return ""
	}

	var builder strings.Builder

	bg := lipgloss.Color(m.network.BgHexColor)
	fg := lipgloss.Color(m.network.FgHexColor)
	nameStyle := nameStyle.Background(bg).Foreground(fg)
	if m.focus {
		nameStyle = nameStyle.BorderForeground(colors.Focus)
	}
	builder.WriteString(nameStyle.Render(m.network.Name))

	builder.WriteString("\n")
	for i, frequency := range m.network.Frequencies {
		color := colors.White
		if frequency.HexColor != nil {
			color = lipgloss.Color(*frequency.HexColor)
		}
		frequencyStyle := frequencyStyle.Foreground(color)
		if m.index == i {
			frequencyStyle = frequencyStyle.Background(colors.BackgroundHighlight)
		}
		builder.WriteString(frequencyStyle.Render(symbol + frequency.Name))
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
		Network: m.network.ID,
		Pos1:    m.index,
		Pos2:    m.index + dir,
	})
	tmp := m.network.Frequencies[m.index]
	m.network.Frequencies[m.index] = m.network.Frequencies[m.index+dir]
	m.network.Frequencies[m.index+dir] = tmp
	m.index += dir
	m.history = append(m.history, func(m *Model) {
		m.index -= dir
		tmp := m.network.Frequencies[m.index]
		m.network.Frequencies[m.index] = m.network.Frequencies[m.index+dir]
		m.network.Frequencies[m.index+dir] = tmp
	})
	return m, cmd
}

func (m *Model) Set(index int) {
	if index != -1 {
		m.network = &state.State.Networks[index]
	} else {
		m.network = nil
	}
}

func (m Model) FrequenciesLength() int {
	return len(m.network.Frequencies)
}

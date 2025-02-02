package frequencylist

import (
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
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
	sepStyle = lipgloss.NewStyle().Width(0).BorderBackground(colors.BackgroundDim).
			Border(lipgloss.ThickBorder(), false, true, false, false)

	nameStyle = lipgloss.NewStyle().
			Padding(1).Align(lipgloss.Center).
			Border(lipgloss.ThickBorder(), false, false, true)
	networkIdStyle = lipgloss.NewStyle().
			MarginBottom(1).Padding(1, 2).Align(lipgloss.Center).
			Border(lipgloss.ThickBorder(), false, false, true)

	margin         = 2
	padding        = 1
	frequencyStyle = lipgloss.NewStyle().MaxHeight(1).
			Margin(0, margin).Padding(0, padding).Align(lipgloss.Left)

	symbolReadWrite     = "󰖩 "
	symbolReadOnly      = "󱛂 "
	symbolReadOnlyAdmin = "󰖩 "
	symbolNoAccess      = "󱚿 "
	symbolNoAccessAdmin = "󱛀 "
	symbolWidth         = 2

	widthWithoutFrequency = ((margin + padding) * 2) + symbolWidth

	ellipsis = "…"

	BackgroundStyle = lipgloss.NewStyle().Background(colors.BackgroundDim)

	notifSymbol  = "◗"
	notifSymbols = func() []string {
		notifs := []string{
			" 󰲠", " 󰲢", " 󰲤", " 󰲦 ", " 󰲨 ", " 󰲪 ", " 󰲬 ", " 󰲮 ", " 󰲰 ", " 󰲲 ",
		}
		for i, notif := range notifs {
			notifs[i] = notifStyle.Render(notif)
		}
		return notifs
	}()
	notifStyle = lipgloss.NewStyle().Foreground(colors.Red).Inline(true)
	notifWidth = 2
)

type Model struct {
	networkIndex int
	base         int
	index        int
	focus        bool
	width        int
	height       int
}

func New() Model {
	return Model{
		networkIndex: -1,
		base:         0,
		index:        -1,
		focus:        false,
		width:        -1,
		height:       1,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	networkId := state.NetworkId(m.networkIndex)
	if networkId == nil {
		return ""
	}

	frequencyStyle := frequencyStyle.Width(m.width - (margin * 2))
	backgroundStyle := BackgroundStyle.Width(m.width)
	maxFrequencyWidth := m.width - widthWithoutFrequency

	isAdmin := state.State.Members[*networkId][*state.UserID].IsAdmin

	var builder strings.Builder

	builder.WriteString(m.renderNetworkName())
	builder.WriteString("\n")

	frequencies := m.Frequencies()
	upper := min(m.base+m.height, len(frequencies))
	frequencies = frequencies[m.base:upper]

	for i, frequency := range frequencies {
		backgroundStyle := backgroundStyle
		maxFrequencyWidth := maxFrequencyWidth

		frequencyStyle := frequencyStyle.Foreground(lipgloss.Color(frequency.HexColor))
		if m.index == m.base+i {
			frequencyStyle = frequencyStyle.Background(colors.BackgroundHighlight)
		}

		symbol := ""
		if frequency.Perms == packet.PermReadWrite {
			symbol = symbolReadWrite
		} else if frequency.Perms == packet.PermRead && !isAdmin {
			symbol = symbolReadOnly
		} else if frequency.Perms == packet.PermRead && isAdmin {
			symbol = symbolReadOnlyAdmin
		} else if frequency.Perms == packet.PermNoAccess && !isAdmin {
			symbol = symbolNoAccess
		} else if frequency.Perms == packet.PermNoAccess && isAdmin {
			symbol = symbolNoAccessAdmin
		}

		notif := ""
		pings, hasNotif := state.State.Notifications[frequency.ID]
		if hasNotif {
			builder.WriteString(notifSymbol)
			frequencyStyle = frequencyStyle.MarginLeft(margin - 1)
			backgroundStyle = backgroundStyle.Width(m.width - 1)

			if pings != 0 {
				notif = notifSymbols[min(pings, 10)-1]
				maxFrequencyWidth -= notifWidth
			}
		}

		frequencyName := ""
		if lipgloss.Width(frequency.Name) <= maxFrequencyWidth {
			frequencyName = lipgloss.NewStyle().
				MaxWidth(maxFrequencyWidth).
				Render(frequency.Name)
		} else {
			frequencyName = lipgloss.NewStyle().
				MaxWidth(maxFrequencyWidth-1).
				Render(frequency.Name) + ellipsis
		}
		frequencyName = lipgloss.NewStyle().Width(maxFrequencyWidth).
			Render(frequencyName)

		frequency := frequencyStyle.Render(symbol + frequencyName + notif)
		builder.WriteString(backgroundStyle.Render(frequency))
		builder.WriteString("\n")
	}

	sidebar := builder.String()
	sep := sepStyle.Height(ui.Height)
	if m.focus {
		sep = sep.BorderForeground(colors.Focus)
	}
	result := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, sep.String())

	return BackgroundStyle.Render(result)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if state.NetworkId(m.networkIndex) != nil {
		// Calculate height for frequencies
		m.height = ui.Height
		m.height -= lipgloss.Height(m.renderNetworkName())
		m.height -= 1
		m.SetIndex(m.index)
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

		case "i":
			_ = clipboard.WriteAll(strconv.FormatInt(int64(m.Network().ID), 10))

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

func (m *Model) Swap(dir int) (model Model, cmd tea.Cmd) {
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
	return *m, cmd
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
		m.base = 0
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
	m.index = min(max(index, 0), m.FrequenciesLength()-1)
	if m.index < m.base {
		m.base = max(m.index, 0)
	} else if m.index >= m.base+m.height {
		m.base = 1 + m.index - m.height
	}
}

func (m Model) renderNetworkName() string {
	bg := lipgloss.Color(m.Network().BgHexColor)
	fg := lipgloss.Color(m.Network().FgHexColor)
	nameStyle := nameStyle.Background(bg).Foreground(fg).Width(m.width)
	if m.focus {
		nameStyle = nameStyle.BorderForeground(colors.Focus)
	}
	networkName := nameStyle.Render(m.Network().Name)

	networkIdStyle := networkIdStyle.Width(m.width)
	if m.focus {
		networkIdStyle = networkIdStyle.BorderForeground(colors.Focus)
	}
	id := "Invite Code\n" + strconv.FormatInt(int64(m.Network().ID), 10)
	networkId := networkIdStyle.Render(id)

	return lipgloss.JoinVertical(0, networkName, networkId)
}

func (m *Model) SetWidth(width int) {
	m.width = width
}

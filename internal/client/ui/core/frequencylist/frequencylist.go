// Eko: A terminal-native social media platform
// Copyright (C) 2025 Kyren223
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package frequencylist

import (
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/config"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
)

var (
	margin  = 2
	padding = 1

	symbolReadWrite     = "󰖩 "
	symbolReadOnly      = "󱛂 "
	symbolReadOnlyAdmin = "󰖩 "
	symbolNoAccess      = "󱚿 "
	symbolNoAccessAdmin = "󱛀 "
	symbolWidth         = 2

	widthWithoutFrequency = ((margin + padding) * 2) + symbolWidth

	ellipsis    = "…"
	notifSymbol = "◗"
	notifsWidth = 2
	notifs      = []string{
		" 󰲠", " 󰲢", " 󰲤", " 󰲦", " 󰲨", " 󰲪", " 󰲬", " 󰲮", " 󰲰", " 󰲲",
	}

	HorizontalSep = "━"
	VerticalSep   = "┃"
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

	notifStyle := lipgloss.NewStyle().Foreground(colors.Red).Inline(true)
	backgroundStyle := lipgloss.NewStyle().Background(colors.BackgroundDim)
	frequencyStyle := lipgloss.NewStyle().
		MaxHeight(1).Width(m.width-(margin*2)).
		Margin(0, margin).Padding(0, padding).Align(lipgloss.Left).
		Background(colors.BackgroundDim).MarginBackground(colors.BackgroundDim)
	maxFrequencyWidth := m.width - widthWithoutFrequency

	isAdmin := state.State.Members[*networkId][*state.UserID].IsAdmin

	var builder strings.Builder

	builder.WriteString(m.renderHeader())
	builder.WriteString("\n")

	frequencies := m.Frequencies()
	upper := min(m.base+m.height, len(frequencies))
	frequencies = frequencies[m.base:upper]

	for i, frequency := range frequencies {
		maxFrequencyWidth := maxFrequencyWidth

		color := lipgloss.Color(frequency.HexColor)
		if colors.IsDarkened() {
			color = colors.DarkenColor(color, colors.DarkeningFactor)
		}

		frequencyStyle := frequencyStyle.
			Foreground(color)
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
		pings, hasNotif := state.MergedNotification(frequency.ID)
		if hasNotif {
			notifSymbol := lipgloss.NewStyle().
				Foreground(colors.White).Render(notifSymbol)
			builder.WriteString(notifSymbol)
			frequencyStyle = frequencyStyle.MarginLeft(margin - 1)

			if pings != 0 {
				notif = notifs[min(pings, 10)-1]
				notif = notifStyle.Render(notif)
				maxFrequencyWidth -= notifsWidth
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
		builder.WriteString(frequency)
		builder.WriteString("\n")
	}

	focusStyle := lipgloss.NewStyle().Background(colors.BackgroundDim).Foreground(colors.White)
	if m.focus {
		focusStyle = focusStyle.Foreground(colors.Focus)
	}

	if config.ReadConfig().ScreenBorders {
		builder.WriteString(strings.Repeat("\n", m.height-len(frequencies)+1))
		builder.WriteString(focusStyle.Render(strings.Repeat(HorizontalSep, m.width)))
	}

	sidebar := builder.String()

	sep := ""
	if config.ReadConfig().ScreenBorders {
		sep = HorizontalSep + strings.Repeat("\n"+VerticalSep, ui.Height-2) + "\n" + HorizontalSep
	} else {
		sep = strings.Repeat(VerticalSep+"\n", ui.Height)
		sep = sep[:len(sep)-1] // Strip last \n
	}
	sep = focusStyle.Render(sep)

	result := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, sep)

	return backgroundStyle.Render(result)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if state.NetworkId(m.networkIndex) != nil {
		// Calculate height for frequencies
		m.height = ui.Height
		m.height -= lipgloss.Height(m.renderHeader())
		m.height -= 1 // For bottom margin
		if config.ReadConfig().ScreenBorders {
			m.height -= 1 // Only bottom, top is calculated in renderHeader
		}
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

func (m Model) renderHeader() string {
	focusColor := colors.White
	if m.focus {
		focusColor = colors.Focus
	}

	bg := lipgloss.Color(m.Network().BgHexColor)
	fg := lipgloss.Color(m.Network().FgHexColor)

	if colors.IsDarkened() {
		bg = colors.DarkenColor(bg, colors.DarkeningFactor)
		fg = colors.DarkenColor(fg, colors.DarkeningFactor)
	}

	nameStyle := lipgloss.NewStyle().Width(m.width).
		Padding(1).Align(lipgloss.Center).
		Border(lipgloss.ThickBorder(), config.ReadConfig().ScreenBorders, false, true).
		BorderForeground(focusColor).
		Background(bg).Foreground(fg)
	networkName := nameStyle.Render(m.Network().Name)

	networkIdStyle := lipgloss.NewStyle().Width(m.width).
		MarginBottom(1).Padding(1, 2).Align(lipgloss.Center).
		Border(lipgloss.ThickBorder(), false, false, true).
		Background(colors.BackgroundDim).Foreground(colors.White).
		BorderForeground(focusColor)
	id := "Invite Code\n" + strconv.FormatInt(int64(m.Network().ID), 10)
	networkId := networkIdStyle.Render(id)

	return lipgloss.JoinVertical(0, networkName, networkId)
}

func (m *Model) SetWidth(width int) {
	m.width = width
}

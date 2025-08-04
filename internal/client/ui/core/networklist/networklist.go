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

package networklist

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/config"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/packet"
)

const (
	SignalsIndex  = -1
	HorizontalSep = "━"
	VerticalSep   = "┃"
)

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

	focusColor := colors.White
	if m.focus {
		focusColor = colors.Focus
	}
	focusStyle := lipgloss.NewStyle().Background(colors.BackgroundDimmer).Foreground(focusColor)

	var builder strings.Builder

	pings := 0
	for _, signalId := range state.Data.Signals {
		p, _ := state.MergedNotification(signalId)
		pings += p
	}
	var signalsIcon lipgloss.Style
	if pings == 0 {
		signalsIcon = ui.IconStyle(" ", colors.Turquoise, colors.DarkerCyan, colors.BackgroundDimmer)
	} else {
		signalsIcon = ui.IconStyleNotif(" ", colors.Turquoise, colors.DarkerCyan, colors.BackgroundDimmer, pings)
	}

	if config.ReadConfig().ScreenBorders {
		builder.WriteString(focusStyle.Render(strings.Repeat(HorizontalSep, 8)))
		builder.WriteByte('\n')
	}

	builder.WriteByte('\n')
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

	builder.WriteByte('\n')
	builder.WriteString(focusStyle.Render(" ━━━━━━ "))
	builder.WriteByte('\n')

	if m.base != 0 {
		builder.WriteString(lipgloss.NewStyle().Foreground(colors.Purple).Render("   󰜷󰜷   "))
	}
	builder.WriteString("\n")

	networks := state.Data.Networks
	upper := min(m.base+m.height, len(networks))
	networks = networks[m.base:upper]
	for i, networkId := range networks {
		network := state.State.Networks[networkId]
		fg := lipgloss.Color(network.FgHexColor)
		bg := lipgloss.Color(network.BgHexColor)

		if colors.IsDarkened() {
			fg = colors.DarkenColor(fg, colors.DarkeningFactor)
			bg = colors.DarkenColor(bg, colors.DarkeningFactor)
		}

		pings, ok := 0, false
		frequencies := state.State.Frequencies[networkId]
		for _, frequency := range frequencies {
			fpings, fok := state.MergedNotification(frequency.ID)
			pings += fpings
			ok = ok || fok
		}

		var icon lipgloss.Style
		if ok && pings != 0 {
			icon = ui.IconStyleNotif(network.Icon, fg, bg, colors.BackgroundDimmer, pings)
		} else {
			icon = ui.IconStyle(network.Icon, fg, bg, colors.BackgroundDimmer)
		}

		if m.index == m.base+i {
			builder.WriteString(lipgloss.JoinHorizontal(
				ui.Center,
				selectedIndicator,
				icon.Background(colors.BackgroundDimmer).Padding(0, 1, 0, 0).String(),
			))
		} else if ok {
			builder.WriteString(lipgloss.JoinHorizontal(
				ui.Center,
				notification,
				icon.Background(colors.BackgroundDimmer).Padding(0, 1, 0, 0).String(),
			))
		} else {
			builder.WriteString(icon.Background(colors.BackgroundDimmer).Padding(0, 1, 0).String())
		}
		builder.WriteString("\n")

		if upper != len(state.Data.Networks) && i == upper-1 {
			builder.WriteString(lipgloss.NewStyle().Foreground(colors.Purple).Render("   󰜮󰜮   "))
		}

		builder.WriteString("\n")
	}

	if config.ReadConfig().ScreenBorders {
		height := lipgloss.Height(builder.String())
		builder.WriteString(strings.Repeat("\n", ui.Height-height))
		builder.WriteString(focusStyle.Render(strings.Repeat(HorizontalSep, 8)))
	}

	result := builder.String()
	result = result[:len(result)-1] // Strip last \n

	sep := ""
	if config.ReadConfig().ScreenBorders {
		sep = HorizontalSep + strings.Repeat("\n"+VerticalSep, ui.Height-2) + "\n" + HorizontalSep
	} else {
		sep = strings.Repeat(VerticalSep+"\n", ui.Height)
	}
	sep = focusStyle.Render(sep)

	result = lipgloss.JoinHorizontal(lipgloss.Top, result, sep)

	return lipgloss.NewStyle().Background(colors.BackgroundDimmer).MaxHeight(ui.Height).Render(result)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focus {
		return m, nil
	}

	m.height = ui.Height
	if config.ReadConfig().ScreenBorders {
		m.height -= 2 // Top/bottom Borders
	}
	m.height -= 1 // Inital top margin
	m.height -= 2 // Sep line + margin under it
	m.height /= 4 // 4 per icon (3 icon + 1 margin)
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

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

package banreason

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/client/ui/field"
	"github.com/kyren223/eko/internal/client/ui/layouts/flex"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

var (
	width = 48

	blurredBanStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Padding(0, 1).
			Background(colors.Gray).Foreground(colors.White)
	}
	focusedBanStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Padding(0, 1).
			Background(colors.Blue).Foreground(colors.White)
	}
)

const (
	BanReasonField = iota
	BanField
	FieldCount
)

type Model struct {
	networkId snowflake.ID
	userId    snowflake.ID
	banReason field.Model
	banStyle  lipgloss.Style

	selected  int
	nameWidth int
}

func New(userId, networkId snowflake.ID) Model {
	headerStyle := lipgloss.NewStyle().Foreground(colors.Turquoise)

	blurredTextStyle := lipgloss.NewStyle().
		Background(colors.Background).Foreground(colors.White)
	focusedTextStyle := blurredTextStyle.Foreground(colors.Focus)

	fieldBlurredStyle := lipgloss.NewStyle().
		PaddingLeft(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colors.DarkCyan).
		BorderBackground(colors.Background).
		Background(colors.Background)
	fieldFocusedStyle := fieldBlurredStyle.
		Border(lipgloss.ThickBorder()).
		BorderForeground(colors.Focus)

	banReason := field.New(width)
	banReason.Header = "Ban Reason"
	banReason.HeaderStyle = headerStyle
	banReason.FocusedStyle = fieldFocusedStyle
	banReason.BlurredStyle = fieldBlurredStyle
	banReason.FocusedTextStyle = focusedTextStyle
	banReason.BlurredTextStyle = blurredTextStyle
	banReason.Input.CharLimit = packet.MaxBanReasonBytes
	banReason.Focus()
	nameWidth := lipgloss.Width(banReason.View())

	return Model{
		networkId: networkId,
		userId:    userId,
		banReason: banReason,
		banStyle:  blurredBanStyle(),
		selected:  0,
		nameWidth: nameWidth,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	name := m.banReason.View()

	ban := lipgloss.NewStyle().
		Width(m.nameWidth).
		Background(colors.Background).
		Align(lipgloss.Center).
		Render(m.banStyle.Render("Ban", state.State.Users[m.userId].Name))

	content := flex.NewVertical(name, ban).WithGap(1).View()

	return lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		Padding(1, 4).
		Align(lipgloss.Center, lipgloss.Center).
		BorderBackground(colors.Background).
		BorderForeground(colors.White).
		Background(colors.Background).
		Foreground(colors.White).
		Render(content)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.Type
		switch key {
		case tea.KeyTab:
			return m, m.cycle(1)
		case tea.KeyShiftTab:
			return m, m.cycle(-1)

		default:
			var cmd tea.Cmd
			switch m.selected {
			case BanReasonField:
				m.banReason, cmd = m.banReason.Update(msg)
			}
			return m, cmd
		}
	}

	return m, nil
}

func (m *Model) cycle(step int) tea.Cmd {
	m.selected += step
	if m.selected < 0 {
		m.selected = FieldCount - 1
	} else {
		m.selected %= FieldCount
	}
	return m.updateFocus()
}

func (m *Model) updateFocus() tea.Cmd {
	m.banReason.Blur()
	m.banStyle = blurredBanStyle()
	switch m.selected {
	case BanReasonField:
		return m.banReason.Focus()
	case BanField:
		m.banStyle = focusedBanStyle()
		return nil
	default:
		assert.Never("missing switch statement field in update focus", "selected", m.selected)
		return nil
	}
}

func (m *Model) Select() tea.Cmd {
	if m.selected != BanField {
		return nil
	}

	banReason := m.banReason.Input.Value()

	yes := true
	return gateway.Send(&packet.SetMember{
		Member:    nil,
		Admin:     nil,
		Muted:     nil,
		Banned:    &yes,
		BanReason: &banReason,
		Network:   m.networkId,
		User:      m.userId,
	})
}

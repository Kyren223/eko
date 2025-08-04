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

package banview

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/client/ui/layouts/flex"
	"github.com/kyren223/eko/pkg/snowflake"
)

var width = 48

type Model struct {
	banReason string
	name      string
	id        string
}

func New(userId, networkId snowflake.ID) Model {
	return Model{
		banReason: *state.State.Members[networkId][userId].BanReason,
		name:      state.State.Users[userId].Name,
		id:        strconv.FormatInt(int64(userId), 10),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	headerStyle := lipgloss.NewStyle().
		Width(width).
		Background(colors.Background).
		Foreground(colors.Focus)
	grayStyle := lipgloss.NewStyle().
		Width(width).
		Background(colors.Background).
		Foreground(colors.LightGray)

	nameHeader := headerStyle.Render("Banned Username:")
	name := m.name + grayStyle.Render(" ("+m.id+")")
	name = lipgloss.NewStyle().
		Width(width).
		Background(colors.Background).
		Render(name)

	banReasonHeader := headerStyle.Render("Ban Reason:")
	banReason := lipgloss.NewStyle().Width(width).Render(m.banReason)

	content := flex.NewVertical(nameHeader, name, banReasonHeader, banReason).WithGap(1).View()

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
	return m, nil
}

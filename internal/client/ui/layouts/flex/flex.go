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

package flex

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	Style     lipgloss.Style
	ItemStyle lipgloss.Style

	contents []string

	Position lipgloss.Position
	Gap      int
	Vertical bool
}

func NewHorizontal(contents ...string) Model {
	return Model{
		contents: contents,
		Vertical: false,
	}
}

func NewVertical(contents ...string) Model {
	return Model{
		contents: contents,
		Vertical: true,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	contents := make([]string, len(m.contents))
	for i, content := range m.contents {
		contents[i] = m.ItemStyle.Render(content)
		if i != 0 {
			if m.Vertical {
				contents[i] = lipgloss.NewStyle().PaddingTop(m.Gap).Render(contents[i])
			} else {
				contents[i] = lipgloss.NewStyle().PaddingLeft(m.Gap).Render(contents[i])
			}
		}
	}

	var result string
	if m.Vertical {
		result = lipgloss.JoinVertical(m.Position, contents...)
	} else {
		result = lipgloss.JoinHorizontal(m.Position, contents...)
	}

	return m.Style.Render(result)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m *Model) SetContents(contents ...string) {
	m.contents = contents
}

func (m Model) WithGap(gap int) Model {
	m.Gap = gap
	return m
}

// Eko: A terminal based social media platform
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

package memberlist

import (
	"bytes"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/config"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/banlist"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/snowflake"
)

var (
	margin  = 2
	padding = 1

	symbolWidth        = 2
	widthWithoutMember = ((margin + padding) * 2) + symbolWidth

	ellipsis      = "…"
	HorizontalSep = "━"
	VerticalSep   = "┃"
)

type Model struct {
	networkIndex   int
	frequencyIndex int
	base           int
	index          int
	focus          bool
	width          int
	height         int

	banlist *banlist.Model
}

func New() Model {
	return Model{
		networkIndex:   -1,
		frequencyIndex: -1,
		base:           0,
		index:          -1,
		focus:          false,
		width:          -1,
		height:         -1,
		banlist:        nil,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	networkId := state.NetworkId(m.networkIndex)
	if networkId == nil || m.frequencyIndex == -1 {
		return ""
	}

	if m.banlist != nil {
		return m.banlist.View()
	}

	backgroundStyle := lipgloss.NewStyle().Background(colors.BackgroundDim)

	memberStyle := lipgloss.NewStyle().Width(m.width-(margin*2)).
		Background(colors.BackgroundDim).
		Margin(0, margin).Padding(0, padding).Align(lipgloss.Left).
		Background(colors.BackgroundDim).MarginBackground(colors.BackgroundDim)

	maxMemberWidth := m.width - widthWithoutMember
	ownerId := state.State.Networks[*networkId].OwnerID

	var builder strings.Builder

	builder.WriteString(m.renderHeader())
	builder.WriteString("\n")

	members := m.Members()
	upper := min(m.base+m.height, len(members))
	members = members[m.base:upper]

	for i, member := range members {
		memberStyle := memberStyle
		if m.index == m.base+i {
			memberStyle = memberStyle.Background(colors.BackgroundHighlight)
		}

		user := state.State.Users[member.UserID]
		trustedPublicKey, isTrusted := state.State.TrustedUsers[user.ID]
		keysMatch := bytes.Equal(trustedPublicKey, user.PublicKey)

		var userStyle lipgloss.Style
		if isTrusted && keysMatch {
			if ownerId == member.UserID {
				userStyle = ui.TrustedOwnerStyle()
			} else if member.IsAdmin {
				userStyle = ui.TrustedAdminStyle()
			} else {
				userStyle = ui.TrustedMemberStyle()
			}
		} else {
			if ownerId == member.UserID {
				userStyle = ui.OwnerStyle()
			} else if member.IsAdmin {
				userStyle = ui.AdminStyle()
			} else {
				userStyle = ui.UserStyle()
			}
		}
		memberName := m.Users()[member.UserID].Name
		memberName = userStyle.Render(memberName)
		if isTrusted && !keysMatch {
			memberName = ui.UntrustedSymbol() + memberName
		}

		if lipgloss.Width(memberName) <= maxMemberWidth {
			memberName = lipgloss.NewStyle().
				MaxWidth(maxMemberWidth).
				Render(memberName)
		} else {
			ellipsisStyle := lipgloss.NewStyle().
				Background(memberStyle.GetBackground()).Foreground(userStyle.GetForeground())
			memberName = lipgloss.NewStyle().
				MaxWidth(maxMemberWidth-1).
				Render(memberName) + ellipsisStyle.Render(ellipsis)
		}

		builder.WriteString(memberStyle.Render(memberName))
		builder.WriteString("\n")
	}

	focusStyle := lipgloss.NewStyle().Background(colors.BackgroundDim).Foreground(colors.White)
	if m.focus {
		focusStyle = focusStyle.Foreground(colors.Focus)
	}

	if config.ReadConfig().ScreenBorders {
		builder.WriteString(strings.Repeat("\n", m.height-len(members)+1))
		builder.WriteString(focusStyle.Render(strings.Repeat(HorizontalSep, m.width)))
	}

	sidebar := backgroundStyle.Height(ui.Height).Render(builder.String())

	sep := ""
	if config.ReadConfig().ScreenBorders {
		sep = HorizontalSep + strings.Repeat("\n"+VerticalSep, ui.Height-2) + "\n" + HorizontalSep
	} else {
		sep = strings.Repeat(VerticalSep+"\n", ui.Height)
		sep = sep[:len(sep)-1]
	}
	sep = focusStyle.Render(sep)

	result := lipgloss.JoinHorizontal(lipgloss.Top, sep, sidebar)
	return result
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if state.NetworkId(m.networkIndex) == nil {
		return m, nil
	}

	if key, ok := msg.(tea.KeyMsg); ok && m.focus {
		if key.String() == "b" {
			if m.banlist == nil {
				banlist := banlist.New(m.Network().ID)
				m.banlist = &banlist
				m.banlist.SetWidth(m.width)
				return m, m.banlist.Init()
			} else {
				m.banlist = nil
				return m, nil
			}
		}
	}

	if m.banlist != nil {
		banlist, cmd := m.banlist.Update(msg)
		m.banlist = &banlist
		return m, cmd
	}

	// Calculate height for members
	m.height = ui.Height
	m.height -= lipgloss.Height(m.renderHeader())
	m.height -= 1 // For bottom margin
	if config.ReadConfig().ScreenBorders {
		m.height -= 1 // Only bottom, top is calculated in renderHeader
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focus {
			return m, nil
		}

		key := msg.String()
		switch key {
		case "k":
			m.SetIndex(m.index - 1)
		case "j":
			m.SetIndex(m.index + 1)
		case "g":
			m.SetIndex(0)
		case "G":
			m.SetIndex(m.MembersLength() - 1)
		case "ctrl+u":
			m.SetIndex(m.index - m.height/2)
		case "ctrl+d":
			m.SetIndex(m.index + m.height/2)

		case "p":
			member := m.Members()[m.index]

			return m, func() tea.Msg {
				return ui.ProfilePopupMsg{
					User: member.UserID,
				}
			}

		// Normal
		case "T":
			member := m.Members()[m.index]

			if member.UserID == *state.UserID {
				return m, nil
			}

			_, isTrusting := state.State.TrustedUsers[member.UserID]

			_, isBlocked := state.State.BlockedUsers[member.UserID]
			if !isTrusting && isBlocked {
				return m, nil
			}

			return m, gateway.Send(&packet.TrustUser{
				User:  member.UserID,
				Trust: !isTrusting,
			})

		// Admin
		case "K":
			networkId := state.NetworkId(m.networkIndex)
			network := state.State.Networks[*networkId]
			member := m.Members()[m.index]

			if !m.MembersMap()[*state.UserID].IsAdmin {
				return m, nil
			}

			if member.UserID == *state.UserID {
				return m, nil
			}

			if member.IsAdmin && network.OwnerID != *state.UserID {
				return m, nil
			}

			no := false
			return m, gateway.Send(&packet.SetMember{
				Member:    &no,
				Admin:     nil,
				Muted:     nil,
				Banned:    nil,
				BanReason: nil,
				Network:   *state.NetworkId(m.networkIndex),
				User:      member.UserID,
			})
		case "M":
			networkId := state.NetworkId(m.networkIndex)
			network := state.State.Networks[*networkId]
			member := m.Members()[m.index]

			if member.IsMuted {
				return m, nil
			}

			if !m.MembersMap()[*state.UserID].IsAdmin {
				return m, nil
			}

			if member.UserID == *state.UserID {
				return m, nil
			}

			if member.IsAdmin && network.OwnerID != *state.UserID {
				return m, nil
			}

			yes := true
			return m, gateway.Send(&packet.SetMember{
				Member:    nil,
				Admin:     nil,
				Muted:     &yes,
				Banned:    nil,
				BanReason: nil,
				Network:   *state.NetworkId(m.networkIndex),
				User:      member.UserID,
			})

		case "U":
			networkId := state.NetworkId(m.networkIndex)
			network := state.State.Networks[*networkId]
			member := m.Members()[m.index]

			if !member.IsMuted {
				return m, nil
			}

			if !m.MembersMap()[*state.UserID].IsAdmin {
				return m, nil
			}

			if member.UserID == *state.UserID {
				return m, nil
			}

			if member.IsAdmin && network.OwnerID != *state.UserID {
				return m, nil
			}

			no := false
			return m, gateway.Send(&packet.SetMember{
				Member:    nil,
				Admin:     nil,
				Muted:     &no,
				Banned:    nil,
				BanReason: nil,
				Network:   *state.NetworkId(m.networkIndex),
				User:      member.UserID,
			})
		case "B":
			networkId := state.NetworkId(m.networkIndex)
			network := state.State.Networks[*networkId]
			member := m.Members()[m.index]

			if !m.MembersMap()[*state.UserID].IsAdmin {
				return m, nil
			}

			if member.UserID == *state.UserID {
				return m, nil
			}

			if member.IsAdmin && network.OwnerID != *state.UserID {
				return m, nil
			}

			cmd := func() tea.Msg {
				return ui.BanReasonPopupMsg{
					Network: *state.NetworkId(m.networkIndex),
					User:    member.UserID,
				}
			}
			return m, cmd

		// Owner
		case "D":
			networkId := state.NetworkId(m.networkIndex)
			network := state.State.Networks[*networkId]
			member := m.Members()[m.index]

			// Can't demote yourself
			if member.UserID == *state.UserID {
				return m, nil
			}

			if !member.IsAdmin || network.OwnerID != *state.UserID {
				return m, nil
			}

			no := false
			return m, gateway.Send(&packet.SetMember{
				Member:    nil,
				Admin:     &no,
				Muted:     nil,
				Banned:    nil,
				BanReason: nil,
				Network:   *state.NetworkId(m.networkIndex),
				User:      member.UserID,
			})
		case "P":
			networkId := state.NetworkId(m.networkIndex)
			network := state.State.Networks[*networkId]
			member := m.Members()[m.index]

			// Can't promote yourself
			if member.UserID == *state.UserID {
				return m, nil
			}

			if member.IsAdmin || network.OwnerID != *state.UserID {
				return m, nil
			}

			yes := true
			return m, gateway.Send(&packet.SetMember{
				Member:    nil,
				Admin:     &yes,
				Muted:     nil,
				Banned:    nil,
				BanReason: nil,
				Network:   *state.NetworkId(m.networkIndex),
				User:      member.UserID,
			})

		}
	}

	return m, nil
}

func (m *Model) Focus() {
	m.focus = true
	if m.index == -1 {
		m.SetIndex(0)
	}
}

func (m *Model) Blur() {
	m.focus = false
	m.index = -1
	m.base = 0

	m.banlist = nil
}

func (m *Model) SetNetworkAndFrequency(networkIndex, frequencyIndex int) {
	if networkIndex == -1 || frequencyIndex == -1 {
		m.networkIndex = -1
		m.frequencyIndex = -1
		m.index = -1
		m.base = 0
		return
	}
	m.networkIndex = networkIndex

	if m.frequencyIndex != -1 && m.frequencyIndex < len(m.Frequencies()) {
		fromFrequency := m.Frequencies()[m.frequencyIndex]
		toFrequency := m.Frequencies()[frequencyIndex]

		diffPerms := fromFrequency.Perms != toFrequency.Perms
		fromIsNoAccess := fromFrequency.Perms == packet.PermNoAccess
		toIsNoAccess := toFrequency.Perms == packet.PermNoAccess
		if (fromIsNoAccess || toIsNoAccess) && diffPerms {
			m.base = 0
			m.index = -1
		}
	}

	m.frequencyIndex = frequencyIndex
}

func (m *Model) MembersLength() int {
	networkId := state.NetworkId(m.networkIndex)
	if networkId == nil {
		return 0
	}
	return len(m.Members())
}

func (m *Model) Network() *data.Network {
	networkId := state.NetworkId(m.networkIndex)
	if networkId == nil {
		return nil
	}
	network := state.State.Networks[*networkId]
	return &network
}

func (m *Model) Frequencies() []data.Frequency {
	networkId := state.NetworkId(m.networkIndex)
	if networkId == nil {
		return nil
	}
	return state.State.Frequencies[*networkId]
}

func (m *Model) Members() []data.Member {
	networkId := state.NetworkId(m.networkIndex)
	ownerId := state.State.Networks[*networkId].OwnerID
	membersMap := m.MembersMap()
	members := make([]data.Member, 0, len(membersMap))
	for _, member := range membersMap {
		if member.IsMember {
			members = append(members, member)
		}
	}
	slices.SortFunc(members, func(a, b data.Member) int {
		if a.UserID == ownerId {
			return -1
		} else if b.UserID == ownerId {
			return 1
		}

		if a.IsAdmin && !b.IsAdmin {
			return -1
		} else if !a.IsAdmin && b.IsAdmin {
			return 1
		}

		aName, bName := m.Users()[a.UserID].Name, m.Users()[b.UserID].Name
		if aName == bName {
			return int(a.UserID.Time()) - int(b.UserID.Time())
		} else if aName < bName {
			return -1
		} else {
			return 1
		}
	})

	return members
}

func (m *Model) MembersMap() map[snowflake.ID]data.Member {
	networkId := state.NetworkId(m.networkIndex)
	if networkId == nil {
		return nil
	}
	return state.State.Members[*networkId]
}

func (m *Model) Users() map[snowflake.ID]data.User {
	networkId := state.NetworkId(m.networkIndex)
	if networkId == nil {
		return nil
	}
	return state.State.Users
}

func (m *Model) Index() int {
	return m.index
}

func (m *Model) SetIndex(index int) {
	m.index = min(max(index, 0), m.MembersLength()-1)
	if m.index < m.base && m.index != -1 {
		m.base = m.index
	} else if m.index >= m.base+m.height {
		m.base = 1 + m.index - m.height
	}
}

func (m Model) renderHeader() string {
	headerStyle := lipgloss.NewStyle().Width(m.width).
		Background(colors.BackgroundDim).
		Margin(0, 0, 1).Padding(1).Align(lipgloss.Center).
		Border(lipgloss.ThickBorder(), config.ReadConfig().ScreenBorders, false, true).
		BorderForeground(colors.White).Foreground(colors.White)
	if m.focus {
		headerStyle = headerStyle.BorderForeground(colors.Focus)
	}
	return headerStyle.Render("Member List")
}

func (m *Model) SetWidth(width int) {
	m.width = width
	if m.banlist != nil {
		m.banlist.SetWidth(width)
	}
}

func (m *Model) IsBanList() bool {
	return m.banlist != nil
}

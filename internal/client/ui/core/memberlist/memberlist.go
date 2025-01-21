package memberlist

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/snowflake"
)

var (
	sepStyle = lipgloss.NewStyle().Width(0).
			Border(lipgloss.ThickBorder(), false, true, false, false)

	headerStyle = lipgloss.NewStyle().
			Margin(0, 0, 1).Padding(1).Align(lipgloss.Center).
			Border(lipgloss.ThickBorder(), false, false, true)

	margin      = 2
	padding     = 1
	memberStyle = lipgloss.NewStyle().
			Margin(0, margin).Padding(0, padding).Align(lipgloss.Left)

	symbolReadWrite     = "󰖩 "
	symbolReadOnly      = "󱛂 "
	symbolReadOnlyAdmin = "󰖩 "
	symbolNoAccess      = "󱚿 "
	symbolNoAccessAdmin = "󱛀 "
	symbolWidth         = 2

	widthWithoutMember = ((margin + padding) * 2) + symbolWidth

	ellipsis = "…"
)

type Model struct {
	networkIndex   int
	frequencyIndex int
	base           int
	index          int
	focus          bool
	width          int
	height         int
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

	memberStyle := memberStyle.Width(m.width - (margin * 2))
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

		userStyle := ui.NormalMemberStyle
		if ownerId == member.UserID {
			userStyle = ui.OwnerMemberStyle
		} else if member.IsAdmin {
			userStyle = ui.AdminMemberStyle
		}
		memberName := m.Users()[member.UserID].Name
		memberName = userStyle.Render(memberName)

		if lipgloss.Width(memberName) <= maxMemberWidth {
			memberName = lipgloss.NewStyle().
				MaxWidth(maxMemberWidth).
				Render(memberName)
		} else {
			memberName = lipgloss.NewStyle().
				MaxWidth(maxMemberWidth-1).
				Render(memberName) + ellipsis
		}

		builder.WriteString(memberStyle.Render(memberName))
		builder.WriteString("\n")
	}

	sidebar := builder.String()
	sep := sepStyle.Height(ui.Height)
	if m.focus {
		sep = sep.BorderForeground(colors.Focus)
	}
	result := lipgloss.JoinHorizontal(lipgloss.Top, sep.String(), sidebar)

	return result
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if state.NetworkId(m.networkIndex) != nil {
		// Calculate height for frequencies
		m.height = ui.Height
		m.height -= lipgloss.Height(m.renderHeader())
		m.height -= 1
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
			// TODO: profile

		// Admin
		case "x", "K":
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

			if member.UserID == network.OwnerID {
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
		case "b":
			// TODO: ban

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
		case "T":
			// TODO: transfer ownership

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
	return len(state.State.Members[*networkId])
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
		members = append(members, member)
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
	headerStyle := headerStyle.Width(m.width)
	if m.focus {
		headerStyle = headerStyle.BorderForeground(colors.Focus)
	}
	return headerStyle.Render("Member List")
}

func (m *Model) SetWidth(width int) {
	m.width = width
}

package banlist

import (
	"bytes"
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
	sepStyle = lipgloss.NewStyle().Width(0).Background(colors.BackgroundDim).
			Border(lipgloss.ThickBorder(), false, true, false, false)

	headerStyle = lipgloss.NewStyle().Background(colors.BackgroundDim).
			Margin(0, 0, 1).Padding(1).Align(lipgloss.Center).
			Border(lipgloss.ThickBorder(), false, false, true)

	margin      = 2
	padding     = 1
	memberStyle = lipgloss.NewStyle().Background(colors.BackgroundDim).
			Margin(0, margin).Padding(0, padding).Align(lipgloss.Left)

	symbolWidth        = 2
	widthWithoutMember = ((margin + padding) * 2) + symbolWidth

	ellipsis = "…"

	BackgroundStyle = lipgloss.NewStyle().Background(colors.BackgroundDim)
)

type Model struct {
	networkId snowflake.ID
	base      int
	index     int
	width     int
	height    int
}

func New(networkId snowflake.ID) Model {
	m := Model{
		networkId: networkId,
		base:      0,
		index:     -1,
		width:     -1,
		height:    -1,
	}

	m.height = ui.Height
	m.height -= lipgloss.Height(m.renderHeader())
	m.height -= 1

	return m
}

func (m Model) Init() tea.Cmd {
	return gateway.Send(&packet.GetBannedMembers{
		Network: m.networkId,
	})
}

func (m Model) View() string {
	memberStyle := memberStyle.Width(m.width - (margin * 2))
	backgroundStyle := BackgroundStyle.Width(m.width)
	maxMemberWidth := m.width - widthWithoutMember
	ownerId := state.State.Networks[m.networkId].OwnerID

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
		trustedPublicKey, isTrusted := state.State.Trusteds[user.ID]
		keysMatch := bytes.Equal(trustedPublicKey, user.PublicKey)

		var userStyle lipgloss.Style
		if isTrusted && keysMatch {
			if ownerId == member.UserID {
				userStyle = ui.TrustedOwnerMemberStyle
			} else if member.IsAdmin {
				userStyle = ui.TrustedAdminMemberStyle
			} else {
				userStyle = ui.TrustedNormalMemberStyle
			}
		} else {
			if ownerId == member.UserID {
				userStyle = ui.OwnerMemberStyle
			} else if member.IsAdmin {
				userStyle = ui.AdminMemberStyle
			} else {
				userStyle = ui.NormalMemberStyle
			}
		}
		memberName := state.State.Users[member.UserID].Name
		memberName = userStyle.Render(memberName)
		if isTrusted && !keysMatch {
			memberName = ui.UntrustedSymbol + memberName
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

		builder.WriteString(backgroundStyle.Render(memberStyle.Render(memberName)))
		builder.WriteString("\n")
	}

	sidebar := BackgroundStyle.Height(ui.Height).Render(builder.String())
	sep := sepStyle.Height(ui.Height)
	sep = sep.BorderForeground(colors.Focus)
	result := lipgloss.JoinHorizontal(lipgloss.Top, sep.String(), sidebar)

	return result
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	// Calculate height for banned members
	m.height = ui.Height
	m.height -= lipgloss.Height(m.renderHeader())
	m.height -= 1

	switch msg := msg.(type) {
	case tea.KeyMsg:
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

		case "v", "V":
			members := m.Members()
			if 0 <= m.index && m.index < len(members) {
				member := members[m.index]
				return m, func() tea.Msg {
					return ui.BanViewPopupmsg{
						Network: m.networkId,
						User:    member.UserID,
					}
				}
			}

		case "U":
			members := m.Members()
			if 0 <= m.index && m.index < len(members) {
				member := members[m.index]
				no := false
				return m, gateway.Send(&packet.SetMember{
					Member:    nil,
					Admin:     nil,
					Muted:     nil,
					Banned:    &no,
					BanReason: nil,
					Network:   m.networkId,
					User:      member.UserID,
				})
			}

		}
	}

	return m, nil
}

func (m *Model) MembersLength() int {
	return len(m.Members())
}

func (m *Model) Members() []data.Member {
	ownerId := state.State.Networks[m.networkId].OwnerID
	membersMap := state.State.Members[m.networkId]
	members := make([]data.Member, 0, len(membersMap))
	for _, member := range membersMap {
		if member.IsBanned {
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

		aName, bName := state.State.Users[a.UserID].Name, state.State.Users[b.UserID].Name
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

func (m *Model) ndex() int {
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
	headerStyle = headerStyle.BorderForeground(colors.Focus)
	return headerStyle.Render("Ban List")
}

func (m *Model) SetWidth(width int) {
	m.width = width
}

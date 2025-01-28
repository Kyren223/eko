package peerlist

import (
	"bytes"
	"slices"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/snowflake"
)

var (
	sepStyle = lipgloss.NewStyle().Width(0).
			Border(lipgloss.ThickBorder(), false, true, false, false)

	nameStyle = lipgloss.NewStyle().
			Padding(1).Align(lipgloss.Center).
			Border(lipgloss.ThickBorder(), false, false, true)
	userIdStyle = lipgloss.NewStyle().
			MarginBottom(1).Padding(1, 2).Align(lipgloss.Center).
			Border(lipgloss.ThickBorder(), false, false, true)

	margin    = 2
	padding   = 1
	peerStyle = lipgloss.NewStyle().
			Margin(0, margin).Padding(0, padding).Align(lipgloss.Left)

	symbolWidth      = 2
	widthWithoutUser = ((margin + padding) * 2) + symbolWidth

	ellipsis = "…"

	BackgroundStyle = lipgloss.NewStyle().Background(colors.BackgroundDim)
)

type Model struct {
	base   int
	index  int
	focus  bool
	width  int
	height int
}

func New() Model {
	return Model{
		base:   0,
		index:  -1,
		focus:  false,
		width:  -1,
		height: 1,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	peerStyle := peerStyle.Width(m.width - (margin * 2))
	backgroundStyle := BackgroundStyle.Width(m.width)
	maxUserWidth := m.width - widthWithoutUser

	var builder strings.Builder

	builder.WriteString(m.renderHeader())

	builder.WriteString("\n")
	peers := m.Peers()
	upper := min(m.base+m.height, len(peers))
	peers = peers[m.base:upper]
	for i, peer := range peers {
		peerStyle := peerStyle

		user := state.State.Users[peer]
		trustedPublicKey, isTrusted := state.State.Trusteds[user.ID]
		keysMatch := bytes.Equal(trustedPublicKey, user.PublicKey)

		var userStyle lipgloss.Style
		if isTrusted && keysMatch {
			userStyle = ui.TrustedUserStyle.Background(colors.BackgroundDim)
		} else {
			userStyle = ui.UserStyle.Background(colors.BackgroundDim)
		}

		if m.index == m.base+i {
			peerStyle = peerStyle.Background(colors.BackgroundHighlight)
			userStyle = userStyle.Background(colors.BackgroundHighlight)
		}

		username := user.Name
		username = userStyle.Render(username)
		if isTrusted && !keysMatch {
			username = ui.UntrustedSymbol + username
		}

		if lipgloss.Width(username) <= maxUserWidth {
			username = lipgloss.NewStyle().
				MaxWidth(maxUserWidth).
				Render(username)
		} else {
			ellipsisStyle := lipgloss.NewStyle().
				Background(userStyle.GetBackground()).
				Foreground(userStyle.GetForeground())
			username = lipgloss.NewStyle().
				MaxWidth(maxUserWidth-1).
				Render(username) + ellipsisStyle.Render(ellipsis)
		}

		builder.WriteString(backgroundStyle.Render(peerStyle.Render(username)))
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
	// Calculate height for frequencies
	m.height = ui.Height
	m.height -= lipgloss.Height(m.renderHeader())
	m.height -= 1
	m.SetIndex(m.index)

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
			m.SetIndex(len(m.Peers()) - 1)
		case "ctrl+u":
			m.SetIndex(m.index - m.height/2)
		case "ctrl+d":
			m.SetIndex(m.index + m.height/2)

		case "i":
			_ = clipboard.WriteAll(strconv.FormatInt(int64(*state.UserID), 10))

		case "c":
			if m.index == -1 {
				return m, nil
			}
			state.Data.Peers = slices.Delete(state.Data.Peers, m.index, m.index+1)
			if m.index == len(state.Data.Peers) {
				m.SetIndex(m.index - 1)
			}
			data := state.JsonUserData()
			return m, gateway.Send(&packet.SetUserData{
				Data: &data,
				User: nil,
			})

		case "T":
			if m.index == -1 {
				return m, nil
			}
			userId := state.Data.Peers[m.index]

			_, isTrusting := state.State.Trusteds[userId]

			return m, gateway.Send(&packet.TrustUser{
				User:  userId,
				Trust: !isTrusting,
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

func (m Model) Peers() []snowflake.ID {
	return state.Data.Peers
}

func (m *Model) Index() int {
	return m.index
}

func (m *Model) SetIndex(index int) {
	m.index = min(max(index, 0), len(m.Peers())-1)
	if m.index < m.base {
		m.base = max(m.index, 0)
	} else if m.index >= m.base+m.height {
		m.base = 1 + m.index - m.height
	}
}

func (m Model) renderHeader() string {
	nameStyle := nameStyle.Background(colors.DarkerCyan).Foreground(colors.Turquoise).Width(m.width)
	if m.focus {
		nameStyle = nameStyle.BorderForeground(colors.Focus)
	}
	peersName := nameStyle.Render("User Signals")

	userIdStyle := userIdStyle.Width(m.width)
	if m.focus {
		userIdStyle = userIdStyle.BorderForeground(colors.Focus)
	}
	id := "Your User ID\n" + strconv.FormatInt(int64(*state.UserID), 10)
	userId := userIdStyle.Render(id)

	return lipgloss.JoinVertical(0, peersName, userId)
}

func (m *Model) SetWidth(width int) {
	m.width = width
}

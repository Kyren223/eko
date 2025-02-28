package signallist

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
	margin  = 2
	padding = 1

	widthWithoutUser = ((margin + padding) * 2)

	ellipsis = "…"

	notifWidth = 2
	notifs     = []string{
		" 󰲠", " 󰲢", " 󰲤", " 󰲦", " 󰲨", " 󰲪", " 󰲬", " 󰲮", " 󰲰", " 󰲲",
	}
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
	signalStyle := lipgloss.NewStyle().Width(m.width-(margin*2)).
		Margin(0, margin).Padding(0, padding).Align(lipgloss.Left).
		Background(colors.BackgroundDim).MarginBackground(colors.BackgroundDim)
	backgroundStyle := lipgloss.NewStyle().Background(colors.BackgroundDim)
	maxUserWidth := m.width - widthWithoutUser

	var builder strings.Builder

	builder.WriteString(m.renderHeader())

	builder.WriteString("\n")
	signals := m.Signals()
	upper := min(m.base+m.height, len(signals))
	signals = signals[m.base:upper]
	for i, signal := range signals {
		signalStyle := signalStyle
		maxUserWidth := maxUserWidth

		user := state.State.Users[signal]
		trustedPublicKey, isTrusted := state.State.TrustedUsers[user.ID]
		keysMatch := bytes.Equal(trustedPublicKey, user.PublicKey)

		var userStyle lipgloss.Style
		if isTrusted && keysMatch {
			userStyle = ui.TrustedUserStyle().Background(colors.BackgroundDim)
		} else {
			userStyle = ui.UserStyle().Background(colors.BackgroundDim)
		}

		notif := ""
		pings := state.State.Notifications[signal]
		if pings != 0 {
			notif = notifs[min(pings, 10)-1]
			notif = lipgloss.NewStyle().Inline(true).
				Foreground(colors.Red).Background(colors.BackgroundDim).
				Render(notif)
			maxUserWidth -= notifWidth
		}

		if m.index == m.base+i {
			signalStyle = signalStyle.Background(colors.BackgroundHighlight)
			userStyle = userStyle.Background(colors.BackgroundHighlight)
			notif = lipgloss.NewStyle().Background(colors.BackgroundHighlight).
				Render(notif)
		}

		username := user.Name
		username = userStyle.Render(username)
		if isTrusted && !keysMatch {
			username = ui.UntrustedSymbol() + username
		}

		blockSymbol := ""
		if _, ok := state.State.BlockedUsers[user.ID]; ok {
			maxUserWidth -= 2 // blocked symbol width
			blockSymbol = ui.BlockedSymbol()
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
		if m.index == m.base+i {
			blockSymbol = lipgloss.NewStyle().
				Background(colors.BackgroundHighlight).Render(blockSymbol)
			username = lipgloss.NewStyle().Width(maxUserWidth).
				Background(colors.BackgroundHighlight).Render(username + blockSymbol)
		} else {
			blockSymbol = lipgloss.NewStyle().
				Background(colors.BackgroundDim).Render(blockSymbol)
			username = lipgloss.NewStyle().Width(maxUserWidth).
				Background(colors.BackgroundDim).Render(username + blockSymbol)
		}

		signal := signalStyle.Render(username + notif)
		builder.WriteString(signal)
		builder.WriteString("\n")
	}

	sidebar := builder.String()

	sep := lipgloss.NewStyle().Width(0).Height(ui.Height).
		BorderBackground(colors.BackgroundDim).BorderForeground(colors.White).
		Border(lipgloss.ThickBorder(), false, true, false, false)
	if m.focus {
		sep = sep.BorderForeground(colors.Focus)
	}

	result := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, sep.String())
	return backgroundStyle.Render(result)
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
			m.SetIndex(len(m.Signals()) - 1)
		case "ctrl+u":
			m.SetIndex(m.index - m.height/2)
		case "ctrl+d":
			m.SetIndex(m.index + m.height/2)

		case "i":
			_ = clipboard.WriteAll(strconv.FormatInt(int64(*state.UserID), 10))

		case "p":
			if m.index == -1 {
				return m, nil
			}
			userId := state.Data.Signals[m.index]

			return m, func() tea.Msg {
				return ui.ProfilePopupMsg{
					User: userId,
				}
			}

		case "c":
			if m.index == -1 {
				return m, nil
			}
			state.Data.Signals = slices.Delete(state.Data.Signals, m.index, m.index+1)
			if m.index == len(state.Data.Signals) {
				m.SetIndex(m.index - 1)
			}

		case "T":
			if m.index == -1 {
				return m, nil
			}
			userId := state.Data.Signals[m.index]

			_, isTrusting := state.State.TrustedUsers[userId]

			_, isBlocked := state.State.BlockedUsers[userId]
			if !isTrusting && isBlocked {
				return m, nil
			}

			return m, gateway.Send(&packet.TrustUser{
				User:  userId,
				Trust: !isTrusting,
			})

		case "b":
			if m.index == -1 {
				return m, nil
			}
			userId := state.Data.Signals[m.index]

			if userId == *state.UserID {
				return m, nil
			}

			if _, ok := state.State.BlockedUsers[userId]; ok {
				return m, nil
			}

			return m, gateway.Send(&packet.BlockUser{
				User:  userId,
				Block: true,
			})
		case "u":
			if m.index == -1 {
				return m, nil
			}
			userId := state.Data.Signals[m.index]

			if userId == *state.UserID {
				return m, nil
			}

			if _, ok := state.State.BlockedUsers[userId]; !ok {
				return m, nil
			}

			return m, gateway.Send(&packet.BlockUser{
				User:  userId,
				Block: false,
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

func (m Model) Signals() []snowflake.ID {
	return state.Data.Signals
}

func (m *Model) Index() int {
	return m.index
}

func (m *Model) SetIndex(index int) {
	m.index = min(max(index, -1), len(m.Signals())-1)
	if m.index < m.base {
		m.base = max(m.index, 0)
	} else if m.index >= m.base+m.height {
		m.base = 1 + m.index - m.height
	}
}

func (m Model) renderHeader() string {
	nameStyle := lipgloss.NewStyle().Width(m.width).
		Padding(1).Align(lipgloss.Center).
		Border(lipgloss.ThickBorder(), false, false, true).
		Background(colors.DarkerCyan).Foreground(colors.Turquoise).
		BorderForeground(colors.White)
	if m.focus {
		nameStyle = nameStyle.BorderForeground(colors.Focus)
	}
	signalsName := nameStyle.Render("User Signals")

	userIdStyle := lipgloss.NewStyle().Width(m.width).
		MarginBottom(1).Padding(1, 2).Align(lipgloss.Center).
		Border(lipgloss.ThickBorder(), false, false, true).
		BorderForeground(colors.White).Foreground(colors.White).
		Background(colors.BackgroundDim)
	if m.focus {
		userIdStyle = userIdStyle.BorderForeground(colors.Focus)
	}
	id := "Your User ID\n" + strconv.FormatInt(int64(*state.UserID), 10)
	userId := userIdStyle.Render(id)

	return lipgloss.JoinVertical(0, signalsName, userId)
}

func (m *Model) SetWidth(width int) {
	m.width = width
}

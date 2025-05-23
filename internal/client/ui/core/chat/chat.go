package chat

import (
	"bytes"
	"log"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/btree"

	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/client/ui/viminput"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

var (
	blurStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Background(colors.Background).Foreground(colors.White)
	}
	focusStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Background(colors.Background).Foreground(colors.Focus)
	}
	grayStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Background(colors.Background).Foreground(colors.Gray)
	}
	redStyle  = func() lipgloss.Style { return lipgloss.NewStyle().Background(colors.Background).Foreground(colors.Red) }
	editStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Background(colors.Background).Foreground(colors.Gold)
	}

	ViBlurredBorder = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Background(colors.Background).BorderBackground(colors.Background).
			BorderForeground(colors.White).
			Border(lipgloss.RoundedBorder(), true, true, false).
			Padding(0, 1)
	}
	ViFocusedBorder = func() lipgloss.Style { return ViBlurredBorder().BorderForeground(colors.Focus) }
	ViGrayBorder    = func() lipgloss.Style { return ViBlurredBorder().BorderForeground(colors.Gray) }
	ViRedBorder     = func() lipgloss.Style { return ViBlurredBorder().BorderForeground(colors.Red) }
	ViEditBorder    = func() lipgloss.Style { return ViBlurredBorder().BorderForeground(colors.Gold) }

	PaddingCount = 1
	Padding      = strings.Repeat(" ", PaddingCount)
	Border       = lipgloss.RoundedBorder()
	LeftCorner   = Border.BottomLeft + Border.Bottom
	RightCorner  = Border.Bottom + Border.BottomRight

	EmptyMsgs = func() lipgloss.Style {
		return lipgloss.NewStyle().Background(colors.Background).
			Foreground(colors.LightGray).Padding(0, PaddingCount, 1).
			AlignHorizontal(lipgloss.Center).AlignVertical(lipgloss.Bottom).
			SetString("This frequency has no messages, start transmiting!")
	}
	NoMessagesFrequency = func() lipgloss.Style {
		return EmptyMsgs().SetString("This frequency has no messages, start transmiting!")
	}
	NoMessagesSignal = func() lipgloss.Style { return EmptyMsgs().SetString("This signal has no messages, start transmiting!") }
	NothingSelected  = func() lipgloss.Style { return EmptyMsgs().SetString("You haven't seelcted a signal or frequency yet!") }
	NoAccess         = func() lipgloss.Style {
		return EmptyMsgs().SetString("You do not have permission to see messages in this frequency")
	}

	SendMessagePlaceholder  = "Send a message..."
	ReadOnlyPlaceholder     = "You do not have permission to send messages in this frequency"
	MutedPlaceholder        = "You have been muted by a network adminstrator"
	SelectSignalOrFrequency = "Cannot send messages, select a signal or frequency first"
	BlockedPlaceholder      = "You have blocked this user"
	BlockingPlaceholder     = "You have been blocked by this user"

	EditedIndicator = func(bg lipgloss.Color) string {
		return lipgloss.NewStyle().Background(bg).
			Foreground(colors.LightGray).SetString(" (edited)").String()
	}
	EditedIndicatorNL = func(bg lipgloss.Color) string {
		return lipgloss.NewStyle().Background(bg).
			Foreground(colors.LightGray).SetString(" (edited)").String()
	}

	WidthWithoutVi = PaddingCount*2 + lipgloss.Width(LeftCorner) + lipgloss.Width(RightCorner)

	MutedSymbol = func() string { return lipgloss.NewStyle().Foreground(colors.Red).Render(" 󱡣") }

	PingPrefix      = "@"
	PingedAdmins    = func() string { return lipgloss.NewStyle().Foreground(colors.Red).SetString("@admins ").String() }
	PingedEveryone  = func() string { return lipgloss.NewStyle().Foreground(colors.Purple).Render("@everyone ") }
	PingedUserStyle = func() lipgloss.Style { return lipgloss.NewStyle().Foreground(colors.Gold) }

	NewText   = "━━ NEW ━━"
	NewSymbol = "━"
)

const (
	MaxCharCount        = 2000
	MaxViewableMessages = 200

	TimeGap = 7 * 60 * 1000 // 7 minutes in millis

	SnapToBottom = -1
	Unselected   = -1
)

type Model struct {
	vi     viminput.Model
	focus  bool
	locked bool

	hasReadAccess  bool
	hasWriteAccess bool
	networkIndex   int
	receiverIndex  int
	frequencyIndex int

	base            int
	index           int
	selectedMessage *data.Message
	editingMessage  *data.Message

	previousLastReadMsg  *snowflake.ID
	keepPreviousLastRead bool

	messagesHeight    int
	maxMessagesHeight int
	messagesCache     *string
	prerender         string

	width int

	style       lipgloss.Style
	borderStyle lipgloss.Style
}

func New() Model {
	vi := viminput.New()

	return Model{
		vi:                   vi,
		focus:                false,
		locked:               false,
		hasReadAccess:        false,
		hasWriteAccess:       false,
		networkIndex:         -1,
		receiverIndex:        -1,
		frequencyIndex:       -1,
		base:                 SnapToBottom,
		index:                Unselected,
		selectedMessage:      nil,
		editingMessage:       nil,
		previousLastReadMsg:  nil,
		keepPreviousLastRead: false,
		messagesHeight:       0,
		maxMessagesHeight:    -1,
		messagesCache:        nil,
		prerender:            "",
		width:                -1,
		style:                blurStyle(),
		borderStyle:          ViBlurredBorder(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Prerender() {
	frequencyName := m.renderFrequencyName()
	frequencyHeight := lipgloss.Height(frequencyName)

	messagebox := m.renderMessageBox()
	messageBoxHeight := lipgloss.Height(messagebox) - 1

	messagesHeight := ui.Height - messageBoxHeight - frequencyHeight

	// if m.messagesCache == nil || messagesHeight != m.messagesHeight {
	// 	// Re-render
	// }
	m.selectedMessage = nil
	messages := m.renderMessages(messagesHeight)
	m.messagesCache = &messages
	m.messagesHeight = messagesHeight

	m.prerender = lipgloss.NewStyle().
		Background(colors.Background).
		Render(frequencyName + *m.messagesCache + messagebox)
}

func (m Model) View() string {
	return m.prerender
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	// TODO: properly invalidate cache
	m.messagesCache = nil

	viWidth := m.width - WidthWithoutVi
	m.vi.SetWidth(viWidth)
	m.vi.SetMaxHeight(ui.Height / 2)

	networkId := state.NetworkId(m.networkIndex)
	if m.frequencyIndex != -1 && networkId != nil {
		frequencies := state.State.Frequencies[*networkId]
		frequency := frequencies[m.frequencyIndex]
		member := state.State.Members[*networkId][*state.UserID]

		if s, ok := state.State.ChatState[frequency.ID]; ok {
			m.maxMessagesHeight = s.MaxHeight
		}

		lastMsg := state.GetLastMessage(frequency.ID)
		if lastMsg != nil {
			if m.base == SnapToBottom && lastMsg != nil {
				// lastReadMsg := state.State.LastReadMessages[frequency.ID]
				// if m.keepPreviousLastRead && m.previousLastReadMsg != nil && lastReadMsg != nil &&
				// 	*lastReadMsg != *lastMsg && *lastReadMsg != *m.previousLastReadMsg {
				// 	m.keepPreviousLastRead = false
				// }

				log.Println("HIT")
				state.State.LastReadMessages[frequency.ID] = lastMsg
				delete(state.State.RemoteNotifications, frequency.ID)
			} else if m.base == SnapToBottom {
				assert.Never("lastMsg is nil")
			}
			if m.base != SnapToBottom {
				assert.Assert(
					m.keepPreviousLastRead, "debug reached m.keep",
					"m.base", m.base,
					"lastMsg", lastMsg,
					"lastReadMessage", state.State.LastReadMessages[frequency.ID],
					"m.previousLastReadMessage", m.previousLastReadMsg,
				)
				m.keepPreviousLastRead = false
			}
		}

		m.hasReadAccess = frequency.Perms != packet.PermNoAccess || member.IsAdmin
		m.hasWriteAccess = !member.IsMuted && (frequency.Perms == packet.PermReadWrite || member.IsAdmin)

		if member.IsMuted {
			m.vi.Placeholder = MutedPlaceholder
			m.borderStyle = ViRedBorder()
			m.style = redStyle()
			m.vi.SetInactive(true)
			m.locked = false
		} else if frequency.Perms != packet.PermReadWrite && !member.IsAdmin {
			m.vi.Placeholder = ReadOnlyPlaceholder
			m.borderStyle = ViGrayBorder()
			m.style = grayStyle()
			m.vi.SetInactive(true)
			m.locked = false
		} else if m.locked {
			m.vi.Placeholder = SendMessagePlaceholder
			m.borderStyle = ViFocusedBorder()
			m.style = focusStyle()
			m.vi.SetInactive(false)
			m.vi.Focus()
			if m.editingMessage != nil {
				m.borderStyle = ViEditBorder()
				m.style = editStyle()
			}
		} else {
			m.vi.Placeholder = SendMessagePlaceholder
			m.borderStyle = ViBlurredBorder()
			m.style = blurStyle()
			m.vi.SetInactive(false)
		}
	} else if m.receiverIndex != -1 {
		receiverId := state.Data.Signals[m.receiverIndex]

		if s, ok := state.State.ChatState[receiverId]; ok {
			m.maxMessagesHeight = s.MaxHeight
		}

		lastMsg := state.GetLastMessage(receiverId)
		if m.base != SnapToBottom {
			m.keepPreviousLastRead = false
		} else if lastMsg != nil {
			lastReadMsg := state.State.LastReadMessages[receiverId]
			if m.keepPreviousLastRead && m.previousLastReadMsg != nil && lastReadMsg != nil &&
				*lastReadMsg != *lastMsg && *lastReadMsg != *m.previousLastReadMsg {
				m.keepPreviousLastRead = false
			}

			state.State.LastReadMessages[receiverId] = lastMsg
		}

		m.hasReadAccess = true
		m.hasWriteAccess = true

		if _, ok := state.State.BlockedUsers[receiverId]; ok {
			m.hasWriteAccess = false
			m.vi.Placeholder = BlockedPlaceholder
			m.borderStyle = ViRedBorder()
			m.style = redStyle()
			m.vi.SetInactive(true)
			m.locked = false
		} else if _, ok := state.State.BlockingUsers[receiverId]; ok {
			m.hasWriteAccess = false
			m.vi.Placeholder = BlockingPlaceholder
			m.borderStyle = ViRedBorder()
			m.style = redStyle()
			m.vi.SetInactive(true)
			m.locked = false
		} else if m.locked {
			m.vi.Placeholder = SendMessagePlaceholder
			m.borderStyle = ViFocusedBorder()
			m.style = focusStyle()
			m.vi.SetInactive(false)
			m.vi.Focus()
			if m.editingMessage != nil {
				m.borderStyle = ViEditBorder()
				m.style = editStyle()
			}
		} else {
			m.vi.Placeholder = SendMessagePlaceholder
			m.borderStyle = ViBlurredBorder()
			m.style = blurStyle()
			m.vi.SetInactive(false)
		}
	} else {
		// On signal, but nothing is selected
		m.hasReadAccess = true
		m.hasWriteAccess = false // TODO: implement blocking users

		m.vi.Placeholder = SelectSignalOrFrequency
		m.borderStyle = ViGrayBorder()
		m.style = grayStyle()
		m.vi.SetInactive(true)
	}

	if !m.focus {
		m.Prerender()
		return m, nil
	}

	if m.locked {
		if key, ok := msg.(tea.KeyMsg); ok {
			inNormalQ := key.String() == "q" && m.vi.Mode() == viminput.NormalMode
			inInsertCtrlQ := key.String() == "ctrl+q" && m.vi.Mode() == viminput.InsertMode

			if inNormalQ || inInsertCtrlQ {
				m.locked = false
				m.borderStyle = ViBlurredBorder()
				m.style = blurStyle()

				if m.editingMessage != nil {
					m.editingMessage = nil
					m.vi.Reset()
				}

				m.vi, _ = m.vi.Update(msg)
				m.Prerender()
				return m, nil
			}

			if key.Type == tea.KeyEnter {
				var cmd tea.Cmd

				if m.editingMessage != nil {
					cmd = m.editMessage()
					m.locked = false
					m.editingMessage = nil

					m.borderStyle = ViBlurredBorder()
					m.style = blurStyle()
					m.vi.Reset()
				} else {
					cmd = m.sendMessage()
				}

				m.vi, _ = m.vi.Update(msg)
				m.Prerender()
				return m, cmd
			}
		}

		var cmd tea.Cmd
		m.vi, cmd = m.vi.Update(msg)
		m.Prerender()
		return m, cmd
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "i":
			if !m.vi.Inactive() {
				m.borderStyle = ViFocusedBorder()
				m.style = focusStyle()
				m.locked = true
				m.vi.SetMode(viminput.InsertMode)
				m.SetIndex(Unselected)
			}
		case "k":
			m.Scroll(1)
		case "j":
			m.Scroll(-1)
		case "ctrl+u":
			m.Scroll(m.messagesHeight / 2)
		case "ctrl+d":
			m.Scroll(-m.messagesHeight / 2)
		case "g":
			if m.maxMessagesHeight != -1 {
				m.SetIndex(m.maxMessagesHeight)
			}
		case "G":
			m.base = SnapToBottom
			m.SetIndex(Unselected)
		case "enter":
			if !m.vi.Inactive() {
				m.borderStyle = ViFocusedBorder()
				m.style = focusStyle()
				m.locked = true
				m.vi.SetMode(viminput.InsertMode)
				m.base = SnapToBottom
				m.SetIndex(Unselected)
			}

		case "p":
			if m.selectedMessage == nil {
				return m, nil
			}

			return m, func() tea.Msg {
				return ui.ProfilePopupMsg{
					User: m.selectedMessage.SenderID,
				}
			}

		case "x", "d":
			if m.selectedMessage == nil {
				return m, nil
			}
			log.Println("deleting message:", m.selectedMessage)
			cmd = gateway.Send(&packet.DeleteMessage{
				Message: m.selectedMessage.ID,
			})

		case "e":
			if m.selectedMessage == nil {
				return m, nil
			}
			if m.selectedMessage.SenderID != *state.UserID {
				return m, nil
			}

			log.Println("editing message:", m.selectedMessage)
			m.editingMessage = m.selectedMessage

			m.vi.Reset()
			m.vi.SetString(m.selectedMessage.Content)

			m.locked = true
			m.vi.SetMode(viminput.InsertMode)
			m.vi.SetCursorLine(len(m.vi.Lines()) - 1)
			m.vi.SetCursorColumn(len(m.vi.Lines()[m.vi.CursorLine()]))

			m.borderStyle = ViEditBorder()
			m.style = editStyle()

		case "T":
			if m.selectedMessage == nil {
				return m, nil
			}

			senderId := m.selectedMessage.SenderID

			if senderId == *state.UserID {
				return m, nil
			}

			_, isTrusting := state.State.TrustedUsers[senderId]

			_, isBlocked := state.State.BlockedUsers[senderId]
			if !isTrusting && isBlocked {
				return m, nil
			}

			return m, gateway.Send(&packet.TrustUser{
				User:  senderId,
				Trust: !isTrusting,
			})

		case "b":
			if m.selectedMessage == nil {
				return m, nil
			}

			userId := m.selectedMessage.SenderID

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
			if m.selectedMessage == nil {
				return m, nil
			}

			userId := m.selectedMessage.SenderID

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

		// Admin
		case "K":
			if m.selectedMessage == nil || m.receiverIndex != -1 {
				return m, nil
			}
			networkId := state.NetworkId(m.networkIndex)
			network := state.State.Networks[*networkId]
			senderId := m.selectedMessage.SenderID
			member := state.State.Members[*networkId][senderId]

			if !state.State.Members[*networkId][*state.UserID].IsAdmin {
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
			if m.selectedMessage == nil || m.receiverIndex != -1 {
				return m, nil
			}
			networkId := state.NetworkId(m.networkIndex)
			network := state.State.Networks[*networkId]
			senderId := m.selectedMessage.SenderID
			member := state.State.Members[*networkId][senderId]

			if member.IsMuted {
				return m, nil
			}

			if !state.State.Members[*networkId][*state.UserID].IsAdmin {
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
			if m.selectedMessage == nil || m.receiverIndex != -1 {
				return m, nil
			}
			networkId := state.NetworkId(m.networkIndex)
			network := state.State.Networks[*networkId]
			senderId := m.selectedMessage.SenderID
			member := state.State.Members[*networkId][senderId]

			if !member.IsMuted {
				return m, nil
			}

			if !state.State.Members[*networkId][*state.UserID].IsAdmin {
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
			if m.selectedMessage == nil || m.receiverIndex != -1 {
				return m, nil
			}
			networkId := state.NetworkId(m.networkIndex)
			network := state.State.Networks[*networkId]
			senderId := m.selectedMessage.SenderID
			member := state.State.Members[*networkId][senderId]

			if !state.State.Members[*networkId][*state.UserID].IsAdmin {
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
			if m.selectedMessage == nil || m.receiverIndex != -1 {
				return m, nil
			}

			networkId := state.NetworkId(m.networkIndex)
			network := state.State.Networks[*networkId]
			senderId := m.selectedMessage.SenderID
			member := state.State.Members[*networkId][senderId]

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
			if m.selectedMessage == nil || m.receiverIndex != -1 {
				return m, nil
			}
			networkId := state.NetworkId(m.networkIndex)
			network := state.State.Networks[*networkId]
			senderId := m.selectedMessage.SenderID
			member := state.State.Members[*networkId][senderId]

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

	m.Prerender()
	return m, cmd
}

func (m *Model) Focus() {
	m.focus = true
	m.vi.Focus()
}

func (m *Model) Blur() {
	m.focus = false
	m.SetIndex(Unselected)
	m.vi.Blur()
}

func (m Model) Locked() bool {
	return m.locked
}

func (m *Model) sendMessage() tea.Cmd {
	message := m.vi.String()

	var ping *snowflake.ID = nil

	var receiverId *snowflake.ID = nil
	if m.receiverIndex != -1 {
		receiverId = &state.Data.Signals[m.receiverIndex]
	}

	var frequencyId *snowflake.ID = nil
	networkId := state.NetworkId(m.networkIndex)
	if m.frequencyIndex != -1 && networkId != nil {
		frequencies := state.State.Frequencies[*networkId]
		frequencyId = &frequencies[m.frequencyIndex].ID

		// Parse ping
		if strings.HasPrefix(message, PingPrefix) {
			index := strings.IndexAny(message, "\n ")
			if index != -1 {
				value := message[len(PingPrefix):index]
				switch value {
				case "everyone":
					member := state.State.Members[*networkId][*state.UserID]
					if member.IsAdmin {
						pingValue := packet.PingEveryone
						ping = &pingValue
						message = message[index+1:]
					}
				case "admins":
					pingValue := packet.PingAdmins
					ping = &pingValue
					message = message[index+1:]
				default:
					num, err := strconv.ParseInt(value, 10, 64)
					if err == nil {
						ping = (*snowflake.ID)(&num)
						message = message[index+1:]
					}
				}
			}
		}
	}

	if len(message) > MaxCharCount {
		return nil
	}
	if len(strings.TrimSpace(message)) == 0 {
		return nil
	}

	m.vi.Reset()
	m.base = SnapToBottom

	return gateway.Send(&packet.SendMessage{
		ReceiverID:  receiverId,
		FrequencyID: frequencyId,
		Content:     message,
		Ping:        ping,
	})
}

func (m *Model) SetReceiver(receiverIndex int) tea.Cmd {
	if m.receiverIndex == receiverIndex && m.frequencyIndex == -1 {
		return nil
	}
	m.ResetBeforeSwitch()
	m.receiverIndex = receiverIndex
	m.frequencyIndex = -1
	m.networkIndex = -1
	return m.RestoreAfterSwitch()
}

func (m *Model) SetFrequency(networkIndex, frequencyIndex int) tea.Cmd {
	if m.frequencyIndex == frequencyIndex && m.networkIndex == networkIndex {
		return nil
	}
	m.ResetBeforeSwitch()
	m.receiverIndex = -1
	m.frequencyIndex = frequencyIndex
	m.networkIndex = networkIndex
	return m.RestoreAfterSwitch()
}

func (m *Model) ResetBeforeSwitch() {
	defer func() {
		m.vi.Reset()
		m.base = SnapToBottom
		m.SetIndex(Unselected)
		m.maxMessagesHeight = -1
		m.previousLastReadMsg = nil
		m.keepPreviousLastRead = false
	}()

	networkId := state.NetworkId(m.networkIndex)
	if m.frequencyIndex != -1 && networkId != nil {
		frequencies := state.State.Frequencies[*networkId]
		if len(frequencies) <= m.frequencyIndex {
			return
		}
		frequencyId := frequencies[m.frequencyIndex].ID
		log.Println("Saving frequency", frequencyId)
		state.State.ChatState[frequencyId] = state.ChatState{
			IncompleteMessage: m.vi.String(),
			Base:              m.base,
			MaxHeight:         m.maxMessagesHeight,
		}

	} else if m.receiverIndex != -1 {
		receiverId := state.Data.Signals[m.receiverIndex]
		log.Println("Saving signal:", receiverId)
		state.State.ChatState[receiverId] = state.ChatState{
			IncompleteMessage: m.vi.String(),
			Base:              m.base,
			MaxHeight:         m.maxMessagesHeight,
		}
	}
}

func (m *Model) RestoreAfterSwitch() tea.Cmd {
	msgs := state.State.ChatState
	networkId := state.NetworkId(m.networkIndex)
	if m.frequencyIndex != -1 && networkId != nil {
		frequencies := state.State.Frequencies[*networkId]
		frequency := frequencies[m.frequencyIndex]
		log.Println("Restoring frequency:", frequency.ID)

		m.previousLastReadMsg = state.State.LastReadMessages[frequency.ID]
		lastMsg := state.GetLastMessage(frequency.ID)
		if m.previousLastReadMsg != nil && lastMsg != nil && *m.previousLastReadMsg != *lastMsg {
			m.keepPreviousLastRead = true
		}

		if val, ok := msgs[frequency.ID]; ok {
			m.vi.SetString(val.IncompleteMessage)
			m.base = val.Base
			m.SetIndex(Unselected)
			m.maxMessagesHeight = val.MaxHeight

			// Don't ask for messages if you already visited this frequency
			return nil
		}

		log.Println("Requesting frequency messages")
		return gateway.Send(&packet.RequestMessages{
			ReceiverID:  nil,
			FrequencyID: &frequency.ID,
		})
	} else if m.receiverIndex != -1 {
		receiverId := state.Data.Signals[m.receiverIndex]
		log.Println("Restoring signal:", receiverId)

		m.previousLastReadMsg = state.State.LastReadMessages[receiverId]
		lastMsg := state.GetLastMessage(receiverId)
		if m.previousLastReadMsg != nil && lastMsg != nil && *m.previousLastReadMsg != *lastMsg {
			m.keepPreviousLastRead = true
		}

		if val, ok := msgs[receiverId]; ok {
			m.vi.SetString(val.IncompleteMessage)
			m.base = val.Base
			m.SetIndex(Unselected)
			m.maxMessagesHeight = val.MaxHeight

			// Don't ask for messages if you already visited this frequency
			return nil
		}

		log.Println("Requesting signal messages:", receiverId)
		return gateway.Send(&packet.RequestMessages{
			ReceiverID:  &receiverId,
			FrequencyID: nil,
		})
	}

	return nil
}

func (m *Model) renderMessageBox() string {
	var builder strings.Builder

	input := m.borderStyle.Render(m.vi.View())
	builder.WriteString(input)
	builder.WriteByte('\n')

	width := lipgloss.Width(input)

	leftAngle := m.style.Render("")
	rightAngle := m.style.Render("")

	builder.WriteString(m.style.Render(LeftCorner))
	width -= lipgloss.Width(LeftCorner)

	builder.WriteString(leftAngle)
	width -= lipgloss.Width(leftAngle)
	vimModeStyle := lipgloss.NewStyle().Bold(true).Background(colors.Background)
	mode := vimModeStyle.Foreground(colors.Gray).Render("  NONE ")
	if m.locked {
		switch m.vi.Mode() {
		case viminput.InsertMode:
			mode = vimModeStyle.Foreground(colors.Green).Render("  INSERT ")
		case viminput.NormalMode:
			mode = vimModeStyle.Foreground(colors.Orange).Render("  NORMAL ")
		case viminput.OpendingMode:
			mode = vimModeStyle.Foreground(colors.Red).Render("  O-PENDING ")
		case viminput.VisualMode:
			mode = vimModeStyle.Foreground(colors.Turquoise).Render("  VISUAL ")
		case viminput.VisualLineMode:
			mode = vimModeStyle.Foreground(colors.Turquoise).Render("  V-LINE ")
		}
	}
	builder.WriteString(mode)
	width -= lipgloss.Width(mode)
	builder.WriteString(rightAngle)
	width -= lipgloss.Width(rightAngle)

	width -= lipgloss.Width(leftAngle)
	count := m.vi.Count()
	countStr := " " + strconv.Itoa(count)
	if count > MaxCharCount {
		countStr = lipgloss.NewStyle().Foreground(colors.Red).Render(countStr)
	} else if m.vi.Inactive() {
		countStr = lipgloss.NewStyle().Foreground(colors.Gray).Render(countStr)
	} else {
		countStr = lipgloss.NewStyle().Foreground(colors.White).Render(countStr)
	}
	countStyle := lipgloss.NewStyle().Background(colors.Background)
	if m.vi.Inactive() {
		countStyle = countStyle.Foreground(colors.Gray)
	} else {
		countStyle = countStyle.Foreground(colors.White)
	}
	countStr += countStyle.Render(" / " + strconv.Itoa(MaxCharCount) + " ")
	width -= lipgloss.Width(countStr)
	width -= lipgloss.Width(rightAngle)

	width -= lipgloss.Width(RightCorner)

	bottomCount := width / lipgloss.Width(Border.Bottom)
	bottom := strings.Repeat(Border.Bottom, bottomCount)
	builder.WriteString(m.style.Render(bottom))

	backgroundStyle := lipgloss.NewStyle().Background(colors.Background)
	builder.WriteString(leftAngle)
	builder.WriteString(backgroundStyle.Render(countStr))
	builder.WriteString(rightAngle)
	builder.WriteString(m.style.Render(RightCorner))
	builder.WriteByte('\n')

	result := builder.String()

	return lipgloss.NewStyle().Padding(0, PaddingCount).
		Background(colors.Background).Render(result)
}

func (m *Model) renderMessages(screenHeight int) string {
	if !m.hasReadAccess {
		return NoAccess().Width(m.width).Height(screenHeight).String() + "\n"
	}

	var id *snowflake.ID
	var btree *btree.BTreeG[data.Message]
	networkId := state.NetworkId(m.networkIndex)
	if m.frequencyIndex != -1 && networkId != nil {
		frequencies := state.State.Frequencies[*networkId]
		frequencyId := frequencies[m.frequencyIndex].ID
		id = &frequencyId
		btree = state.State.Messages[frequencyId]
	} else if m.receiverIndex != -1 {
		receiverId := state.Data.Signals[m.receiverIndex]
		id = &receiverId
		btree = state.State.Messages[receiverId]
	}

	if btree == nil || btree.Len() == 0 || id == nil {
		if m.frequencyIndex != -1 {
			return NoMessagesFrequency().Width(m.width).Height(screenHeight).String() + "\n"
		} else if m.receiverIndex != -1 {
			return NoMessagesSignal().Width(m.width).Height(screenHeight).String() + "\n"
		} else {
			return NothingSelected().Width(m.width).Height(screenHeight).String() + "\n"
		}
	}

	height := screenHeight
	if m.base != SnapToBottom {
		height = m.base + 1
		// bcz base is an index (0 to n) but we want total
		// we need to add one so it's a height
	}
	remainingHeight := height

	renderedGroups := []string{}
	group := []data.Message{}

	lastReadId := state.State.LastReadMessages[*id]
	if m.keepPreviousLastRead {
		lastReadId = m.previousLastReadMsg
	}

	last := snowflake.ID(0)
	btree.Descend(func(message data.Message) bool {
		last = message.ID
		// >= is needed so if the message was deleted
		// Any prior messages will still be separated by "new"
		isLastRead := lastReadId != nil && *lastReadId >= message.ID

		if len(group) == 0 {
			group = append(group, message)

			if isLastRead {
				lastReadId = nil
			}

			return true
		}

		lastMsg := group[0]
		sameSender := lastMsg.SenderID == message.SenderID
		withinTime := lastMsg.ID.Time()-message.ID.Time() <= TimeGap
		if sameSender && withinTime && len(group) < MaxViewableMessages && !isLastRead {
			group = append(group, message)
			return true
		}

		renderedGroup := m.renderMessageGroup(group, &remainingHeight, height)
		renderedGroups = append(renderedGroups, renderedGroup)
		group = []data.Message{message}

		if isLastRead {
			lineWidth := m.width - lipgloss.Width(NewText)
			line := strings.Repeat(NewSymbol, lineWidth)

			newStyle := lipgloss.NewStyle().Foreground(colors.Red).Width(m.width)
			if m.index == height-remainingHeight {
				newStyle = newStyle.Background(colors.BackgroundDim)
			}

			newSep := newStyle.Render(line + NewText)

			renderedGroups = append(renderedGroups, newSep+"\n")
			remainingHeight--

			lastReadId = nil // Stop the rest of the messages from having NEW
		}

		return remainingHeight > 0
	})

	// We ran out of messages, so let's render the last group
	if remainingHeight > 0 && len(group) != 0 {
		renderedGroup := m.renderMessageGroup(group, &remainingHeight, height)
		renderedGroups = append(renderedGroups, renderedGroup)
	}

	first, ok := btree.Min()
	if ok && last == first.ID {
		m.maxMessagesHeight = height - remainingHeight
		m.SetIndex(m.index)
		if s, ok := state.State.ChatState[*id]; ok {
			s.MaxHeight = m.maxMessagesHeight
			state.State.ChatState[*id] = s
		} else {
			state.State.ChatState[*id] = state.ChatState{
				MaxHeight: m.maxMessagesHeight,
			}
		}
	}

	var builder strings.Builder

	// Add blank newline to fill any remaining height
	for i := 0; i < remainingHeight; i++ {
		builder.WriteByte('\n')
	}

	for i := len(renderedGroups) - 1; i >= 0; i-- {
		builder.WriteString(renderedGroups[i])
	}

	result := builder.String()

	if m.base == SnapToBottom {
		// Truncate any excess and show only the bottom
		newlines := 0
		index := -1
		for i := len(result) - 1; i >= 0; i-- {
			if result[i] == '\n' {
				newlines++
			}
			// The reason for this +1 is bcz height gives the newline of the
			// first line, so we go an extra one to get the newline BEFORE
			// the first line and then we trim to result[index+1:]
			// which prints the first char of the first line onwards
			if newlines == height+1 {
				index = i
				break
			}
		}
		// If index wasn't found, then -1+1 will be 0
		// which is the desired value
		return result[index+1:]
	} else {
		// Show from the offset up to offset+height
		newlines := 0
		baseIndex := -1
		upToIndex := -1
		for i := len(result) - 1; i >= 0; i-- {
			if result[i] == '\n' {
				newlines++
			}
			// The reason for this +2 is bcz base needs to be adjusted to
			// a height rather than an index and after the adjustment,
			// base gives the newline of the first line we want,
			// so we go an extra +1 to get the newline BEFORE
			// the first line and then we trim to result[baseIndex+1:...]
			// which prints the first char of the first line onwards
			if newlines == m.base+2 && baseIndex == -1 {
				baseIndex = i
			}
			// The first +1 is for adjusting the height and the 2nd +1
			// is for the same reason as the previous comment
			if newlines == m.base-screenHeight+1+1 && upToIndex == -1 {
				upToIndex = i
			}
			if baseIndex != -1 && upToIndex != -1 {
				break
			}
		}
		if upToIndex != -1 {
			// If baseIndex wasn't found, then -1+1 will be 0
			// which is the desired value
			return result[baseIndex+1 : upToIndex+1]
		}
		assert.Never("unreachable",
			"base", m.base,
			"index", m.index,
			"baseIndex", baseIndex,
			"upToIndex", upToIndex,
			"messageHeight", m.messagesHeight,
		)
		return ""
	}
}

func (m *Model) renderMessageGroup(group []data.Message, remaining *int, height int) string {
	assert.Assert(len(group) != 0, "cannot render a group with length 0")

	firstMsg := group[len(group)-1]
	buf := m.renderHeader(firstMsg, false)

	heights := make([]int, len(group))
	checkpoints := make([]int, len(group))

	// Render all messages content

	messageStyle := lipgloss.NewStyle().Width(m.width).
		Background(colors.Background).Foreground(colors.White).
		PaddingLeft(PaddingCount + 2).PaddingRight(PaddingCount)

	pingedMessageStyle := lipgloss.NewStyle().Width(m.width-2).
		MarginLeft(PaddingCount).PaddingLeft(1).PaddingRight(PaddingCount).
		Border(lipgloss.Border{Left: "┃"}, false, false, false, true)

	for i := len(group) - 1; i >= 0; i-- {
		extra := ""
		messageStyle := messageStyle
		backgroundStyle := lipgloss.NewStyle().Background(colors.Background).Foreground(colors.White)
		if m.frequencyIndex != -1 && group[i].Ping != nil {
			members := state.State.Members[*state.NetworkId(m.networkIndex)]
			switch *group[i].Ping {
			case packet.PingEveryone:
				extra = PingedEveryone()
				messageStyle = pingedMessageStyle.
					BorderForeground(colors.Purple).
					Background(colors.MutedPurple)
				backgroundStyle = backgroundStyle.Background(colors.MutedPurple)
			case packet.PingAdmins:
				extra = PingedAdmins()
				if members[*state.UserID].IsAdmin {
					messageStyle = pingedMessageStyle.
						BorderForeground(colors.Red).
						Background(colors.MutedRed)
					backgroundStyle = backgroundStyle.Background(colors.MutedRed)
				}
			default:
				name := "@Unknown "
				if user, ok := state.State.Users[*group[i].Ping]; ok {
					name = "@" + user.Name + " "
				}
				extra = PingedUserStyle().Render(name)
				if *group[i].Ping == *state.UserID {
					messageStyle = pingedMessageStyle.
						BorderForeground(colors.Gold).
						Background(colors.MutedGold)
					backgroundStyle = backgroundStyle.Background(colors.MutedGold)
				}
			}
		}

		if _, ok := state.State.BlockedUsers[group[i].SenderID]; ok {
			extra = ""
			messageStyle = pingedMessageStyle.
				BorderForeground(colors.Gray).
				Background(colors.DarkGray)
			backgroundStyle = backgroundStyle.Background(colors.DarkGray).Foreground(colors.DarkGray)
		}

		rawContent := extra + backgroundStyle.Render(group[i].Content)
		content := messageStyle.Render(rawContent)
		heights[i] = lipgloss.Height(content)
		if group[i].Edited {
			color := backgroundStyle.GetBackground().(lipgloss.Color)
			before := heights[i]
			content = messageStyle.Render(rawContent + EditedIndicator(color))
			heights[i] = lipgloss.Height(content)
			if before != heights[i] {
				content = messageStyle.Render(rawContent + EditedIndicatorNL(color))
				heights[i] = lipgloss.Height(content)
			}
		}

		checkpoints[i] = len(buf)

		buf = append(buf, content...)
		buf = append(buf, '\n')
	}

	// Gap between each message group
	if m.index == height-*remaining {
		gap := lipgloss.NewStyle().Background(colors.BackgroundDim).Width(m.width).String()
		buf = append(buf, gap...)
	}
	buf = append(buf, '\n')
	*remaining--

	selectedIndex := -1
	for i, h := range heights {
		bottom := height - *remaining
		top := bottom + h
		*remaining -= h
		if bottom <= m.index && m.index <= top {
			selectedIndex = i
		}
	}
	*remaining-- // For the header

	if selectedIndex != -1 {
		m.selectedMessage = &group[selectedIndex]

		if selectedIndex == len(group)-1 {
			buf = m.renderHeader(group[selectedIndex], true)
		} else {
			buf = buf[:checkpoints[selectedIndex]] // Revert
		}

		selectedStyle := messageStyle.Background(colors.BackgroundDim)
		selectedPingedStyle := pingedMessageStyle.Background(colors.BackgroundDim)
		selectedBackgroundStyle := lipgloss.NewStyle().Background(colors.BackgroundDim).Foreground(colors.White)

		extra := ""
		if m.frequencyIndex != -1 && group[selectedIndex].Ping != nil {
			members := state.State.Members[*state.NetworkId(m.networkIndex)]
			switch *group[selectedIndex].Ping {
			case packet.PingEveryone:
				extra = PingedEveryone()
				selectedStyle = selectedPingedStyle.
					BorderForeground(colors.Purple).
					Background(colors.DarkMutedPurple)
				selectedBackgroundStyle = selectedBackgroundStyle.Background(colors.DarkMutedPurple)
			case packet.PingAdmins:
				extra = PingedAdmins()
				if members[*state.UserID].IsAdmin {
					selectedStyle = selectedPingedStyle.
						BorderForeground(colors.Red).
						Background(colors.DarkMutedRed)
					selectedBackgroundStyle = selectedBackgroundStyle.Background(colors.DarkMutedRed)
				}
			default:
				name := "@Unknown "
				if user, ok := state.State.Users[*group[selectedIndex].Ping]; ok {
					name = "@" + user.Name + " "
				}
				extra = PingedUserStyle().Render(name)
				if *group[selectedIndex].Ping == *state.UserID {
					selectedStyle = selectedPingedStyle.
						BorderForeground(colors.Gold).
						Background(colors.DarkMutedGold)
					selectedBackgroundStyle = selectedBackgroundStyle.Background(colors.DarkMutedGold)
				}
			}
		}

		rawContent := group[selectedIndex].Content
		rawContent = selectedBackgroundStyle.Render(rawContent)
		rawContent = extra + rawContent
		content := selectedStyle.Render(rawContent)
		if group[selectedIndex].Edited {
			color := selectedBackgroundStyle.GetBackground().(lipgloss.Color)
			before := lipgloss.Height(content)
			content = selectedStyle.Render(rawContent + EditedIndicator(color))
			after := lipgloss.Height(content)
			if before != after {
				content = selectedStyle.Render(rawContent + EditedIndicatorNL(color))
			}
		}

		buf = append(buf, content...)
		buf = append(buf, '\n')

		// Redraw rest
		for i := selectedIndex - 1; i >= 0; i-- {
			extra := ""
			messageStyle := messageStyle
			backgroundStyle := lipgloss.NewStyle().Background(colors.Background).Foreground(colors.White)
			if m.frequencyIndex != -1 && group[i].Ping != nil {

				members := state.State.Members[*state.NetworkId(m.networkIndex)]
				switch *group[i].Ping {
				case packet.PingEveryone:
					extra = PingedEveryone()
					messageStyle = pingedMessageStyle.
						BorderForeground(colors.Purple).
						Background(colors.MutedPurple)
					backgroundStyle = backgroundStyle.Background(colors.MutedPurple)
				case packet.PingAdmins:
					extra = PingedAdmins()
					if members[*state.UserID].IsAdmin {
						messageStyle = pingedMessageStyle.
							BorderForeground(colors.Red).
							Background(colors.MutedRed)
						backgroundStyle = backgroundStyle.Background(colors.MutedRed)
					}
				default:
					name := "@Unknown "
					if user, ok := state.State.Users[*group[i].Ping]; ok {
						name = "@" + user.Name + " "
					}
					extra = PingedUserStyle().Render(name)
					if *group[i].Ping == *state.UserID {
						messageStyle = pingedMessageStyle.
							BorderForeground(colors.Gold).
							Background(colors.MutedGold)
						backgroundStyle = backgroundStyle.Background(colors.MutedGold)
					}
				}
			}

			rawContent := extra + backgroundStyle.Render(group[i].Content)
			content := messageStyle.Render(rawContent)
			heights[i] = lipgloss.Height(content)
			if group[i].Edited {
				color := backgroundStyle.GetBackground().(lipgloss.Color)
				before := heights[i]
				content = messageStyle.Render(rawContent + EditedIndicator(color))
				heights[i] = lipgloss.Height(content)
				if before != heights[i] {
					content = messageStyle.Render(rawContent + EditedIndicatorNL(color))
					heights[i] = lipgloss.Height(content)
				}
			}

			buf = append(buf, content...)
			buf = append(buf, '\n')
		}
		buf = append(buf, '\n') // Gap between each message group
	}

	return string(buf)
}

func (m *Model) renderHeader(message data.Message, selected bool) []byte {
	var buf []byte
	buf = append(buf, Padding...)

	blockStyle := lipgloss.NewStyle().Background(colors.Background)
	if selected {
		blockStyle = blockStyle.Background(colors.BackgroundDim)
	}

	networkId := state.NetworkId(m.networkIndex)
	if networkId != nil {
		ownerId := state.State.Networks[*networkId].OwnerID
		members := state.State.Members[*networkId]
		member := members[message.SenderID]
		user := state.State.Users[message.SenderID]
		trustedPublicKey, isTrusted := state.State.TrustedUsers[user.ID]
		keysMatch := bytes.Equal(trustedPublicKey, user.PublicKey)

		if isTrusted && !keysMatch {
			buf = append(buf, ui.UntrustedSymbol()...)
		}

		var senderStyle lipgloss.Style
		if isTrusted && keysMatch {
			if ownerId == member.UserID {
				senderStyle = ui.TrustedOwnerStyle()
			} else if member.IsAdmin {
				senderStyle = ui.TrustedAdminStyle()
			} else if member.IsMember {
				senderStyle = ui.TrustedMemberStyle()
			} else {
				senderStyle = ui.TrustedMemberStyle().Foreground(colors.White)
			}
		} else {
			if ownerId == member.UserID {
				senderStyle = ui.OwnerStyle()
			} else if member.IsAdmin {
				senderStyle = ui.AdminStyle()
			} else if member.IsMember {
				senderStyle = ui.UserStyle()
			} else {
				senderStyle = ui.UserStyle().Foreground(colors.White)
			}
		}

		sender := senderStyle.Render(user.Name)
		buf = append(buf, sender...)

		if member.IsMuted {
			buf = append(buf, []byte(MutedSymbol())...)
		}

		if _, ok := state.State.BlockedUsers[user.ID]; ok {
			buf = append(buf, blockStyle.Render(ui.BlockedSymbol())...)
		}

	} else if m.receiverIndex != -1 {
		user := state.State.Users[message.SenderID]
		trustedPublicKey, isTrusted := state.State.TrustedUsers[user.ID]
		keysMatch := bytes.Equal(trustedPublicKey, user.PublicKey)

		if isTrusted && !keysMatch {
			buf = append(buf, ui.UntrustedSymbol()...)
		}

		var senderStyle lipgloss.Style
		if isTrusted && keysMatch {
			senderStyle = ui.TrustedUserStyle()
		} else {
			senderStyle = ui.UserStyle()
		}

		sender := senderStyle.Render(user.Name)
		buf = append(buf, sender...)

		if _, ok := state.State.BlockedUsers[user.ID]; ok {
			buf = append(buf, blockStyle.Render(ui.BlockedSymbol())...)
		}
	}

	// Render header time format
	now := time.Now()
	unixTime := time.UnixMilli(message.ID.Time()).Local()
	var datetime string
	if unixTime.Year() == now.Year() && unixTime.YearDay() == now.YearDay() {
		datetime = " Today at " + unixTime.Format("3:04 PM")
	} else if unixTime.Year() == now.Year() && unixTime.YearDay() == now.YearDay()-1 {
		datetime = " Yesterday at " + unixTime.Format("3:04 PM")
	} else {
		datetime = unixTime.Format(" 02/01/2006 3:04 PM")
	}

	dateTimeStyle := lipgloss.NewStyle().
		Background(colors.Background).Foreground(colors.LightGray)
	if selected {
		dateTimeStyle = dateTimeStyle.Background(colors.BackgroundDim)
	}

	buf = append(buf, dateTimeStyle.Render(datetime)...)

	if selected {
		style := lipgloss.NewStyle().Background(colors.BackgroundDim).Width(m.width).Inline(true)
		buf = []byte(style.Render(string(buf)))
	} else {
		style := lipgloss.NewStyle().Background(colors.Background).Width(m.width).Inline(true)
		buf = []byte(style.Render(string(buf)))
	}

	buf = append(buf, '\n')
	return buf
}

func (m *Model) Scroll(amount int) {
	index := m.index
	if index == Unselected {
		log.Println("unselected, base:", m.base, "height", m.messagesHeight)
		if m.base != SnapToBottom {
			index = m.base - m.messagesHeight
		} else if amount > 0 {
			amount++ // Skip the blank line at the bottom
		}
	}
	m.SetIndex(index + amount)
	if m.index == Unselected {
		m.base = SnapToBottom
	}
}

func (m *Model) SetIndex(index int) {
	maxHeight := index
	if m.maxMessagesHeight != -1 {
		maxHeight = m.maxMessagesHeight - 1
	}
	// Order is significant, max(unselected) must be the last operation
	m.index = max(min(index, maxHeight), Unselected)

	if m.index == Unselected {
		return
	}

	upperBound := m.base
	if m.base == SnapToBottom {
		upperBound = m.messagesHeight
	}
	if m.index >= upperBound {
		m.base = m.index
	}

	if m.index <= upperBound-m.messagesHeight {
		m.base = m.index + m.messagesHeight - 1
	}

	// If at bottom snap to it
	if m.index <= 1 {
		m.base = SnapToBottom
	}

	if m.index == 0 {
		m.index = Unselected
	}
}

func (m *Model) editMessage() tea.Cmd {
	message := m.vi.String()
	if len(message) > MaxCharCount {
		return nil
	}
	if len(strings.TrimSpace(message)) == 0 {
		return nil
	}

	return gateway.Send(&packet.EditMessage{
		Message: m.editingMessage.ID,
		Content: message,
	})
}

func (m *Model) SetWidth(width int) {
	m.width = width
}

func (m *Model) renderFrequencyName() string {
	name := ""
	color := colors.White

	networkId := state.NetworkId(m.networkIndex)
	if m.frequencyIndex != -1 && networkId != nil {
		frequency := state.State.Frequencies[*networkId][m.frequencyIndex]
		color = lipgloss.Color(frequency.HexColor)
		name = frequency.Name
	} else if m.receiverIndex != -1 {
		signal := state.Data.Signals[m.receiverIndex]
		user := state.State.Users[signal]
		trustedPublicKey, isTrusted := state.State.TrustedUsers[user.ID]
		keysMatch := bytes.Equal(trustedPublicKey, user.PublicKey)

		color = colors.Purple
		if isTrusted && keysMatch {
			color = colors.Turquoise
		} else if isTrusted {
			color = colors.Red
		}

		name = user.Name
	} else {
		return ""
	}

	nameStyle := lipgloss.NewStyle().Width(m.width).
		Background(colors.Background).Foreground(color).
		AlignHorizontal(lipgloss.Center).
		Border(lipgloss.ThickBorder(), false, false, true).
		BorderForeground(colors.White)
	if m.focus {
		nameStyle = nameStyle.BorderForeground(colors.Focus)
	}

	return nameStyle.Render(name) + "\n"
}

func (m *Model) Mode() int {
	return m.vi.Mode()
}

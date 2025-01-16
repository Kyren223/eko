package chat

import (
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
	blurStyle     = lipgloss.NewStyle().Foreground(colors.White)
	focusStyle    = lipgloss.NewStyle().Foreground(colors.Focus)
	readOnlyStyle = lipgloss.NewStyle().Foreground(colors.Gray)
	mutedStyle    = lipgloss.NewStyle().Foreground(colors.Red)

	ViBlurredBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), true, true, false).
			Padding(0, 1)
	ViFocusedBorder  = ViBlurredBorder.BorderForeground(colors.Focus)
	ViReadOnlyBorder = ViBlurredBorder.BorderForeground(colors.Gray)
	ViMutedBorder    = ViBlurredBorder.BorderForeground(colors.Red)

	VimModeStyle = lipgloss.NewStyle().Bold(true)

	NormalMemberStyle = lipgloss.NewStyle().Foreground(colors.Purple).SetString("󰀉")
	AdminMemberStyle  = lipgloss.NewStyle().Foreground(colors.Red).Bold(true).SetString("󰓏")
	OwnerMemberStyle  = AdminMemberStyle.Foreground(colors.Gold).SetString("󱟜")

	DateTimeStyle = lipgloss.NewStyle().Foreground(colors.LightGray).SetString("")

	PaddingCount = 1
	Padding      = strings.Repeat(" ", PaddingCount)
	Border       = lipgloss.RoundedBorder()
	LeftCorner   = Border.BottomLeft + Border.Bottom
	RightCorner  = Border.Bottom + Border.BottomRight

	NoMessages = lipgloss.NewStyle().
			Foreground(colors.LightGray).Padding(0, PaddingCount, 1).
			AlignHorizontal(lipgloss.Center).AlignVertical(lipgloss.Bottom).
			SetString("This frequency has no messages, start transmiting!")
	NoAccess = NoMessages.SetString("You do not have permissions to see messages in this frequency")

	SelectedGap = lipgloss.NewStyle().Background(colors.BackgroundDim)

	SendMessagePlaceholder = "Send a message..."
	ReadOnlyPlaceholder    = "You do not have permissions to send messages in this frequency"
	MutedPlaceholder       = "You have been muted by a server adminstrator"
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
	selectedMessage *snowflake.ID

	messagesHeight    int
	maxMessagesHeight int
	messagesCache     *string
	prerender         string

	width int

	style       lipgloss.Style
	borderStyle lipgloss.Style
}

func New(width int) Model {
	viWidth := width - PaddingCount*2 - lipgloss.Width(LeftCorner) - lipgloss.Width(RightCorner)
	vi := viminput.New(viWidth, ui.Height/2)
	vi.Placeholder = SendMessagePlaceholder
	vi.PlaceholderStyle = lipgloss.NewStyle().Foreground(colors.Gray)

	return Model{
		vi:                vi,
		focus:             false,
		locked:            false,
		networkIndex:      -1,
		receiverIndex:     -1,
		frequencyIndex:    -1,
		base:              SnapToBottom,
		index:             Unselected,
		messagesHeight:    0,
		maxMessagesHeight: -1,
		messagesCache:     nil,
		prerender:         "",
		width:             width,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Prerender() {
	messagebox := m.renderMessageBox()
	messagesHeight := ui.Height - lipgloss.Height(messagebox)
	messagesHeight -= 1 // For the extra \n at the end

	// if m.messagesCache == nil || messagesHeight != m.messagesHeight {
	// 	// Re-render
	// }
	m.selectedMessage = nil
	messages := m.renderMessages(messagesHeight)
	m.messagesCache = &messages
	m.messagesHeight = messagesHeight

	m.prerender = *m.messagesCache + messagebox
}

func (m Model) View() string {
	return m.prerender
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	// TODO: properly invalidate cache
	m.messagesCache = nil

	networkId := state.NetworkId(m.networkIndex)
	if m.frequencyIndex != -1 && networkId != nil {
		frequencies := state.State.Frequencies[*networkId]
		frequency := frequencies[m.frequencyIndex]
		member := state.State.Members[*networkId][*state.UserID]

		m.hasReadAccess = frequency.Perms != packet.PermNoAccess || member.IsAdmin
		m.hasWriteAccess = !member.IsMuted && (frequency.Perms == packet.PermReadWrite || member.IsAdmin)

		if member.IsMuted {
			m.vi.Placeholder = MutedPlaceholder
			m.borderStyle = ViMutedBorder
			m.style = mutedStyle
			m.vi.SetInactive(true)
			m.locked = false
		} else if frequency.Perms != packet.PermReadWrite && !member.IsAdmin {
			m.vi.Placeholder = ReadOnlyPlaceholder
			m.borderStyle = ViReadOnlyBorder
			m.style = readOnlyStyle
			m.vi.SetInactive(true)
			m.locked = false
		} else if m.locked {
			m.vi.Placeholder = SendMessagePlaceholder
			m.borderStyle = ViFocusedBorder
			m.style = focusStyle
			m.vi.SetInactive(false)
		} else {
			m.vi.Placeholder = SendMessagePlaceholder
			m.borderStyle = ViBlurredBorder
			m.style = blurStyle
			m.vi.SetInactive(false)
		}
	} else if m.receiverIndex != -1 {
		// TODO: receiver
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
				m.Prerender()
				return m, nil
			}

			if key.Type == tea.KeyEnter {
				cmd := m.sendMessage()
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
				m.locked = true
				m.vi.SetMode(viminput.InsertMode)
				m.base = SnapToBottom
				m.SetIndex(Unselected)
			}

		case "x", "X", "d", "D":
			if m.selectedMessage != nil {
				log.Println("deleting message:", m.selectedMessage)
				cmd = gateway.Send(&packet.DeleteMessage{
					Message: *m.selectedMessage,
				})
			}
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
	if len(message) > MaxCharCount {
		return nil
	}
	if len(strings.TrimSpace(message)) == 0 {
		return nil
	}

	m.vi.Reset()
	m.base = SnapToBottom

	var receiverId *snowflake.ID = nil
	if m.receiverIndex != -1 {
		// TODO: do nothing for now, until trusted friends are implemented
	}

	var frequencyId *snowflake.ID = nil
	networkId := state.NetworkId(m.networkIndex)
	if m.frequencyIndex != -1 && networkId != nil {
		frequencies := state.State.Frequencies[*networkId]
		frequencyId = &frequencies[m.frequencyIndex].ID
	}

	return gateway.Send(&packet.SendMessage{
		ReceiverID:  receiverId,
		FrequencyID: frequencyId,
		Content:     message,
	})
}

func (m *Model) SetReceiver(receiverIndex int) {
	if m.receiverIndex == receiverIndex {
		return
	}
	m.ResetBeforeSwitch()
	m.receiverIndex = receiverIndex
	m.frequencyIndex = -1
	m.networkIndex = -1
	m.RestoreAfterSwitch()
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
	m.vi.Reset()
	m.base = SnapToBottom
	m.SetIndex(Unselected)
	m.maxMessagesHeight = -1

	networkId := state.NetworkId(m.networkIndex)
	if m.frequencyIndex != -1 && networkId != nil {
		frequencies := state.State.Frequencies[*networkId]
		if len(frequencies) >= m.frequencyIndex {
			return
		}
		frequencyId := frequencies[m.frequencyIndex].ID
		log.Println("Saving", frequencyId)
		state.State.FrequencyState[frequencyId] = state.FrequencyState{
			IncompleteMessage: m.vi.String(),
			Base:              m.base,
			MaxHeight:         m.maxMessagesHeight,
		}
	} else if m.receiverIndex != -1 {
		// TODO: receiver
	}
}

func (m *Model) RestoreAfterSwitch() tea.Cmd {
	msgs := state.State.FrequencyState
	networkId := state.NetworkId(m.networkIndex)
	if m.frequencyIndex != -1 && networkId != nil {
		frequencies := state.State.Frequencies[*networkId]
		frequency := frequencies[m.frequencyIndex]
		log.Println("Restoring", frequency.ID)

		if val, ok := msgs[frequency.ID]; ok {
			m.vi.SetString(val.IncompleteMessage)
			m.base = val.Base
			m.SetIndex(Unselected)
			m.maxMessagesHeight = val.MaxHeight

			// Don't ask for messages if you already visited this frequency
			return nil
		}
		return gateway.Send(&packet.RequestMessages{
			ReceiverID:  nil,
			FrequencyID: &frequency.ID,
		})
	} else if m.receiverIndex != -1 {
		// TODO: receiver
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
	mode := VimModeStyle.Foreground(colors.Gray).Render("  NONE ")
	if m.locked {
		switch m.vi.Mode() {
		case viminput.InsertMode:
			mode = VimModeStyle.Foreground(colors.Green).Render("  INSERT ")
		case viminput.NormalMode:
			mode = VimModeStyle.Foreground(colors.Orange).Render("  NORMAL ")
		case viminput.OpendingMode:
			mode = VimModeStyle.Foreground(colors.Red).Render("  O-PENDING ")
		case viminput.VisualMode:
			mode = VimModeStyle.Foreground(colors.Turquoise).Render("  VISUAL ")
		case viminput.VisualLineMode:
			mode = VimModeStyle.Foreground(colors.Turquoise).Render("  V-LINE ")
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
	}
	countStr += " / " + strconv.Itoa(MaxCharCount) + " "
	width -= lipgloss.Width(countStr)
	width -= lipgloss.Width(rightAngle)

	width -= lipgloss.Width(RightCorner)

	bottomCount := width / lipgloss.Width(Border.Bottom)
	bottom := strings.Repeat(Border.Bottom, bottomCount)
	builder.WriteString(m.style.Render(bottom))

	builder.WriteString(leftAngle)
	builder.WriteString(countStr)
	builder.WriteString(rightAngle)
	builder.WriteString(m.style.Render(RightCorner))

	result := builder.String()

	return lipgloss.NewStyle().Padding(0, PaddingCount).Render(result)
}

func (m *Model) renderMessages(screenHeight int) string {
	var btree *btree.BTreeG[data.Message]
	networkId := state.NetworkId(m.networkIndex)
	if m.frequencyIndex != -1 && networkId != nil {
		frequencies := state.State.Frequencies[*networkId]
		frequencyId := frequencies[m.frequencyIndex].ID
		btree = state.State.Messages[frequencyId]
	} else {
		// TODO: implement support for receiver id
	}

	if !m.hasReadAccess {
		return NoAccess.Width(m.width).Height(screenHeight).String() + "\n"
	}

	if btree == nil {
		return NoMessages.Width(m.width).Height(screenHeight).String() + "\n"
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

	last := snowflake.ID(0)
	btree.Descend(func(message data.Message) bool {
		last = message.ID

		if len(group) == 0 {
			group = append(group, message)
			return true
		}

		lastMsg := group[0]
		sameSender := lastMsg.SenderID == message.SenderID
		withinTime := lastMsg.ID.Time()-message.ID.Time() <= TimeGap
		if sameSender && withinTime && len(group) < MaxViewableMessages {
			group = append(group, message)
			return true
		}

		renderedGroup := m.renderMessageGroup(group, &remainingHeight, height)
		renderedGroups = append(renderedGroups, renderedGroup)
		group = []data.Message{message}

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
		PaddingLeft(PaddingCount + 2).PaddingRight(PaddingCount)
	for i := len(group) - 1; i >= 0; i-- {
		content := messageStyle.Render(group[i].Content)
		heights[i] = lipgloss.Height(content)
		checkpoints[i] = len(buf)

		buf = append(buf, content...)
		buf = append(buf, '\n')
	}

	// Gap between each message group
	if m.index == height-*remaining {
		gap := SelectedGap.Width(m.width).String()
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
		m.selectedMessage = &group[selectedIndex].ID

		if selectedIndex == len(group)-1 {
			buf = m.renderHeader(group[selectedIndex], true)
		} else {
			buf = buf[:checkpoints[selectedIndex]] // Revert
		}

		content := group[selectedIndex].Content
		content = messageStyle.Background(colors.BackgroundDim).Render(content)
		buf = append(buf, content...)
		buf = append(buf, '\n')

		// Redraw rest
		for i := selectedIndex - 1; i >= 0; i-- {
			content := messageStyle.Render(group[i].Content)
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

	networkId := state.NetworkId(m.networkIndex)
	if networkId != nil {
		ownerId := state.State.Networks[*networkId].OwnerID
		members := state.State.Members[*networkId]
		member := members[message.SenderID]
		user := state.State.Users[message.SenderID]

		senderStyle := NormalMemberStyle
		if ownerId == member.UserID {
			senderStyle = OwnerMemberStyle
		} else if member.IsAdmin {
			senderStyle = AdminMemberStyle
		}

		sender := senderStyle.Render(user.Name)
		buf = append(buf, sender...)
	} else if m.receiverIndex != -1 {
		// TODO: receiver
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

	dateTimeStyle := DateTimeStyle
	if selected {
		dateTimeStyle = dateTimeStyle.Background(colors.BackgroundDim)
	}

	buf = append(buf, dateTimeStyle.Render(datetime)...)

	if selected {
		style := lipgloss.NewStyle().Background(colors.BackgroundDim).Width(m.width).Inline(true)
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
	m.index = min(max(index, Unselected), maxHeight)

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

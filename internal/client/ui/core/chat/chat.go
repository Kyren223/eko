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
	focusStyle = lipgloss.NewStyle().Foreground(colors.Focus)

	ViBlurredBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), true, true, false).
			Padding(0, 1)
	ViFocusedBorder = ViBlurredBorder.BorderForeground(colors.Focus)

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

	NilBtreeError = lipgloss.NewStyle().
			Foreground(colors.Red).Padding(0, PaddingCount).
			SetString("Error loading messages!")
)

const (
	MaxCharCount        = 2000
	MaxViewableMessages = 200

	TimeGap = 7 * 60 * 1000 // 7 minutes in millis
)

type Model struct {
	vi     viminput.Model
	focus  bool
	locked bool

	networkIndex   int
	receiverIndex  int
	frequencyIndex int

	snapToBottom bool
	offset       int
	index        int

	width int
}

func New(width int) Model {
	viWidth := width - PaddingCount*2 - lipgloss.Width(LeftCorner) - lipgloss.Width(RightCorner)
	vi := viminput.New(viWidth, ui.Height/2)
	vi.Placeholder = "Send a message..."
	vi.PlaceholderStyle = lipgloss.NewStyle().Foreground(colors.Gray)

	return Model{
		vi:             vi,
		focus:          false,
		locked:         false,
		networkIndex:   -1,
		receiverIndex:  -1,
		frequencyIndex: -1,
		snapToBottom:   true,
		offset:         -1,
		index:          -1,
		width:          width,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	messagebox := m.renderMessageBox()
	messagesHeight := ui.Height - lipgloss.Height(messagebox)

	messages := m.renderMessages(messagesHeight)

	result := messages + "\n" + messagebox
	return result
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focus {
		return m, nil
	}

	if m.locked {
		if key, ok := msg.(tea.KeyMsg); ok {
			InNormal := m.vi.Mode() == viminput.NormalMode
			if key.String() == "q" && InNormal {
				m.locked = false
				return m, nil
			}

			if key.Type == tea.KeyEnter {
				cmd := m.sendMessage()
				return m, cmd
			}
		}

		var cmd tea.Cmd
		m.vi, cmd = m.vi.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "i":
			m.locked = true
			m.vi.SetMode(viminput.InsertMode)
			m.index = -1
		case "k":
			m.index++
		case "j":
			m.index = max(-1, m.index-1)
		}
	}

	return m, nil
}

func (m *Model) Focus() {
	m.focus = true
	m.vi.Focus()
}

func (m *Model) Blur() {
	m.focus = false
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

	var receiverId *snowflake.ID = nil
	if m.receiverIndex != -1 {
		// TODO: do nothing for now, until trusted friends are implemented
	}

	var frequencyId *snowflake.ID = nil
	if m.frequencyIndex != -1 && m.networkIndex != -1 {
		network := state.State.Networks[m.networkIndex]
		frequencyId = &network.Frequencies[m.frequencyIndex].ID
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
	m.index = -1
	if m.frequencyIndex != -1 && m.networkIndex != -1 {
		network := state.State.Networks[m.networkIndex]
		frequencyId := network.Frequencies[m.frequencyIndex].ID
		log.Println("Saving", frequencyId)
		state.State.IncompleteMessages[frequencyId] = m.vi.String()
		m.vi.Reset()
	} else if m.receiverIndex != -1 {
		// TODO: receiver
	}
}

func (m *Model) RestoreAfterSwitch() tea.Cmd {
	msgs := state.State.IncompleteMessages
	if m.frequencyIndex != -1 && m.networkIndex != -1 {
		network := state.State.Networks[m.networkIndex]
		frequencyId := network.Frequencies[m.frequencyIndex].ID
		log.Println("Restoring", frequencyId)
		if val, ok := msgs[frequencyId]; ok {
			m.vi.SetString(val)
		}
		return gateway.Send(&packet.RequestMessages{
			ReceiverID:  nil,
			FrequencyID: &frequencyId,
		})
	} else if m.receiverIndex != -1 {
		// TODO: receiver
	}

	return nil
}

func (m *Model) renderMessage(message data.Message, width int, header bool, selected bool) string {
	metadata := ""
	if header && m.networkIndex != -1 {
		var member *data.GetNetworkMembersRow = nil
		network := state.State.Networks[m.networkIndex]
		for _, networkMember := range network.Members {
			if networkMember.User.ID == message.SenderID {
				member = &networkMember
			}
		}
		assert.NotNil(member, "user should always exist")

		senderStyle := NormalMemberStyle
		if network.OwnerID == member.User.ID {
			senderStyle = OwnerMemberStyle
		} else if member.IsAdmin {
			senderStyle = AdminMemberStyle
		}

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
			senderStyle = senderStyle.Background(colors.BackgroundDim)
		}

		sender := senderStyle.Render(member.User.Name)
		sentDatetime := dateTimeStyle.Render(datetime)
		metadata = Padding + sender + " " + sentDatetime + Padding + "\n"
	}

	messageStyle := lipgloss.NewStyle().Width(width).
		PaddingLeft(PaddingCount + 2).PaddingRight(PaddingCount)
	if selected {
		messageStyle = messageStyle.Background(colors.BackgroundDim)
	}
	content := messageStyle.Render(message.Content)

	return metadata + content
}

func (m *Model) renderMessageBox() string {
	var builder strings.Builder

	input := m.vi.View()
	if m.locked {
		input = ViFocusedBorder.Render(input)
	} else {
		input = ViBlurredBorder.Render(input)
	}
	builder.WriteString(input)
	builder.WriteByte('\n')

	style := lipgloss.NewStyle()
	if m.locked {
		style = focusStyle
	}
	width := lipgloss.Width(input)

	leftAngle := style.Render("")
	rightAngle := style.Render("")

	builder.WriteString(style.Render(LeftCorner))
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
	builder.WriteString(style.Render(bottom))

	builder.WriteString(leftAngle)
	builder.WriteString(countStr)
	builder.WriteString(rightAngle)
	builder.WriteString(style.Render(RightCorner))

	result := builder.String()

	return lipgloss.NewStyle().Padding(0, PaddingCount).Render(result)
}

func (m *Model) renderMessages(height int) string {
	var btree *btree.BTreeG[data.Message]
	if m.frequencyIndex != -1 && m.networkIndex != -1 {
		network := state.State.Networks[m.networkIndex]
		frequencyId := network.Frequencies[m.frequencyIndex].ID
		btree = state.State.Messages[frequencyId]
	} else {
		// TODO: implement support for receiver id
	}

	if btree == nil {
		return NilBtreeError.Height(height).String()
	}

	remainingHeight := height
	renderedGroups := []string{}
	group := []data.Message{}

	if m.snapToBottom {
		btree.Descend(func(message data.Message) bool {
			if len(group) == 0 {
				group = append(group, message)
				return true
			}

			lastMsg := group[0]
			sameSender := lastMsg.SenderID == message.SenderID
			withinTime := lastMsg.ID.Time()-message.ID.Time() <= TimeGap
			log.Println(sameSender, withinTime)
			if sameSender && withinTime && len(group) < MaxViewableMessages {
				group = append(group, message)
				return true
			}

			renderedGroup := m.renderMessageGroup(group)
			renderedGroups = append(renderedGroups, renderedGroup)
			remainingHeight -= lipgloss.Height(renderedGroup)

			return remainingHeight > 0
		})

		// We ran out of messages, so let's render the last group
		if remainingHeight > 0 {
			renderedGroup := m.renderMessageGroup(group)
			renderedGroups = append(renderedGroups, renderedGroup)
		}
	}

	// selectedStart := -1
	// selectedMessage := ""
	// _ = selectedMessage

	// const timeGap = 7 * 60 * 1000 // 7 minutes in millis
	// initialTime := int64(0)
	// previousSender := snowflake.ID(0)
	//
	// count := btree.Len()

	// btree.Ascend(func(message data.Message) bool {
	// 	count--
	//
	// 	outsideTimeRange := message.ID.Time()-timeGap > initialTime
	// 	header := outsideTimeRange || previousSender != message.SenderID
	//
	// 	if header {
	// 		initialTime = message.ID.Time()
	// 		previousSender = message.SenderID
	// 		builder.WriteByte('\n')
	// 		remains -= 1
	//
	// 	}
	//
	// 	if m.index == count {
	// 		selectedStart = height - remains
	// 		var b strings.Builder
	//
	// 		DateTimeStyle = DateTimeStyle.Background(SelectedBackground.GetBackground())
	// 		remains -= m.renderMessage(message, &b, header)
	// 		DateTimeStyle = DateTimeStyle.UnsetBackground()
	//
	// 		selectedMessage = b.String()
	// 		builder.WriteString(b.String())
	// 	} else {
	// 		remains -= m.renderMessage(message, &builder, header)
	// 	}
	//
	// 	return true
	// })
	//
	// messages := builder.String()
	// actualMessagesHeight := lipgloss.Height(messages)
	// if actualMessagesHeight < height {
	// 	diff := height - actualMessagesHeight
	// 	messages = strings.Repeat("\n", diff) + messages
	// 	if selectedStart != -1 {
	// 		selectedStart += diff
	// 	}
	// } else if actualMessagesHeight > height {
	// 	diff := actualMessagesHeight - height
	// 	newlines := 0
	// 	index := -1
	// 	for i, c := range messages {
	// 		if newlines == diff {
	// 			index = i
	// 			break
	// 		}
	// 		if c == '\n' {
	// 			newlines++
	// 		}
	// 	}
	// 	assert.Assert(index != -1, "must always have enough newlines if it's greater")
	// 	messages = messages[index:]
	// }

	var builder strings.Builder
	for i := len(renderedGroups) - 1; i >= 0; i-- {
		builder.WriteString(renderedGroups[i])
		builder.WriteByte('\n') // Gap between each group
	}

	return builder.String()
}

func (m *Model) renderMessageGroup(group []data.Message) string {
	assert.Assert(len(group) != 0, "cannot render a group with length 0")

	var builder strings.Builder
	firstMsg := group[len(group)-1]

	// Render the header of the first group
	builder.WriteString(Padding)
	if m.networkIndex != -1 {
		var member *data.GetNetworkMembersRow = nil
		network := state.State.Networks[m.networkIndex]
		for _, networkMember := range network.Members {
			if networkMember.User.ID == firstMsg.SenderID {
				member = &networkMember
			}
		}
		assert.NotNil(member, "sender should always exist")

		senderStyle := NormalMemberStyle
		if network.OwnerID == member.User.ID {
			senderStyle = OwnerMemberStyle
		} else if member.IsAdmin {
			senderStyle = AdminMemberStyle
		}
		sender := senderStyle.Render(member.User.Name)
		builder.WriteString(sender)
	} else if m.receiverIndex != -1 {
		// TODO: receiver
	}

	// Render header time format
	now := time.Now()
	unixTime := time.UnixMilli(firstMsg.ID.Time()).Local()
	var datetime string
	if unixTime.Year() == now.Year() && unixTime.YearDay() == now.YearDay() {
		datetime = " Today at " + unixTime.Format("3:04 PM")
	} else if unixTime.Year() == now.Year() && unixTime.YearDay() == now.YearDay()-1 {
		datetime = " Yesterday at " + unixTime.Format("3:04 PM")
	} else {
		datetime = unixTime.Format(" 02/01/2006 3:04 PM")
	}
	dateTimeStyle := DateTimeStyle
	// if selected {
	// 	dateTimeStyle = dateTimeStyle.Background(colors.BackgroundDim)
	// 	senderStyle = senderStyle.Background(colors.BackgroundDim)
	// }
	builder.WriteByte(' ') // Gap between sender and time format
	builder.WriteString(dateTimeStyle.Render(datetime))
	builder.WriteByte('\n')

	// Render all messages content
	messageStyle := lipgloss.NewStyle().Width(m.width).
		PaddingLeft(PaddingCount + 2).PaddingRight(PaddingCount)
	for i := len(group) - 1; i >= 0; i-- {
		// if selected {
		// 	messageStyle = messageStyle.Background(colors.BackgroundDim)
		// }
		content := messageStyle.Render(group[i].Content)
		builder.WriteString(content)
		builder.WriteByte('\n')
	}

	return builder.String()
}

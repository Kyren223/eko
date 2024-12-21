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

	Padding     = 1
	Border      = lipgloss.RoundedBorder()
	LeftCorner  = Border.BottomLeft + Border.Bottom
	RightCorner = Border.Bottom + Border.BottomRight

	SelectedBackground = lipgloss.NewStyle().Padding(0, Padding).
				Background(colors.BackgroundDim)
)

const MaxCharCount = 2000

type Model struct {
	vi     viminput.Model
	focus  bool
	locked bool

	networkIndex   int // Note this might be invalid, rely on frequencyIndex
	receiverIndex  *int
	frequencyIndex *int

	index int

	width int
}

func New(width int) Model {
	viWidth := width - Padding*2 - lipgloss.Width(LeftCorner) - lipgloss.Width(RightCorner)
	vi := viminput.New(viWidth, ui.Height/2)
	vi.Placeholder = "Send a message..."
	vi.PlaceholderStyle = lipgloss.NewStyle().Foreground(colors.Gray)

	return Model{
		vi:     vi,
		focus:  false,
		locked: false,
		width:  width,
		index:  -1,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
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
	//  NORMAL   master  󰀦 1   LSP                                                                utf-8     go  51%   67:21

	messagebox := builder.String()
	messagesHeight := ui.Height - lipgloss.Height(messagebox) + 1
	remains := messagesHeight

	builder.Reset()
	var btree *btree.BTreeG[data.Message]
	if m.frequencyIndex != nil {
		network := state.State.Networks[m.networkIndex]
		frequencyId := network.Frequencies[*m.frequencyIndex].ID
		btree = state.State.Messages[frequencyId]
	} else {
		// TODO: implement support for receiver id
	}

	selectedStart := -1
	selectedMessage := ""

	if btree != nil {
		const timeGap = 7 * 60 * 1000 // 7 minutes in millis
		initialTime := int64(0)
		previousSender := snowflake.ID(0)

		count := btree.Len()

		btree.Ascend(func(message data.Message) bool {
			count--

			outsideTimeRange := message.ID.Time()-timeGap > initialTime
			header := outsideTimeRange || previousSender != message.SenderID

			if header {
				initialTime = message.ID.Time()
				previousSender = message.SenderID
				builder.WriteByte('\n')
				remains -= 1

			}

			if m.index == count {
				selectedStart = messagesHeight - remains
				var b strings.Builder

				DateTimeStyle = DateTimeStyle.Background(SelectedBackground.GetBackground())
				remains -= m.renderMessage(message, &b, header)
				DateTimeStyle = DateTimeStyle.UnsetBackground()

				selectedMessage = b.String()
				builder.WriteString(b.String())
			} else {
				remains -= m.renderMessage(message, &builder, header)
			}

			return true
		})
	}

	messages := builder.String()
	actualMessagesHeight := lipgloss.Height(messages)
	if actualMessagesHeight < messagesHeight {
		diff := messagesHeight - actualMessagesHeight
		messages = strings.Repeat("\n", diff) + messages
		if selectedStart != -1 {
			selectedStart += diff
		}
	} else if actualMessagesHeight > messagesHeight {
		diff := actualMessagesHeight - messagesHeight
		newlines := 0
		index := -1
		for i, c := range messages {
			if newlines == diff {
				index = i
				break
			}
			if c == '\n' {
				newlines++
			}
		}
		assert.Assert(index != -1, "must always have enough newlines if it's greater")
		messages = messages[index:]
	}

	result := messages + messagebox
	result = lipgloss.NewStyle().Padding(0, Padding).
		MaxWidth(m.width).MaxHeight(ui.Height).
		Render(result)

	if selectedStart != -1 {
		selectedMessage = selectedMessage[:len(selectedMessage)-1]
		selectedMessage = SelectedBackground.Width(m.width).
			Render(selectedMessage)
		result = ui.PlaceOverlay(0, selectedStart, selectedMessage, result)
	}

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
	if m.receiverIndex != nil {
		// TODO: do nothing for now, until trusted friends are implemented
	}

	var frequencyId *snowflake.ID = nil
	if m.frequencyIndex != nil {
		network := state.State.Networks[m.networkIndex]
		frequencyId = &network.Frequencies[*m.frequencyIndex].ID
	}

	return gateway.Send(&packet.SendMessage{
		ReceiverID:  receiverId,
		FrequencyID: frequencyId,
		Content:     message,
	})
}

// func (m *Model) SetNetworkIndex(networkIndex int) {
// 	if m.networkIndex != networkIndex {
// 		m.index = -1
// 	}
// 	m.networkIndex = networkIndex
// }

func (m *Model) SetReceiver(receiverIndex int) {
	if m.receiverIndex != nil && *m.receiverIndex == receiverIndex {
		return
	}
	m.ResetBeforeSwitch()
	m.receiverIndex = &receiverIndex
	m.frequencyIndex = nil
	m.RestoreAfterSwitch()
}

func (m *Model) SetFrequency(networkIndex, frequencyIndex int) {
	if m.frequencyIndex != nil && *m.frequencyIndex == frequencyIndex {
		return
	}
	m.ResetBeforeSwitch()
	m.receiverIndex = nil
	m.frequencyIndex = &frequencyIndex
	m.networkIndex = networkIndex
	m.RestoreAfterSwitch()
}

func (m *Model) ResetBeforeSwitch() {
	source := m.frequencyIndex
	if source == nil {
		source = m.receiverIndex
	}
	if source == nil {
		return
	}

	log.Println("Resetting", source)
	state.State.IncompleteMessages[snowflake.ID(*source)] = m.vi.String()
	m.vi.Reset()
	m.index = -1
}

func (m *Model) RestoreAfterSwitch() {
	source := m.frequencyIndex
	if source == nil {
		source = m.receiverIndex
	}
	if source == nil {
		return
	}

	log.Println("Restoring", source)
	msgs := state.State.IncompleteMessages
	if val, ok := msgs[snowflake.ID(*source)]; ok {
		m.vi.SetString(val)
	}
}

func (m *Model) renderMessage(message data.Message, builder *strings.Builder, header bool) int {
	lines := 0

	if header {
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
		builder.WriteString(senderStyle.Render(member.User.Name))

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

		builder.WriteString(DateTimeStyle.Render(datetime))

		builder.WriteByte('\n')
		lines += 1
	}

	for _, line := range strings.Split(message.Content, "\n") {
		builder.WriteString("  ")
		builder.WriteString(line)
		builder.WriteByte('\n')
		lines += 1
	}

	return lines
}

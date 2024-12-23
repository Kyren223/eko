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

	SnapToBottom = -1
)

type Model struct {
	vi     viminput.Model
	focus  bool
	locked bool

	networkIndex   int
	receiverIndex  int
	frequencyIndex int

	offset int
	index  int

	messagesHeight int
	messagesCache  *string
	prerender      string

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
		offset:         SnapToBottom,
		index:          -1,
		messagesHeight: 0,
		messagesCache:  nil,
		prerender:      "",
		width:          width,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Prerender() {
	messagebox := m.renderMessageBox()
	messagesHeight := ui.Height - lipgloss.Height(messagebox)

	// if m.messagesCache == nil || messagesHeight != m.messagesHeight {
	// 	// Re-render
	// }
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

	if !m.focus {
		m.Prerender()
		return m, nil
	}

	if m.locked {
		if key, ok := msg.(tea.KeyMsg); ok {
			InNormal := m.vi.Mode() == viminput.NormalMode
			if key.String() == "q" && InNormal {
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

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "i":
			m.locked = true
			m.vi.SetMode(viminput.InsertMode)
			m.index = -1
		case "k":
			// TODO: if not snap to bottom and -1 then do offset - lastHeight
			m.index++
			maxHeight := m.offset
			if m.offset == SnapToBottom {
				maxHeight = m.messagesHeight
			}
			if m.index == maxHeight-2 {
				log.Println("Index:", m.index, "Height:", m.messagesHeight)
				m.offset = maxHeight + 1
				m.index--

				// Max height is here
				// Should be rendered up to here <-
				// After add
				// Here
			}
		case "j":
			m.index = max(-1, m.index-1)
			if m.offset != SnapToBottom {
				diff := m.offset - m.messagesHeight - m.index
				if diff > 0 {
					m.offset -= diff
				}
			}
			if m.offset == 0 {
				m.offset = SnapToBottom
			}
		}
	}

	m.Prerender()
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

	remainingHeight := m.offset
	if m.offset == SnapToBottom {
		remainingHeight = height
	}

	renderedGroups := []string{}
	group := []data.Message{}

	btree.Descend(func(message data.Message) bool {
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
		group = []data.Message{}
		renderedGroups = append(renderedGroups, renderedGroup)

		return remainingHeight > 0
	})

	// We ran out of messages, so let's render the last group
	if remainingHeight > 0 && len(group) != 0 {
		renderedGroup := m.renderMessageGroup(group, &remainingHeight, height)
		renderedGroups = append(renderedGroups, renderedGroup)
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

	if m.offset == SnapToBottom {
		// Truncate any excess and show only the bottom
		newlines := 0
		index := -1
		for i := len(result) - 1; i >= 0; i-- {
			if result[i] == '\n' {
				newlines++
			}
			if newlines == height {
				index = i
				break
			}
		}
		if index != -1 {
			return result[index+1:]
		}
	} else {
		// Show from the offset up to offset+height
		newlines := 0
		offsetIndex := -1
		upToIndex := -1
		for i := len(result) - 1; i >= 0; i-- {
			if result[i] == '\n' {
				newlines++
			}
			if newlines == m.offset && offsetIndex == -1 {
				offsetIndex = i
			}
			if newlines == m.offset-height && upToIndex == -1 {
				upToIndex = i
			}
			if offsetIndex != -1 && upToIndex != -1 {
				break
			}
		}
		if offsetIndex != -1 && upToIndex != -1 {
			return result[offsetIndex+1 : upToIndex+1]
		}
	}

	return result
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

	selectedIndex := -1
	for i, h := range heights {
		bottom := height - *remaining
		top := bottom + h
		*remaining -= h
		if bottom <= m.index && m.index <= top {
			selectedIndex = i
		}
	}
	buf = append(buf, '\n') // Gap between each message group
	*remaining--            // For gap
	*remaining--            // For the header

	log.Println(selectedIndex)

	if selectedIndex != -1 {
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

			heights[i] = lipgloss.Height(content)
			checkpoints[i] = len(buf)
		}
		buf = append(buf, '\n') // Gap between each message group
	}

	return string(buf)
}

func (m *Model) renderHeader(message data.Message, selected bool) []byte {
	var buf []byte
	buf = append(buf, Padding...)

	if m.networkIndex != -1 {
		var member *data.GetNetworkMembersRow = nil
		network := state.State.Networks[m.networkIndex]
		for _, networkMember := range network.Members {
			if networkMember.User.ID == message.SenderID {
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

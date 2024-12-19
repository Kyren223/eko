package chat

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/client/ui/viminput"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/snowflake"
)

var (
	focusStyle = lipgloss.NewStyle().Foreground(colors.Focus)

	ViBlurredBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), true, true, false).
			Padding(0, 1)
	ViFocusedBorder = ViBlurredBorder.BorderForeground(colors.Focus)

	VimModeStyle = lipgloss.NewStyle().Bold(true)
)

const MaxCharCount = 2000

type Model struct {
	vi     viminput.Model
	focus  bool
	locked bool

	networkIndex   int // Note this might be invalid, rely on frequencyIndex
	receiverIndex  *int
	frequencyIndex *int
}

func New() Model {
	vi := viminput.New(90, 20)
	vi.Placeholder = "Send a message..."
	vi.PlaceholderStyle = lipgloss.NewStyle().Foreground(colors.Gray)

	return Model{
		vi:     vi,
		focus:  false,
		locked: false,
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

	border := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle()
	if m.locked {
		style = focusStyle
	}
	width := lipgloss.Width(input)

	leftAngle := style.Render("")
	rightAngle := style.Render("")

	leftCorner := border.BottomLeft + border.Bottom
	builder.WriteString(style.Render(leftCorner))
	width -= lipgloss.Width(leftCorner)

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

	rightCorner := border.Bottom + border.BottomRight
	width -= lipgloss.Width(rightCorner)

	bottomCount := width / lipgloss.Width(border.Bottom)
	bottom := strings.Repeat(border.Bottom, bottomCount)
	builder.WriteString(style.Render(bottom))

	builder.WriteString(leftAngle)
	builder.WriteString(countStr)
	builder.WriteString(rightAngle)
	builder.WriteString(style.Render(rightCorner))
	//  NORMAL   master  󰀦 1   LSP                                                                utf-8     go  51%   67:21

	return builder.String()
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

func (m *Model) SetNetworkIndex(networkIndex int) {
	m.networkIndex = networkIndex
}

func (m *Model) Set(receiverIndex, frequencyIndex *int) {
	m.receiverIndex = receiverIndex
	m.frequencyIndex = frequencyIndex
}

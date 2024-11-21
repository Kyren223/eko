package chat

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/v2/cursor"
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	"github.com/kyren223/eko/internal/client/api"
	"github.com/kyren223/eko/internal/client/ui/messagebox"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
)

type Model struct {
	messagebox messagebox.Model
	textarea   textarea.Model
	err        error
}

func New() Model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return Model{
		textarea:   ta,
		messagebox: messagebox.New(30, 20),
		err:        nil,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, api.GetMessages)
}

func (m Model) View() string {
	return fmt.Sprintf(
		"%s\n%s",
		m.messagebox.View(),
		m.textarea.View(),
	) + ""
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.messagebox.Viewport.Width = msg.Width
		m.messagebox.Viewport.Height = msg.Height - m.textarea.Height()
		m.messagebox.Viewport.GotoBottom()

		m.textarea.SetWidth(msg.Width)
		log.Println("resized to:", msg.Width, "x", msg.Height)

		var mbCmd, taCmd tea.Cmd
		m.messagebox, mbCmd = m.messagebox.Update(msg)
		m.textarea, taCmd = m.textarea.Update(msg)
		return m, tea.Batch(mbCmd, taCmd)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyEnter:
			value := m.textarea.Value()

			content := strings.TrimSpace(value)
			if content == "" {
				return m, nil
			}
			m.textarea.Reset()

			return m, api.SendMessage(content)

		default:
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}

	case packet.Payload:
		switch msg := msg.(type) {
		case *packet.Messages:
			m.messagebox.SetMessages(msg.Messages)

			var cmd tea.Cmd
			m.messagebox, cmd = m.messagebox.Update(msg)
			return m, cmd

		default:
			return m, nil
		}

	case api.AppendMessage:
		m.messagebox.AppendMessage(data.Message(msg))

		var cmd tea.Cmd
		m.messagebox, cmd = m.messagebox.Update(msg)
		return m, cmd

	case api.UserProfileUpdate:
		var cmd tea.Cmd
		m.messagebox, cmd = m.messagebox.Update(msg)
		return m, cmd

	case cursor.BlinkMsg:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

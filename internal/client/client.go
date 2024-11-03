package client

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kyren223/eko/internal/client/api"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui/messagebox"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
)

type BubbleTeaCloser struct {
	program *tea.Program
}

func (c BubbleTeaCloser) Close() error {
	c.program.Quit()
	return nil
}

func Run() {
	log.Println("client started")
	program := tea.NewProgram(initialModel(), tea.WithAltScreen())
	assert.AddFlush(BubbleTeaCloser{program})

	_, privKey, err := ed25519.GenerateKey(nil)
	assert.NoError(err, "private key gen should not error")

	gateway.Connect(context.Background(), program, privKey)
	if _, err := program.Run(); err != nil {
		log.Println(err)
	}
}

type model struct {
	messagebox messagebox.Model
	textarea   textarea.Model
	err        error
}

func initialModel() model {
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

	return model{
		textarea:   ta,
		messagebox: messagebox.New(30, 20),
		err:        nil,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, api.GetMessages)
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n%s",
		m.messagebox.View(),
		m.textarea.View(),
	) + ""
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

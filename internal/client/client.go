package client

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kyren223/eko/internal/client/api"
	"github.com/kyren223/eko/internal/client/gateway"
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
	viewport    viewport.Model
	messages    []string
	textarea    textarea.Model
	senderStyle lipgloss.Style
	err         error
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

	vp := viewport.New(30, 20)
	// vp.SetContent("Welcome to Eko!\n Type a message and press Enter to send.")

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return model{
		textarea:    ta,
		messages:    []string{},
		viewport:    vp,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:         nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n%s",
		m.viewport.View(),
		m.textarea.View(),
	) + ""
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - m.textarea.Height()
		m.viewport.GotoBottom()

		m.textarea.SetWidth(msg.Width)
		log.Println("resized to:", msg.Width, "x", msg.Height)

		var vpCmd, taCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		m.textarea, taCmd = m.textarea.Update(msg)
		return m, tea.Batch(vpCmd, taCmd)

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
			slices.SortFunc(msg.Messages, func(a, b data.Message) int {
				return a.CmpTimestamp(b)
			})

			m.messages = []string{}
			for _, message := range msg.Messages {
				m.messages = append(m.messages, message.Contents)
			}

			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			m.viewport.GotoBottom()

			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd

		default:
			return m, nil
		}

	case cursor.BlinkMsg:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

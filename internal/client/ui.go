package client

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kyren223/eko/pkg/assert"
)

func startUI() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Println("charm ui error:", err)
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
	vp.SetContent("Welcome to Eko!\n Type a message and press Enter to send.")

	ta.KeyMap.InsertNewline.SetEnabled(false)

	messages, err := getMessages()
	assert.NoError(err, "TODO HANDLE ERROR")
	vp.SetContent(strings.Join(messages, "\n"))

	return model{
		textarea:    ta,
		messages:    messages,
		viewport:    vp,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:         nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width)
		return m, tea.Printf("%dx%d", msg.Width, msg.Height)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			// Quit.
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case tea.KeyEnter:
			value := m.textarea.Value()

			content := strings.TrimSpace(value)
			if content == "" {
				// Don't send empty messages.
				return m, nil
			}
			assert.NoError(sendMessage(content), "TODO HANDLE ERROR")
			messages, err := getMessages()
			assert.NoError(err, "TODO HANDLE ERROR")

			// m.messages = append(m.messages, m.senderStyle.Render("You: ")+content)
			m.messages = messages
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			m.textarea.Reset()
			m.viewport.GotoBottom()
			return m, nil
		default:
			// Send all other keypresses to the textarea.
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}

	case cursor.BlinkMsg:
		// Textarea should also process cursor blinks.
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	) + "\n\n"
}

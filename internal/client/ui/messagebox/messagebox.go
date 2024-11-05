package messagebox

import (
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kyren223/eko/internal/client/api"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/pkg/snowflake"
)

var (
	senderStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)
	timestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#585c62")).Italic(true)
)

type user struct {
	name       string
	isFetching bool
}

type Model struct {
	Viewport viewport.Model
	messages []data.Message
	users    map[snowflake.ID]user
}

func New(width, height int) Model {
	return Model{
		Viewport: viewport.New(width, height),
		messages: nil,
		users:    make(map[snowflake.ID]user),
	}
}

func (m Model) Init() tea.Cmd {
	return api.GetMessages
}

func (m Model) View() string {
	return m.Viewport.View()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	for id, user := range m.users {
		if !user.isFetching && user.name == "" {
			cmds = append(cmds, api.GetUserById(id))
			user.isFetching = true
			m.users[id] = user
		}
	}

	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case api.UserProfileUpdate:
		m.users[msg.ID] = user{msg.Name, false}
		m.UpdateMessages()
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) AppendMessage(message data.Message) {
	m.messages = append(m.messages, message)
	m.UpdateMessages()
}

func (m *Model) SetMessages(messages []data.Message) {
	m.messages = messages
	m.UpdateMessages()
}

func (m *Model) UpdateMessages() {
	for _, message := range m.messages {
		if _, exists := m.users[message.SenderID]; !exists {
			m.users[message.SenderID] = user{name: "", isFetching: false}
		}
	}

	m.sortMessages()
	m.updateContent()
	m.Viewport.GotoBottom()
}

func (m *Model) sortMessages() {
	slices.SortFunc(m.messages, func(a, b data.Message) int {
		if a.ID-b.ID < 0 {
			return -1
		} else {
			return 1
		}
	})
}

func (m *Model) updateContent() {
	if len(m.messages) == 0 {
		return
	}

	var builder strings.Builder

	for _, message := range m.messages {
		sender := m.users[message.SenderID].name
		builder.WriteString(senderStyle.Render(sender))
		builder.WriteByte(' ')

		timestamp := timestampFromID(message.ID)
		builder.WriteString(timestampStyle.Render(timestamp))
		builder.WriteByte('\n')

		builder.WriteString(message.Content)
		builder.WriteByte('\n')
	}

	m.Viewport.SetContent(builder.String()[:builder.Len()-1])
}

func timestampFromID(id snowflake.ID) string {
	localTime := time.UnixMilli(id.Time())
	return localTime.Format("02/01/2006 3:04 PM")
}

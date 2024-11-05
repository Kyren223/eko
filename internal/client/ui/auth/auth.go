package auth

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	authfield "github.com/kyren223/eko/internal/client/ui/auth/field"
)

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("198"))
	cursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("198"))
	noStyle      = lipgloss.NewStyle()
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F16265"))
	fieldStyle   = lipgloss.NewStyle().
			PaddingLeft(1).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#007E8A"))

	focusedButton = focusedStyle.Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("Submit"))
)

type Model struct {
	width  int
	height int

	focusIndex int
	fields     []authfield.Model
}

func New() Model {
	m := Model{
		fields: make([]authfield.Model, 3),
	}

	var field authfield.Model
	for i := range m.fields {
		field = authfield.New(32)
		field.Input.Cursor.Style = cursorStyle
		field.Style = fieldStyle
		field.ErrorStyle = errorStyle
		field.FocusedStyle = focusedStyle
		field.BlurredStyle = noStyle

		switch i {
		case 0:
			field.Input.Placeholder = "Username"
			field.Focus()
		case 1:
			field.Input.Placeholder = "Email"
			field.Input.CharLimit = 64
			field.Input.Validate = func(email string) error {
				if !strings.ContainsRune(email, '@') {
					return errors.New("missing @")
				}
				if !strings.ContainsRune(email, '.') {
					return errors.New("missing .")
				}
				proton := strings.Contains(email, "proton.me") || strings.Contains(email, "protonmail.com")
				gmail := strings.Contains(email, "gmail.com")
				outlook := strings.Contains(email, "hotmail.com")
				if !(proton || gmail || outlook) {
					return errors.New("unknown email provider")
				}
				return nil
			}
		case 2:
			field.Input.Placeholder = "Password"
			field.Input.EchoMode = textinput.EchoPassword
			field.Input.EchoCharacter = '*'
			// field.Input.EchoCharacter = '•'
		}

		m.fields[i] = field
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	var builder strings.Builder

	for i := range m.fields {
		builder.WriteString(m.fields[i].View())
		if i < len(m.fields)-1 {
			builder.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.focusIndex == len(m.fields) {
		button = &focusedButton
	}
	submit := lipgloss.NewStyle().
		PaddingLeft((lipgloss.Width(builder.String()) - lipgloss.Width(*button)) / 2).
		Render(*button)
	fmt.Fprintf(&builder, "\n\n%s", submit)

	width, height := lipgloss.Width(builder.String()), lipgloss.Height(builder.String())

	vp := viewport.New(50, 25)
	vp.SetContent(builder.String())
	vp.Style = vp.Style.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#0000FF")).
		// NOTE: Without -1 it wraps/truncates, not sure why
		Padding((vp.Height-height)/2-1, (vp.Width-width)/2-1)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		vp.View(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		key := msg.Type
		switch key {
		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyTab, tea.KeyShiftTab, tea.KeyEnter, tea.KeyUp, tea.KeyDown:
			// User pressed submit
			if key == tea.KeyEnter && m.focusIndex == len(m.fields) {
				return m, tea.Quit
			}

			// Cycle indexes
			if key == tea.KeyUp || key == tea.KeyShiftTab {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.fields) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.fields)
			}

			cmds := make([]tea.Cmd, len(m.fields))
			for i := 0; i <= len(m.fields)-1; i++ {
				if i == m.focusIndex {
					cmds[i] = m.fields[i].Focus()
				} else {
					m.fields[i].Blur()
				}
			}

			return m, tea.Batch(cmds...)
		}
	}

	cmd := m.updateInputs(msg)

	return m, cmd
}

func (m *Model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.fields))

	// Only fields with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.fields {
		m.fields[i], cmds[i] = m.fields[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

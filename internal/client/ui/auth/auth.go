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
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#007E8A"))

	focusedButton = focusedStyle.Render("[ sign-up ]")
	blurredButton = fmt.Sprintf("[ %s ]", lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("sign-up"))

	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#54D7A9"))
	titleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#5874FF"))
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
			field.Header = "Username"
			field.Input.Placeholder = "Username"
			field.Focus()
			field.Input.Validate = func(username string) error {
				if len(username) == 0 {
					return errors.New("* required field")
				}
				return nil
			}
		case 1:
			field.Header = "Email (Optional)"
			field.Input.Placeholder = "Email"
			field.Input.CharLimit = 64
			field.Input.Validate = func(email string) error {
				if len(email) == 0 {
					return errors.New("* required field")
				}
				var errs []string
				if !strings.ContainsRune(email, '@') {
					errs = append(errs, "- missing @")
				}
				if !strings.ContainsRune(email, '.') {
					errs = append(errs, "- missing .")
				}
				proton := strings.Contains(email, "proton.me") || strings.Contains(email, "protonmail.com")
				gmail := strings.Contains(email, "gmail.com")
				outlook := strings.Contains(email, "hotmail.com")
				if !(proton || gmail || outlook) {
					errs = append(errs, "- unknown email provider")
				}
				if errs != nil {
					return errors.New(strings.Join(errs, "\n"))
				}
				return nil
			}
		case 2:
			field.Header = "Password"
			field.Input.Placeholder = "Password"
			field.Input.EchoMode = textinput.EchoPassword
			field.Input.EchoCharacter = '*'
			// field.Input.EchoCharacter = '•'
			field.Input.Validate = func(password string) error {
				if len(password) == 0 {
					return errors.New("* required field")
				}
				return nil
			}
		}

		field.Header = headerStyle.Render(field.Header)
		m.fields[i] = field
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	var builder strings.Builder

	builder.WriteRune('\n')
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
		Render(*button)
	fmt.Fprintf(&builder, "\n\n%s", submit)

	title := titleStyle.Render(`
____ _ ____ _  _    _  _ ___  
[__  | | __ |\ | __ |  | |__] 
___] | |__] | \|    |__| |    
	`)

	// Tiny offset so odd numbers will have the extra char on the right, ie: 12 <thing> 13
	content := lipgloss.JoinVertical(lipgloss.Center-0.01, title, builder.String())
	width := lipgloss.Width(content)

	vp := viewport.New(50, 25)
	vp.SetContent(content)
	vp.Style = vp.Style.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#007E8A")).
		// NOTE: Without -1 it wraps/truncates, not sure why
		Padding(0, (vp.Width-width)/2-1)

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
				var cmds []tea.Cmd
				for i, field := range m.fields {
					if field.Input.Validate == nil {
						continue
					}
					field.Input.Err = field.Input.Validate(field.Input.Value())
					if field.Input.Err != nil {
						var cmd tea.Cmd
						m.fields[i], cmd = field.Update(msg)
						cmds = append(cmds, cmd)
					}
				}
				if len(cmds) == 0 {
					return m, tea.Quit
				}
				return m, tea.Batch(cmds...)
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

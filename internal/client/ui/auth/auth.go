package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kyren223/eko/internal/client/ui"
	authfield "github.com/kyren223/eko/internal/client/ui/auth/field"
	"github.com/kyren223/eko/internal/client/ui/choicepopup"
	"github.com/kyren223/eko/pkg/assert"
)

const (
	usernameField = iota
	privateKeyField
	passphraseField
	passphraseConfirmField
)

var (
	grayStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("198"))
	cursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("198"))
	noStyle      = lipgloss.NewStyle()
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F16265"))
	fieldStyle   = lipgloss.NewStyle().
			PaddingLeft(1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#007E8A"))

	focusedSignupButton = focusedStyle.Render("[ sign-up ]")
	focusedSigninButton = focusedStyle.Render("[ sign-in ]")
	blurredSignupButton = fmt.Sprintf("[ %s ]", grayStyle.Render("sign-up"))
	blurredSigninButton = fmt.Sprintf("[ %s ]", grayStyle.Render("sign-in"))

	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#54D7A9"))
	titleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#5874FF"))

	signupTitle = titleStyle.Render(`
____ _ ____ _  _    _  _ ___  
[__  | | __ |\ | __ |  | |__] 
___] | |__] | \|    |__| |    
		`)

	signinTitle = titleStyle.Render(`
____ _ ____ _  _    _ _  _
[__  | | __ |\ | __ | |\ |
___] | |__] | \|    | | \|
		`)

	revealIcon  = lipgloss.NewStyle().PaddingLeft(1).Render("󰈈 ")
	concealIcon = lipgloss.NewStyle().PaddingLeft(1).Render("󰈉 ")

	popupStyle            = lipgloss.NewStyle().Border(lipgloss.ThickBorder())
	choiceSelectedStyle   = lipgloss.NewStyle().Background(focusedStyle.GetForeground()).Padding(0, 1).Margin(0, 1)
	choiceUnselectedStyle = lipgloss.NewStyle().Background(grayStyle.GetForeground()).Padding(0, 1).Margin(0, 1)
)

type Model struct {
	width  int
	height int

	focusIndex int
	fields     []authfield.Model

	signup bool

	popup *choicepopup.Model
}

func New() Model {
	m := Model{
		fields: make([]authfield.Model, 4),
	}

	for i := range m.fields {
		field := authfield.New(48)
		field.Input.Cursor.Style = cursorStyle
		field.Style = fieldStyle
		field.ErrorStyle = errorStyle
		field.FocusedStyle = focusedStyle
		field.BlurredStyle = noStyle

		switch i {
		case usernameField:
			field.Header = headerStyle.Render("Username")
			field.Input.Placeholder = "Username"
			field.Input.CharLimit = 48
			field.Input.Validate = func(username string) error {
				if len(username) == 0 {
					return errors.New("Required")
				}
				return nil
			}
		case privateKeyField:
			field.Header = headerStyle.Render("Private Key")
			field.Input.Placeholder = "Path to Private Key"
			field.Input.CharLimit = 100
			field.Input.Validate = func(privKey string) error {
				if len(privKey) == 0 {
					return errors.New("Required")
				}
				return nil
			}

		case passphraseField:
			field.Header = headerStyle.Render("Passphrase (Optional)")
			field.Input.Placeholder = "Passphrase"
			field.SetRevealIcon(revealIcon)
			field.SetConcealIcon(concealIcon)
			field.SetRevealed(false)
			field.Input.EchoCharacter = '*'

		case passphraseConfirmField:
			field.Header = headerStyle.Render("Passphrase Confirm")
			field.Input.Placeholder = "Repeated Passphrase"
			field.SetRevealIcon(revealIcon)
			field.SetConcealIcon(concealIcon)
			field.SetRevealed(false)
			field.Input.EchoCharacter = '*'
		}

		m.fields[i] = field
	}

	m.SetSignup(false)
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	var builder strings.Builder

	builder.WriteRune('\n')
	for i, field := range m.fields {
		if field.Visisble {
			builder.WriteString(m.fields[i].View())
		} else {
			builder.WriteString("\n\n")
		}
		if i < len(m.fields)-1 {
			builder.WriteRune('\n')
		}
	}

	var button *string
	if m.signup {
		if m.focusIndex == len(m.fields) {
			button = &focusedSignupButton
		} else {
			button = &blurredSignupButton
		}
	} else {
		if m.focusIndex == len(m.fields) {
			button = &focusedSigninButton
		} else {
			button = &blurredSigninButton
		}
	}
	fmt.Fprintf(&builder, "\n\n%s", *button)

	var title string
	if m.signup {
		title = signupTitle
	} else {
		title = signinTitle
	}

	// Tiny offset so odd numbers will have the extra char on the right, ie: 12 <thing> 13
	content := lipgloss.JoinVertical(lipgloss.Center-0.01, title, builder.String())
	width := lipgloss.Width(content)

	vp := viewport.New(64, 23)
	vp.SetContent(content)
	vp.Style = vp.Style.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#007E8A")).
		// NOTE: Without -1 it wraps/truncates, not sure why
		Padding(0, (vp.Width-width)/2-1, 1)

	result := lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		vp.View(),
	)

	if m.popup != nil {
		popup := m.popup.View()
		x := (m.width - lipgloss.Width(popup)) / 2
		y := (m.height - lipgloss.Height(popup)) / 2
		result = ui.PlaceOverlay(x, y, popup, result)
	}

	return result
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

		case tea.KeyCtrlT:
			if m.popup == nil && (m.focusIndex == passphraseField || m.focusIndex == passphraseConfirmField) {
				m.fields[m.focusIndex].SetRevealed(!m.fields[m.focusIndex].Revealed())
			}
			return m, nil

		case tea.KeyEnter:
			if m.popup != nil {
				_, choice := m.popup.Select()
				if choice == "sign-up" {
					m.popup = nil
					return m, m.SetSignup(true)
				}
				if choice == "sign-in" {
					m.popup = nil
					return m, m.SetSignup(false)
				}
				if choice == "overwrite" {
					// TODO: how should I handle this
					// I need to somehow use this to notify that the file
					// should be overwritten, or maybe I should remove this option?
					// And make sure the user manually deletes/renames/moves the file
					// So nobody can claim that this deleted their SSH keys
					// (or more likely: I won't accidentally delete my SSH keys)
					m.popup = nil
					return m, nil
				}
				if choice == "cancel" {
					m.popup = nil
					return m, nil
				}
				assert.Never("unexpected choice", "choice", choice)
			}

			pressedButton := key == tea.KeyEnter && m.focusIndex == len(m.fields)
			if pressedButton {
				return m, m.ButtonPressed(msg)
			}

		case tea.KeyShiftTab, tea.KeyUp:
			if m.popup != nil {
				m.popup.ScrollLeft()
				return m, nil
			}
			m.CycleBack()
			return m, m.updateFocus()

		case tea.KeyDown, tea.KeyTab:
			if m.popup != nil {
				m.popup.ScrollRight()
				return m, nil
			}
			m.CycleForward()
			return m, m.updateFocus()
		}
	}

	if m.popup != nil {
		return m, nil
	}

	cmds := make([]tea.Cmd, len(m.fields))
	for i := range m.fields {
		m.fields[i], cmds[i] = m.fields[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) CycleBack() {
	for i := 0; ; i++ {
		m.focusIndex--

		if m.focusIndex < 0 {
			m.focusIndex = len(m.fields)
		}

		if m.focusIndex == len(m.fields) || m.fields[m.focusIndex].Visisble {
			break
		}

		assert.Assert(i < 2*len(m.fields), "CycleBack infinite loop", "i", i)
	}
}

func (m *Model) CycleForward() {
	for i := 0; ; i++ {
		m.focusIndex++

		if m.focusIndex > len(m.fields) {
			m.focusIndex = 0
		}

		if m.focusIndex == len(m.fields) || m.fields[m.focusIndex].Visisble {
			break
		}

		assert.Assert(i < 2*len(m.fields), "CycleForward infinite loop", "i", i)
	}
}

func (m *Model) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.fields))
	for i := 0; i < len(m.fields); i++ {
		if i == m.focusIndex {
			cmds[i] = m.fields[i].Focus()
		} else {
			m.fields[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *Model) SetSignup(signup bool) tea.Cmd {
	m.signup = signup
	for i, field := range m.fields {
		visible := m.signup || (i == privateKeyField || i == passphraseField)
		field.Visisble = visible
		field.Input.Err = nil
		m.fields[i] = field
	}
	m.focusIndex = -1
	m.CycleForward()
	return m.updateFocus()
}

func (m *Model) ButtonPressed(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for i, field := range m.fields {
		if !field.Visisble || field.Input.Validate == nil {
			continue
		}
		field.Input.Err = field.Input.Validate(field.Input.Value())
		if field.Input.Err != nil {
			var cmd tea.Cmd
			m.fields[i], cmd = field.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	if len(cmds) != 0 {
		return tea.Batch(cmds...)
	}

	if m.signup {
		return m.Signup()
	} else {
		return m.signin()
	}
}

func (m *Model) Signup() tea.Cmd {
	// So if signup is on and private key file exists suggest to either:
	// 1. Overwrite it 2. Switch to sign-in 3. Cancel
	privateKeyFilepath := m.fields[privateKeyField].Input.Value()
	_, err := os.ReadFile(privateKeyFilepath)
	if errors.Is(err, os.ErrNotExist) {
		return tea.Quit
	}

	if err != nil {
		m.fields[privateKeyField].Input.Err = errors.Unwrap(err)
		assert.NotNil(errors.Unwrap(err), "there should always be an error to unwrap", "err", err)
		return nil
	}

	content := fmt.Sprintf("File '%s' exist.\nDo you want to overwrite or sign-in instead?", privateKeyFilepath)
	m.popup = createPopup(content, []string{"sign-in", "overwrite"}, []string{"cancel"})
	return nil
}

func (m *Model) signin() tea.Cmd {
	privateKeyFilepath := m.fields[privateKeyField].Input.Value()
	_, err := os.ReadFile(privateKeyFilepath)
	if errors.Is(err, os.ErrNotExist) {
		content := fmt.Sprintf("File '%s' doesn't exist.\nDo you want to sign-up instead?", privateKeyFilepath)
		m.popup = createPopup(content, []string{"sign-up"}, []string{"cancel"})
		return nil
	}
	if err != nil {
		m.fields[privateKeyField].Input.Err = errors.Unwrap(err)
		assert.NotNil(errors.Unwrap(err), "there should always be an error to unwrap", "err", err)
		return nil
	}
	return tea.Quit
}

func createPopup(content string, leftChoices, rightChoices []string) *choicepopup.Model {
	content = lipgloss.NewStyle().Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, false, true).
		Render(content)

	popup := choicepopup.New(lipgloss.Width(content), lipgloss.Height(content)+1)

	popup.SetContent(content)
	popup.SetChoices(leftChoices, rightChoices)
	popup.Cycle = true

	popup.Style = popupStyle
	popup.SelectedStyle = choiceSelectedStyle
	popup.UnselectedStyle = choiceUnselectedStyle

	return &popup
}

func test() {
	// pubKey, privKey, err := ed25519.GenerateKey(nil)
	// sshPrivKey, err := ssh.NewSignerFromSigner(privKey)
	// ssh.MarshalPrivateKey()
	// ssh.MarshalPrivateKey()
	// ssh.ParseRawPrivateKey()
	// ssh.ParseRawPrivateKeyWithPassphrase()
}

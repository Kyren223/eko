package auth

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/crypto/ssh"

	"github.com/kyren223/eko/internal/client/ui"
	authfield "github.com/kyren223/eko/internal/client/ui/auth/field"
	"github.com/kyren223/eko/internal/client/ui/choicepopup"
	"github.com/kyren223/eko/internal/client/ui/loadscreen"
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
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#5874FF"))
	cursorStyle  = focusedStyle
	noStyle      = lipgloss.NewStyle()
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F16265"))

	fieldBlurredStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#007E8A"))
	fieldFocusedStyle = fieldBlurredStyle.BorderForeground(focusedStyle.GetForeground()).Border(lipgloss.ThickBorder())

	focusedSignupButton = focusedStyle.Bold(true).Render("[ SIGN-UP ]")
	focusedSigninButton = focusedStyle.Bold(true).Render("[ SIGN-IN ]")
	blurredSignupButton = fmt.Sprintf("[ %s ]", grayStyle.Render("sign-up"))
	blurredSigninButton = fmt.Sprintf("[ %s ]", grayStyle.Render("sign-in"))

	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#54D7A9"))
	titleStyle  = focusedStyle.Bold(true)

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
	choiceSelectedStyle   = lipgloss.NewStyle().Background(lipgloss.Color("#0029f5")).Padding(0, 1).Margin(0, 1)
	choiceUnselectedStyle = lipgloss.NewStyle().Background(grayStyle.GetForeground()).Padding(0, 1).Margin(0, 1)
)

type Model struct {
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
		field.BlurredStyle = fieldBlurredStyle
		field.FocusedStyle = fieldFocusedStyle
		field.ErrorStyle = errorStyle
		field.FocusedTextStyle = focusedStyle
		field.BlurredTextStyle = noStyle

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
		ui.Width, ui.Height,
		lipgloss.Center, lipgloss.Center,
		vp.View(),
	)

	if m.popup != nil {
		popup := m.popup.View()
		x := (ui.Width - lipgloss.Width(popup)) / 2
		y := (ui.Height - lipgloss.Height(popup)) / 2
		result = ui.PlaceOverlay(x, y, popup, result)
	}

	return result
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
	username := m.fields[usernameField].Input.Value()
	assert.Assert(len(username) != 0, "username must not be empty")
	passphrase := m.fields[passphraseField].Input.Value()
	confirmation := m.fields[passphraseConfirmField].Input.Value()

	hasPassphrase := len(passphrase) != 0
	hasConfirmation := len(confirmation) != 0
	if hasPassphrase && !hasConfirmation {
		m.fields[passphraseConfirmField].Input.Err = errors.New("Confirmation required")
		return nil
	} else if !hasPassphrase && hasConfirmation {
		m.fields[passphraseField].Input.Err = errors.New("Empty passphrase")
		return nil
	} else if hasPassphrase && hasConfirmation && passphrase != confirmation {
		m.fields[passphraseConfirmField].Input.Err = errors.New("Passphrase mismatch")
		return nil
	}

	privateKeyFilepath := expandPath(m.fields[privateKeyField].Input.Value())
	err := os.MkdirAll(filepath.Dir(privateKeyFilepath), 0o755)
	if err != nil {
		m.fields[privateKeyField].Input.Err = errors.Unwrap(err)
		assert.NotNil(errors.Unwrap(err), "there should always be an error to unwrap", "err", err)
		return nil
	}
	file, err := os.OpenFile(privateKeyFilepath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		info, e := os.Stat(privateKeyFilepath)
		assert.NoError(e, "if file exists it should be fine to stat it")
		if info.IsDir() {
			m.fields[privateKeyField].Input.Err = errors.New("File is a directory")
			return nil
		}
		content := fmt.Sprintf("File '%s' exists.\nDo you want to sign-in instead?", privateKeyFilepath)
		m.popup = createPopup(content, []string{"sign-in"}, []string{"cancel"})
		return nil
	}
	if err != nil {
		m.fields[privateKeyField].Input.Err = errors.Unwrap(err)
		if errors.Unwrap(err).Error() == "is a directory" {
			m.fields[privateKeyField].Input.Err = errors.New("File is a directory")
		}
		log.Println("signup open file error:", err)
		assert.NotNil(errors.Unwrap(err), "there should always be an error to unwrap", "err", err)
		return nil
	}
	defer file.Close()

	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		m.fields[privateKeyField].Input.Err = errors.New("Failed private key generation")
		log.Println("ed25519 generate key error:", err)
		return nil
	}
	var pemBlock *pem.Block
	if hasPassphrase {
		pemBlock, err = ssh.MarshalPrivateKeyWithPassphrase(privKey, username, []byte(passphrase))
	} else {
		pemBlock, err = ssh.MarshalPrivateKey(privKey, username)
	}
	if err != nil {
		m.fields[privateKeyField].Input.Err = errors.New("Failed private key marshaling")
		log.Println("ssh marshaling error:", err)
		return nil
	}
	err = pem.Encode(file, pemBlock)
	if err != nil {
		m.fields[privateKeyField].Input.Err = errors.New("Failed writing to disk")
		log.Println("pem encoding to file error:", err)
		return nil
	}

	return authenticate(privKey)
}

func (m *Model) signin() tea.Cmd {
	privateKeyFilepath := expandPath(m.fields[privateKeyField].Input.Value())
	file, err := os.ReadFile(privateKeyFilepath)
	if errors.Is(err, os.ErrNotExist) {
		content := fmt.Sprintf("File '%s' doesn't exist.\nDo you want to sign-up instead?", privateKeyFilepath)
		m.popup = createPopup(content, []string{"sign-up"}, []string{"cancel"})
		return nil
	}
	if err != nil {
		m.fields[privateKeyField].Input.Err = errors.Unwrap(err)
		if errors.Unwrap(err).Error() == "is a directory" {
			m.fields[privateKeyField].Input.Err = errors.New("File is a directory")
		}
		assert.NotNil(errors.Unwrap(err), "there should always be an error to unwrap", "err", err)
		return nil
	}

	var privateKey any
	passphrase := m.fields[passphraseField].Input.Value()

	if len(passphrase) == 0 {
		privateKey, err = ssh.ParseRawPrivateKey(file)
		if err, ok := err.(*ssh.PassphraseMissingError); ok {
			m.fields[passphraseField].Input.Err = errors.New("Missing passphrase")
			log.Println("passphrase missing:", err)
			return nil
		}
		if err != nil {
			m.fields[privateKeyField].Input.Err = errors.New("Invalid private key file format")
			log.Println("passphrase error:", err)
			return nil
		}
	} else {
		privateKey, err = ssh.ParseRawPrivateKeyWithPassphrase(file, []byte(passphrase))
		if err == x509.IncorrectPasswordError {
			m.fields[passphraseField].Input.Err = errors.New("Incorrect Passphrase")
			return nil
		}
		if err != nil && (err.Error() == "ssh: not an encrypted key" || err.Error() == "ssh: key is not password protected") {
			privateKey, err = ssh.ParseRawPrivateKey(file)
		}
		if err != nil {
			m.fields[privateKeyField].Input.Err = errors.New("Invalid private key file format")
			log.Println("passphrase error:", err)
			return nil
		}
	}

	privKey, ok := privateKey.(*ed25519.PrivateKey)
	if !ok {
		m.fields[privateKeyField].Input.Err = errors.New("Must be ed25519")
		keyType := reflect.TypeOf(privateKey)
		log.Println("incorrect private key type, got:", keyType.String(), reflect.ValueOf(privateKey).String())
		return nil
	}

	return authenticate(*privKey)
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

func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		assert.NoError(err, "home directory should always be defined")
		if !strings.HasPrefix(path, "~/") {
			return home
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func authenticate(privKey ed25519.PrivateKey) tea.Cmd {
	return ui.Transition(loadscreen.New())
}


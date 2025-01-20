package usersettings

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/client/ui/field"
	"github.com/kyren223/eko/internal/client/ui/layouts/flex"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
)

var (
	width = 48

	style = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		Padding(1, 4).
		Align(lipgloss.Center, lipgloss.Center)

	headerStyle = lipgloss.NewStyle().Foreground(colors.Turquoise)

	fieldBlurredStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colors.DarkCyan)
	fieldFocusedStyle = fieldBlurredStyle.
				BorderForeground(colors.Focus).
				Border(lipgloss.ThickBorder())

	blurredUpdate = lipgloss.NewStyle().
			Background(colors.Gray).Padding(0, 1).Render("Update User Settings")
	focusedUpdate = lipgloss.NewStyle().
			Background(colors.Blue).Padding(0, 1).Render("Update User Settings")
)

const (
	NameField = iota
	Description
	PrivateField
	UpdateField
	FieldCount
)

type Model struct {
	name        field.Model
	description field.Model
	privateDM   bool
	update      string

	selected  int
	nameWidth int
}

func New() Model {
	user, ok := state.State.Users[*state.UserID]
	assert.Assert(ok, "user should always exist when connected to server")

	name := field.New(width)
	name.Header = "Username"
	name.HeaderStyle = headerStyle
	name.FocusedStyle = fieldFocusedStyle
	name.BlurredStyle = fieldBlurredStyle
	name.ErrorStyle = lipgloss.NewStyle().Foreground(colors.Error)
	name.Input.CharLimit = width
	name.Focus()
	name.Input.Validate = func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("cannot be empty")
		}
		return nil
	}
	nameWidth := lipgloss.Width(name.View())
	name.Input.SetValue(user.Name)

	description := field.New(width)
	description.Header = "Description"
	description.HeaderStyle = headerStyle
	description.FocusedStyle = fieldFocusedStyle
	description.BlurredStyle = fieldBlurredStyle
	description.ErrorStyle = lipgloss.NewStyle().Foreground(colors.Error)
	description.Input.CharLimit = width
	description.Input.SetValue(user.Description)

	return Model{
		name:        name,
		description: description,
		privateDM:   !user.IsPublicDM,
		update:      blurredUpdate,
		selected:    0,
		nameWidth:   nameWidth,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	name := m.name.View()
	description := m.description.View()

	privateStyle := lipgloss.NewStyle().PaddingLeft(1)
	if m.selected == PrivateField {
		privateStyle = privateStyle.Foreground(colors.Focus)
	}
	private := "[ ] Private DMs"
	if m.privateDM {
		private = "[x] Private DMs"
	}
	private = privateStyle.Render(private)

	update := lipgloss.NewStyle().Width(m.nameWidth).Align(lipgloss.Center).Render(m.update)

	content := flex.NewVertical(name, description, private, update).WithGap(1).View()
	return style.Render(content)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.Type
		switch key {
		case tea.KeyTab:
			return m, m.cycle(1)
		case tea.KeyShiftTab:
			return m, m.cycle(-1)

		default:
			var cmd tea.Cmd
			switch m.selected {
			case NameField:
				m.name, cmd = m.name.Update(msg)
			case Description:
				m.description, cmd = m.description.Update(msg)
			}
			return m, cmd
		}
	}

	return m, nil
}

func (m *Model) cycle(step int) tea.Cmd {
	m.selected += step
	if m.selected < 0 {
		m.selected = FieldCount - 1
	} else {
		m.selected %= FieldCount
	}
	return m.updateFocus()
}

func (m *Model) updateFocus() tea.Cmd {
	m.name.Blur()
	m.description.Blur()
	m.update = blurredUpdate
	switch m.selected {
	case NameField:
		return m.name.Focus()
	case Description:
		return m.description.Focus()
	case PrivateField:
		return nil
	case UpdateField:
		m.update = focusedUpdate
		return nil
	default:
		assert.Never("missing switch statement field in update focus", "selected", m.selected)
		return nil
	}
}

func (m *Model) Select() tea.Cmd {
	if m.selected == PrivateField {
		m.privateDM = !m.privateDM
		return nil
	}

	if m.selected != UpdateField {
		return nil
	}

	m.name.Input.Err = m.name.Input.Validate(m.name.Input.Value())
	if m.name.Input.Err != nil {
		return nil
	}

	return gateway.Send(&packet.SetUserData{
		Data: nil,
		User: &data.User{
			ID:          *state.UserID,
			Name:        m.name.Input.Value(),
			Description: m.description.Input.Value(),
			IsPublicDM:  !m.privateDM,
			IsDeleted:   false,
			PublicKey:   nil,
		},
	})
}

package networkjoin

import (
	"errors"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/client/ui/field"
	"github.com/kyren223/eko/internal/client/ui/layouts/flex"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
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

	blurredCreate = lipgloss.NewStyle().
			Background(colors.Gray).Padding(0, 1).Render("Join Network")
	focusedCreate = lipgloss.NewStyle().
			Background(colors.Blue).Padding(0, 1).Render("Join Network")
)

const (
	MaxIconLength = 2
	MaxHexDigits  = 6
)

const (
	NameField = iota
	CreateField
	FieldCount
)

type Model struct {
	name   field.Model
	create string

	selected  int
	nameWidth int
}

func New() Model {
	name := field.New(width)
	name.Header = "Network ID"
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
		if _, err := strconv.ParseInt(s, 10, 64); err != nil {
			return err
		}

		return nil
	}

	nameWidth := lipgloss.Width(name.View())

	return Model{
		name:      name,
		create:    blurredCreate,
		nameWidth: nameWidth,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	name := m.name.View()
	create := lipgloss.NewStyle().
		Width(m.nameWidth).Align(lipgloss.Center).
		Render(m.create)

	content := flex.NewVertical(name, create).WithGap(1).View()
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
	m.create = blurredCreate
	switch m.selected {
	case NameField:
		return m.name.Focus()
	case CreateField:
		m.create = focusedCreate
		return nil
	default:
		assert.Never("missing switch statement field in update focus", "selected", m.selected)
		return nil
	}
}

func (m *Model) Select() tea.Cmd {
	if m.selected != CreateField {
		return nil
	}

	m.name.Input.Err = m.name.Input.Validate(m.name.Input.Value())
	if m.name.Input.Err != nil {
		return nil
	}
	name := m.name.Input.Value()
	id, err := strconv.ParseInt(name, 10, 64)
	assert.NoError(err, "input is already validated to be valid")

	yes := true
	request := packet.SetNetworkUser{
		Member:    &yes,
		Admin:     nil,
		Muted:     nil,
		Banned:    nil,
		BanReason: nil,
		Network:   snowflake.ID(id),
		User:      *state.State.UserID,
	}
	return gateway.Send(&request)
}

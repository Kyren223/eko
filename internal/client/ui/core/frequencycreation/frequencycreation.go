package frequencycreation

import (
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui/colors"
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
	focusStyle  = lipgloss.NewStyle().Foreground(colors.Focus)

	fieldBlurredStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colors.DarkCyan)
	fieldFocusedStyle = fieldBlurredStyle.
				BorderForeground(colors.Focus).
				Border(lipgloss.ThickBorder())

	underlineStyle = func(s string, width int, color lipgloss.Color) string {
		underline := lipgloss.NewStyle().Foreground(color).
			Render(strings.Repeat(lipgloss.ThickBorder().Bottom, width))
		return lipgloss.JoinVertical(lipgloss.Left, s, underline)
	}

	colorHeader = headerStyle.Bold(true).Render(" Color # ")

	blurredCreate = lipgloss.NewStyle().
			Background(colors.Gray).Padding(0, 1).Render("Create Frequency")
	focusedCreate = lipgloss.NewStyle().
			Background(colors.Blue).Padding(0, 1).Render("Create Frequency")

	permsHeader = lipgloss.NewStyle().
			Foreground(colors.Turquoise).
			Render("Permissions for non-admins:")
	leftpad = 1
)

const (
	MaxHexDigits = 6
)

const (
	NameField = iota
	ColorField
	ReadWriteField
	ReadOnlyField
	NoAccessField
	CreateField
	FieldCount
)

type Model struct {
	name             field.Model
	precomputedStyle lipgloss.Style
	lastColor        lipgloss.Color
	create           string
	color            textinput.Model
	perms            int
	nameWidth        int
	selected         int
	network          snowflake.ID
}

func New(network snowflake.ID) Model {
	name := field.New(width)
	name.Header = "Frequency Name"
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

	color := textinput.New()
	color.Prompt = ""
	color.CharLimit = MaxHexDigits
	color.Placeholder = "000000"
	color.Validate = func(s string) error {
		if len(s) != MaxHexDigits {
			return errors.New("err")
		}
		return nil
	}
	color.SetValue(string(colors.White)[1:])

	return Model{
		name:      name,
		color:     color,
		lastColor: lipgloss.Color("#" + color.Value()),
		perms:     packet.PermReadWrite,
		create:    blurredCreate,

		nameWidth:        nameWidth,
		precomputedStyle: lipgloss.NewStyle().Width(nameWidth / 3),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	name := m.name.View()

	color := colors.Gray
	if m.color.Err != nil {
		color = colors.Error
	} else if m.selected == ColorField {
		color = colors.Focus
	}
	colorInput := underlineStyle(m.color.View(), MaxHexDigits, color)
	colorInput = lipgloss.NewStyle().Width(MaxHexDigits + 1).Render(colorInput)
	colorIndicator := lipgloss.NewStyle().Foreground(m.lastColor).Render("■")
	colorText := lipgloss.JoinHorizontal(lipgloss.Top, colorHeader, colorInput, colorIndicator)

	readWrite := "[ ] Read & Write"
	if m.perms == packet.PermReadWrite {
		readWrite = "[x] Read & Write"
	}
	if m.selected == ReadWriteField {
		readWrite = focusStyle.Render(readWrite)
	}

	readOnly := "[ ] Read Only"
	if m.perms == packet.PermRead {
		readOnly = "[x] Read Only"
	}
	if m.selected == ReadOnlyField {
		readOnly = focusStyle.Render(readOnly)
	}

	noAccess := "[ ] No Access"
	if m.perms == packet.PermNoAccess {
		noAccess = "[x] No Access"
	}
	if m.selected == NoAccessField {
		noAccess = focusStyle.Render(noAccess)
	}

	width := lipgloss.Width(noAccess) + lipgloss.Width(readOnly) + lipgloss.Width(readWrite)
	padding := (m.nameWidth - (leftpad * 2) - width) / 2

	readWrite = lipgloss.NewStyle().PaddingRight(padding).Render(readWrite)
	readOnly = lipgloss.NewStyle().PaddingRight(padding).Render(readOnly)

	perms := lipgloss.JoinHorizontal(lipgloss.Top, readWrite, readOnly, noAccess)
	perms = lipgloss.JoinVertical(lipgloss.Left, permsHeader, perms)
	perms = lipgloss.NewStyle().PaddingLeft(leftpad).Render(perms)

	create := lipgloss.NewStyle().Width(m.nameWidth).Align(lipgloss.Center).Render(m.create)

	content := flex.NewVertical(name, perms, colorText, create).WithGap(1).View()
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
			case ColorField:
				oldValue := m.color.Value()
				position := m.color.Position()
				m.color, cmd = m.color.Update(msg)
				newValue := m.color.Value()

				hex := "0123456789abcdefABCDEF"
				invalid := false
				for _, c := range newValue {
					if !strings.ContainsRune(hex, c) {
						invalid = true
						break
					}
				}

				if invalid {
					m.color.SetValue(oldValue)
					m.color.SetCursor(position)
				} else if len(m.color.Value()) == 6 {
					m.lastColor = lipgloss.Color("#" + m.color.Value())
				}

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
	m.color.Blur()
	m.create = blurredCreate
	switch m.selected {
	case NameField:
		return m.name.Focus()
	case ColorField:
		return m.color.Focus()
	case NoAccessField, ReadOnlyField, ReadWriteField:
		return nil
	case CreateField:
		m.create = focusedCreate
		return nil
	default:
		assert.Never("missing switch statement field in update focus", "selected", m.selected)
		return nil
	}
}

func (m *Model) Select() tea.Cmd {
	switch m.selected {
	case NoAccessField:
		m.perms = packet.PermNoAccess
		return nil
	case ReadOnlyField:
		m.perms = packet.PermRead
		return nil
	case ReadWriteField:
		m.perms = packet.PermReadWrite
		return nil
	}

	if m.selected != CreateField {
		return nil
	}

	m.name.Input.Err = m.name.Input.Validate(m.name.Input.Value())
	m.color.Err = m.color.Validate(m.color.Value())
	if m.name.Input.Err != nil || m.color.Err != nil {
		return nil
	}

	request := packet.CreateFrequency{
		Network:  m.network,
		Name:     m.name.Input.Value(),
		HexColor: "#" + m.color.Value(),
		Perms:    m.perms,
	}
	return gateway.Send(&request)
}

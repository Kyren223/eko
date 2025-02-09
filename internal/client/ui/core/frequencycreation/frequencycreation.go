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

	padding = 4

	headerStyle = func() lipgloss.Style { return lipgloss.NewStyle().Foreground(colors.Turquoise) }
	focusStyle  = func() lipgloss.Style {
		return lipgloss.NewStyle().Background(colors.Background).Foreground(colors.Focus)
	}

	underlineStyle = func(s string, width int, color lipgloss.Color) string {
		underline := strings.Repeat("━", width)
		underline = lipgloss.NewStyle().Background(colors.Background).Foreground(color).
			Render(underline + " ")
		return lipgloss.NewStyle().Background(colors.Background).
			Render(lipgloss.JoinVertical(lipgloss.Left, s, underline))
	}

	colorHeader = func() string { return headerStyle().Bold(true).Render(" Color # ") }

	blurredCreate = func() string {
		return lipgloss.NewStyle().Padding(0, 1).
			Background(colors.Gray).Foreground(colors.White).
			Render("Create Frequency")
	}
	focusedCreate = func() string {
		return lipgloss.NewStyle().Padding(0, 1).
			Background(colors.Blue).Foreground(colors.White).
			Render("Create Frequency")
	}

	leftpad = 1
)

const (
	MaxHexDigits = 6
)

const (
	NameField = iota
	ReadWriteField
	ReadOnlyField
	NoAccessField
	ColorField
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
	blurredTextStyle := lipgloss.NewStyle().
		Background(colors.Background).Foreground(colors.White)
	focusedTextStyle := blurredTextStyle.Foreground(colors.Focus)

	fieldBlurredStyle := lipgloss.NewStyle().
		PaddingLeft(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colors.DarkCyan).
		BorderBackground(colors.Background).
		Background(colors.Background)
	fieldFocusedStyle := fieldBlurredStyle.
		Border(lipgloss.ThickBorder()).
		BorderForeground(colors.Focus)

	name := field.New(width)
	name.Header = "Frequency Name"
	name.HeaderStyle = headerStyle()
	name.FocusedStyle = fieldFocusedStyle
	name.BlurredStyle = fieldBlurredStyle
	name.FocusedTextStyle = focusedTextStyle
	name.BlurredTextStyle = blurredTextStyle
	name.ErrorStyle = lipgloss.NewStyle().Background(colors.Background).Foreground(colors.Error)
	name.Input.CharLimit = packet.MaxFrequencyName
	name.Focus()
	name.Input.Validate = func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("cannot be empty")
		}
		return nil
	}

	nameWidth := lipgloss.Width(name.View())

	color := textinput.New()
	color.PlaceholderStyle = blurredTextStyle.Foreground(colors.Gray)
	color.TextStyle = blurredTextStyle
	color.Cursor.Style = blurredTextStyle
	color.Cursor.TextStyle = blurredTextStyle
	color.Prompt = ""
	color.CharLimit = MaxHexDigits
	color.Placeholder = "000000"
	color.Validate = func(s string) error {
		if len(s) != MaxHexDigits {
			return errors.New("err")
		}
		return nil
	}
	color.SetValue(string(packet.DefaultFrequencyColor)[1:])

	return Model{
		name:      name,
		color:     color,
		lastColor: lipgloss.Color("#" + color.Value()),
		perms:     packet.PermReadWrite,
		create:    blurredCreate(),
		network:   network,

		nameWidth: nameWidth,
		precomputedStyle: lipgloss.NewStyle().PaddingRight(padding).
			Background(colors.Background).Foreground(colors.White).MarginBackground(colors.Background),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	name := m.name.View()

	colorStyle := lipgloss.NewStyle().Background(colors.Background).SetString("■\n ")
	color := colors.Gray
	if m.color.Err != nil {
		color = colors.Error
	} else if m.selected == ColorField {
		color = colors.Focus
	}
	c := lipgloss.NewStyle().Width(MaxHexDigits + 1).Background(colors.Background).Render(m.color.View())
	colorInput := underlineStyle(c, MaxHexDigits, color)
	colorInput = lipgloss.NewStyle().Render(colorInput)
	colorIndicator := colorStyle.Foreground(m.lastColor).String()
	colorText := lipgloss.JoinHorizontal(lipgloss.Top, colorHeader(), colorInput, colorIndicator)
	colorText = m.precomputedStyle.Width(m.nameWidth).Render(colorText)

	readWrite := "[ ] Read & Write"
	if m.perms == packet.PermReadWrite {
		readWrite = "[x] Read & Write"
	}
	if m.selected == ReadWriteField {
		readWrite = m.precomputedStyle.Foreground(colors.Focus).Render(readWrite)
	} else {
		readWrite = m.precomputedStyle.Render(readWrite)
	}

	readOnly := "[ ] Read Only"
	if m.perms == packet.PermRead {
		readOnly = "[x] Read Only"
	}
	if m.selected == ReadOnlyField {
		readOnly = m.precomputedStyle.Foreground(colors.Focus).Render(readOnly)
	} else {
		readOnly = m.precomputedStyle.Render(readOnly)
	}

	noAccess := "[ ] No Access"
	if m.perms == packet.PermNoAccess {
		noAccess = "[x] No Access"
	}
	if m.selected == NoAccessField {
		noAccess = m.precomputedStyle.
			PaddingRight(1).
			Foreground(colors.Focus).
			Render(noAccess)
	} else {
		noAccess = m.precomputedStyle.
			PaddingRight(1).
			Render(noAccess)
	}

	width := lipgloss.Width(noAccess) + lipgloss.Width(readOnly) + lipgloss.Width(readWrite)
	padding := (m.nameWidth - (leftpad * 2) - width) / 2

	readWrite = lipgloss.NewStyle().PaddingRight(padding).Render(readWrite)
	readOnly = lipgloss.NewStyle().PaddingRight(padding).Render(readOnly)

	permsHeader := lipgloss.NewStyle().
		Width(m.nameWidth - leftpad).
		Background(colors.Background).
		Foreground(colors.Turquoise).
		Render("Permissions for non-admins:")
	// perms := lipgloss.JoinHorizontal(lipgloss.Top, readWrite, readOnly, noAccess)
	perms := readWrite + readOnly + noAccess
	perms = lipgloss.JoinVertical(lipgloss.Left, permsHeader, perms)
	perms = lipgloss.NewStyle().PaddingLeft(leftpad).Render(perms)

	create := lipgloss.NewStyle().
		Width(m.nameWidth).
		Align(lipgloss.Center).
		Background(colors.Background).
		Render(m.create)

	content := flex.NewVertical(name, perms, colorText, create).WithGap(1).View()

	return lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		Padding(1, 4).
		Align(lipgloss.Center, lipgloss.Center).
		BorderBackground(colors.Background).
		BorderForeground(colors.White).
		Background(colors.Background).
		Foreground(colors.White).
		Render(content)
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
	m.create = blurredCreate()
	switch m.selected {
	case NameField:
		return m.name.Focus()
	case ColorField:
		return m.color.Focus()
	case NoAccessField, ReadOnlyField, ReadWriteField:
		return nil
	case CreateField:
		m.create = focusedCreate()
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

package networkcreation

import (
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/field"
	"github.com/kyren223/eko/internal/client/ui/layouts/flex"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
)

var (
	width = 48

	underlineStyle = func(s string, width int, color lipgloss.Color) string {
		underline := strings.Repeat("━", width)
		underline = lipgloss.NewStyle().Background(colors.Background).Foreground(color).
			Render(underline + " ")
		return lipgloss.NewStyle().Background(colors.Background).
			Render(lipgloss.JoinVertical(lipgloss.Left, s, underline))
	}

	headerStyle = func() lipgloss.Style { return lipgloss.NewStyle().Foreground(colors.Turquoise) }

	iconHeader    = func() string { return headerStyle().Bold(true).Render("Icon: ") }
	bgColorHeader = func() string { return headerStyle().Bold(true).Render("BG # ") }
	fgColorHeader = func() string { return headerStyle().Bold(true).Render(" FG # ") }

	blurredCreate = func() string {
		return lipgloss.NewStyle().Padding(0, 1).
			Background(colors.Gray).Foreground(colors.White).
			Render("Create Network")
	}
	focusedCreate = func() string {
		return lipgloss.NewStyle().Padding(0, 1).
			Background(colors.Blue).Foreground(colors.White).
			Render("Create Network")
	}
)

const (
	MaxIconLength = 2
	MaxHexDigits  = 6
)

const (
	NameField = iota
	FgColorField
	BgColorField
	IconField
	PrivateField
	CreateField
	FieldCount
)

type Model struct {
	precomputedStyle lipgloss.Style

	name    field.Model
	icon    textinput.Model
	bgColor textinput.Model
	fgColor textinput.Model
	private bool
	create  string

	selected  int
	nameWidth int

	lastFg lipgloss.Color
	lastBg lipgloss.Color
}

func New() Model {
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
	name.Header = "Network Name"
	name.HeaderStyle = headerStyle()
	name.FocusedStyle = fieldFocusedStyle
	name.BlurredStyle = fieldBlurredStyle
	name.FocusedTextStyle = focusedTextStyle
	name.BlurredTextStyle = blurredTextStyle
	name.ErrorStyle = lipgloss.NewStyle().Background(colors.Background).Foreground(colors.Error)
	name.Input.CharLimit = width
	name.Focus()
	name.Input.Validate = func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("cannot be empty")
		}
		return nil
	}

	nameWidth := lipgloss.Width(name.View())

	icon := textinput.New()
	icon.PlaceholderStyle = blurredTextStyle.Foreground(colors.Gray)
	icon.PromptStyle = blurredTextStyle
	icon.TextStyle = blurredTextStyle
	icon.Cursor.Style = blurredTextStyle
	icon.Cursor.TextStyle = blurredTextStyle
	icon.Prompt = ""
	icon.CharLimit = MaxIconLength
	icon.Placeholder = "ic"
	icon.Validate = func(s string) error {
		if len(s) == 0 {
			return errors.New("err")
		}
		return nil
	}

	bgColor := textinput.New()
	bgColor.PlaceholderStyle = blurredTextStyle.Foreground(colors.Gray)
	bgColor.TextStyle = blurredTextStyle
	bgColor.Cursor.Style = blurredTextStyle
	bgColor.Cursor.TextStyle = blurredTextStyle
	bgColor.Prompt = ""
	bgColor.CharLimit = MaxHexDigits
	bgColor.Placeholder = "000000"
	bgColor.Validate = func(s string) error {
		if len(s) != MaxHexDigits {
			return errors.New("err")
		}
		return nil
	}
	bgColor.SetValue(string(colors.Gray)[1:])

	fgColor := textinput.New()
	fgColor.PlaceholderStyle = blurredTextStyle.Foreground(colors.Gray)
	fgColor.TextStyle = blurredTextStyle
	fgColor.Cursor.Style = blurredTextStyle
	fgColor.Cursor.TextStyle = blurredTextStyle
	fgColor.Prompt = ""
	fgColor.CharLimit = MaxHexDigits
	fgColor.Placeholder = "000000"
	fgColor.Validate = func(s string) error {
		if len(s) != MaxHexDigits {
			return errors.New("err")
		}
		return nil
	}
	fgColor.SetValue(string(colors.White)[1:])

	return Model{
		name:    name,
		icon:    icon,
		bgColor: bgColor,
		fgColor: fgColor,
		lastBg:  lipgloss.Color("#" + bgColor.Value()),
		lastFg:  lipgloss.Color("#" + fgColor.Value()),
		create:  blurredCreate(),

		nameWidth: nameWidth,
		precomputedStyle: lipgloss.NewStyle().Width(nameWidth / 3).
			Background(colors.Background).Foreground(colors.White).MarginBackground(colors.Background),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	name := m.name.View()

	width := lipgloss.Width(name)

	colorStyle := lipgloss.NewStyle().Background(colors.Background).SetString("■\n ")

	iconPreview := ui.IconStyle(m.icon.Value(), m.lastFg, m.lastBg, colors.Background).String()
	iconPreview = lipgloss.NewStyle().
		Width(m.nameWidth).
		Align(lipgloss.Center).
		Background(colors.Background).
		Render(iconPreview)

	color := colors.Gray
	if m.icon.Err != nil {
		color = colors.Error
	} else if m.selected == IconField {
		color = colors.Focus
	}
	ic := lipgloss.NewStyle().Width(MaxIconLength + 1).Background(colors.Background).Render(m.icon.View())
	iconInput := underlineStyle(ic, MaxIconLength, color)
	iconInput = lipgloss.NewStyle().Background(colors.Background).Render(iconInput)
	iconText := m.precomputedStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, iconHeader(), iconInput))

	color = colors.Gray
	if m.bgColor.Err != nil {
		color = colors.Error
	} else if m.selected == BgColorField {
		color = colors.Focus
	}
	bg := lipgloss.NewStyle().Width(MaxHexDigits + 1).Background(colors.Background).Render(m.bgColor.View())
	bgColorInput := underlineStyle(bg, MaxHexDigits, color)
	bgColorInput = lipgloss.NewStyle().Render(bgColorInput)
	bgColorIndicator := colorStyle.Foreground(m.lastBg).String()
	bgColorText := lipgloss.JoinHorizontal(lipgloss.Top, bgColorHeader(), bgColorInput, bgColorIndicator)
	bgColorText = m.precomputedStyle.Render(bgColorText)

	color = colors.Gray
	if m.fgColor.Err != nil {
		color = colors.Error
	} else if m.selected == FgColorField {
		color = colors.Focus
	}
	fg := lipgloss.NewStyle().Width(7).Background(colors.Background).Render(m.fgColor.View())
	fgColorInput := underlineStyle(fg, MaxHexDigits, color)
	fgColorInput = lipgloss.NewStyle().Width(MaxHexDigits + 1).Render(fgColorInput)
	fgColorIndicator := colorStyle.Foreground(m.lastFg).String()
	fgColorText := lipgloss.JoinHorizontal(lipgloss.Top, fgColorHeader(), fgColorInput, fgColorIndicator)
	fgColorText = m.precomputedStyle.Render(fgColorText)

	icon := lipgloss.JoinHorizontal(lipgloss.Top, fgColorText, bgColorText, iconText)
	icon = lipgloss.NewStyle().Width(width).Background(colors.Background).Render(icon)

	privateStyle := lipgloss.NewStyle().
		Width(width).
		PaddingLeft(1).
		Background(colors.Background).
		Foreground(colors.White)

	if m.selected == PrivateField {
		privateStyle = privateStyle.Foreground(colors.Focus)
	}
	private := "[ ] Private"
	if m.private {
		private = "[x] Private"
	}
	private = privateStyle.Render(private)

	create := lipgloss.NewStyle().
		Width(m.nameWidth).
		Align(lipgloss.Center).
		Background(colors.Background).
		Render(m.create)

	content := flex.NewVertical(iconPreview, name, icon, private, create).WithGap(1).View()

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
			case IconField:
				m.icon, cmd = m.icon.Update(msg)
			case BgColorField:
				oldValue := m.bgColor.Value()
				position := m.bgColor.Position()
				m.bgColor, cmd = m.bgColor.Update(msg)
				newValue := m.bgColor.Value()

				hex := "0123456789abcdefABCDEF"
				invalid := false
				for _, c := range newValue {
					if !strings.ContainsRune(hex, c) {
						invalid = true
						break
					}
				}

				if invalid {
					m.bgColor.SetValue(oldValue)
					m.bgColor.SetCursor(position)
				} else if len(m.bgColor.Value()) == 6 {
					m.lastBg = lipgloss.Color("#" + m.bgColor.Value())
				}
			case FgColorField:
				oldValue := m.fgColor.Value()
				position := m.fgColor.Position()
				m.fgColor, cmd = m.fgColor.Update(msg)
				newValue := m.fgColor.Value()

				hex := "0123456789abcdefABCDEF"
				invalid := false
				for _, c := range newValue {
					if !strings.ContainsRune(hex, c) {
						invalid = true
						break
					}
				}

				if invalid {
					m.fgColor.SetValue(oldValue)
					m.fgColor.SetCursor(position)
				} else if len(m.fgColor.Value()) == 6 {
					m.lastFg = lipgloss.Color("#" + m.fgColor.Value())
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
	m.icon.Blur()
	m.bgColor.Blur()
	m.fgColor.Blur()
	m.create = blurredCreate()
	switch m.selected {
	case NameField:
		return m.name.Focus()
	case IconField:
		return m.icon.Focus()
	case BgColorField:
		return m.bgColor.Focus()
	case FgColorField:
		return m.fgColor.Focus()
	case PrivateField:
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
	if m.selected == PrivateField {
		m.private = !m.private
		return nil
	}

	if m.selected != CreateField {
		return nil
	}

	m.name.Input.Err = m.name.Input.Validate(m.name.Input.Value())
	m.icon.Err = m.icon.Validate(m.icon.Value())
	m.bgColor.Err = m.bgColor.Validate(m.bgColor.Value())
	m.fgColor.Err = m.fgColor.Validate(m.fgColor.Value())
	if m.name.Input.Err != nil || m.icon.Err != nil || m.bgColor.Err != nil || m.fgColor.Err != nil {
		return nil
	}

	request := packet.CreateNetwork{
		Name:       m.name.Input.Value(),
		Icon:       m.icon.Value(),
		BgHexColor: "#" + m.bgColor.Value(),
		FgHexColor: "#" + m.fgColor.Value(),
		IsPublic:   !m.private,
	}
	return gateway.Send(&request)
}

package networkcreation

import (
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/field"
	"github.com/kyren223/eko/internal/client/ui/layouts/flex"
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

	underlineStyle = func(s string, width int, color lipgloss.Color) string {
		underline := lipgloss.NewStyle().Foreground(color).
			Render(strings.Repeat(lipgloss.ThickBorder().Bottom, width))
		return lipgloss.JoinVertical(lipgloss.Left, s, underline)
	}

	iconHeader  = headerStyle.Bold(true).Render(" Icon: ")
	colorHeader = headerStyle.Bold(true).Italic(true).Render("# ")

	blurredCreate = lipgloss.NewStyle().
			Background(colors.Gray).Padding(0, 1).Render("Create Network")
	focusedCreate = lipgloss.NewStyle().
			Background(colors.Blue).Padding(0, 1).Render("Create Network")
)

const (
	MaxIconLength = 2
	MaxHexDigits  = 6
)

const (
	NameField = iota
	IconField
	ColorField
	CreateField
	FieldCount
)

type Model struct {
	precomputedIconStyle lipgloss.Style

	name   field.Model
	icon   textinput.Model
	color  textinput.Model
	create string

	selected  int
	nameWidth int
}

func New() Model {
	name := field.New(width)
	name.Header = "Network Name"
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
		if strings.TrimSpace(s) != s {
			return errors.New("no leading/trailing spaces")
		}
		return nil
	}

	nameWidth := lipgloss.Width(name.View())

	icon := textinput.New()
	icon.Prompt = ""
	icon.CharLimit = MaxIconLength
	icon.Placeholder = "ic"
	icon.Validate = func(s string) error {
		if len(s) == 0 {
			return errors.New("err")
		}
		return nil
	}

	color := textinput.New()
	color.Prompt = ""
	color.CharLimit = MaxHexDigits
	color.Placeholder = "000000"
	color.Validate = func(s string) error {
		if len(s) != 0 && len(s) != MaxHexDigits {
			return errors.New("err")
		}
		return nil
	}

	return Model{
		name:   name,
		icon:   icon,
		color:  color,
		create: blurredCreate,

		nameWidth:            nameWidth,
		precomputedIconStyle: lipgloss.NewStyle().Width(nameWidth / 2),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	name := m.name.View()

	color := colors.Gray
	if m.icon.Err != nil {
		color = colors.Error
	} else if m.selected == IconField {
		color = colors.Focus
	}
	iconInput := underlineStyle(m.icon.View(), MaxIconLength, color)
	iconText := m.precomputedIconStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, iconHeader, iconInput))

	color = colors.Gray
	if m.color.Err != nil {
		color = colors.Error
	} else if m.selected == ColorField {
		color = colors.Focus
	}
	colorInput := underlineStyle(m.color.View(), MaxHexDigits, color)
	indicatorColor := colors.Background
	if len(m.color.Value()) == 6 {
		indicatorColor = lipgloss.Color("#" + m.color.Value())
	}
	colorIndicator := lipgloss.NewStyle().Foreground(indicatorColor).Render(" ■")
	colorText := lipgloss.JoinHorizontal(lipgloss.Top, colorHeader, colorInput, colorIndicator)

	icon := lipgloss.JoinHorizontal(lipgloss.Top, iconText, colorText)

	// create := lipgloss.NewStyle().Width(m.nameWidth).Align(lipgloss.Center).Render(m.create)
	create := m.create

	content := flex.NewVertical(name, icon, create).WithGap(1).View()
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
			case IconField:
				m.icon, cmd = m.icon.Update(msg)
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
	m.color.Blur()
	m.create = blurredCreate
	switch m.selected {
	case NameField:
		return m.name.Focus()
	case IconField:
		return m.icon.Focus()
	case ColorField:
		return m.color.Focus()
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
	m.icon.Err = m.icon.Validate(m.icon.Value())
	m.color.Err = m.color.Validate(m.color.Value())
	if m.name.Input.Err != nil || m.icon.Err != nil || m.color.Err != nil {
		return nil
	}

	// TODO: send api request for creating the server
	return nil
}

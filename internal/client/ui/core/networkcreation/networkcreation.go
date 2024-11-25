package networkcreation

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/field"
	"github.com/kyren223/eko/pkg/assert"
)

var (
	style       = lipgloss.NewStyle().Border(lipgloss.ThickBorder())
	headerStyle = lipgloss.NewStyle().Foreground(colors.Turquoise)

	fieldBlurredStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colors.DarkCyan)
	fieldFocusedStyle = fieldBlurredStyle.
				BorderForeground(colors.Focus).
				Border(lipgloss.ThickBorder())

	underlineStyle = func(s string, width int, focus bool) string {
		color := colors.Gray
		if focus {
			color = colors.Focus
		}
		underline := lipgloss.NewStyle().Foreground(color).
			Render(strings.Repeat(lipgloss.ThickBorder().Bottom, width))
		return lipgloss.JoinVertical(lipgloss.Left, s, underline)
	}

	iconHeader  = headerStyle.Bold(true).Render(" Icon: ")
	colorHeader = headerStyle.Bold(true).Italic(true).Render("# ")
)

const (
	MaxIconLength = 2
	MaxHexDigits  = 6
)

const (
	NameField = iota
	IconField
	ColorField
	FieldCount
)

type Model struct {
	name                 field.Model
	Style                lipgloss.Style
	precomputedIconStyle lipgloss.Style

	icon                 textinput.Model
	color                textinput.Model
	Width                int
	Height               int
	selected             int
}

func New() Model {
	name := field.New(30)
	name.Header = "Network Name"
	name.HeaderStyle = headerStyle
	name.FocusedStyle = fieldFocusedStyle
	name.BlurredStyle = fieldBlurredStyle
	name.Input.CharLimit = 32
	name.Focus()

	icon := textinput.New()
	icon.Prompt = ""
	icon.CharLimit = MaxIconLength
	icon.Placeholder = "ic"

	color := textinput.New()
	color.Prompt = ""
	color.CharLimit = MaxHexDigits
	color.Placeholder = "000000"

	return Model{
		Width:  50,
		Height: 10,
		name:   name,
		icon:   icon,
		color:  color,
		Style:  style,

		precomputedIconStyle: lipgloss.NewStyle().Width(lipgloss.Width(name.View()) / 2),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	name := m.name.View()

	iconInput := underlineStyle(m.icon.View(), MaxIconLength, m.selected == IconField)
	iconText := m.precomputedIconStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, iconHeader, iconInput))

	colorInput := underlineStyle(m.color.View(), MaxHexDigits, m.selected == ColorField)
	colorText := lipgloss.JoinHorizontal(lipgloss.Top, colorHeader, colorInput)

	icon := lipgloss.JoinHorizontal(lipgloss.Top, iconText, colorText)

	content := lipgloss.JoinVertical(lipgloss.Left, name, "\n", icon)
	popup := lipgloss.NewStyle().Width(m.Width).Height(m.Height).Render(content)
	return m.Style.Render(popup)
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
				m.color, cmd = m.color.Update(msg)
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
	switch m.selected {
	case NameField:
		return m.name.Focus()
	case IconField:
		return m.icon.Focus()
	case ColorField:
		return m.color.Focus()
	default:
		assert.Never("missing switch statement field in update focus", "selected", m.selected)
		return nil
	}
}

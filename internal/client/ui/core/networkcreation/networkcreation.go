package networkcreation

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/field"
)

var (
	style        = lipgloss.NewStyle().Border(lipgloss.ThickBorder())
	focusedStyle = lipgloss.NewStyle().Foreground(colors.Focus)
	headerStyle  = lipgloss.NewStyle().Foreground(colors.Turquoise)

	fieldBlurredStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colors.DarkCyan)
	fieldFocusedStyle = fieldBlurredStyle.
				BorderForeground(colors.Focus).
				Border(lipgloss.ThickBorder())

	blurredUnderlineStyle = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder(), false, false, true, false).
				BorderForeground(colors.Gray)
	focusedUnderlineStyle = blurredUnderlineStyle.BorderForeground(colors.Focus)
)

const (
	NameField = iota
	IconField
	ColorField
	FieldCount
)

type Model struct {
	Width  int
	Height int

	name  field.Model
	icon  textinput.Model
	color textinput.Model

	selected int

	Style lipgloss.Style
}

func New() Model {
	name := field.New(30)
	name.Header = "Network Name"
	name.HeaderStyle = headerStyle
	name.FocusedStyle = fieldFocusedStyle
	name.BlurredStyle = fieldBlurredStyle
	name.Focus()

	icon := textinput.New()
	icon.Prompt = ""
	icon.Placeholder = "ic"

	color := textinput.New()
	color.Prompt = ""
	color.Placeholder = "000000"

	return Model{
		Width:  50,
		Height: 10,
		name:   name,
		icon:   icon,
		color:  color,
		Style:  style,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	name := m.name.View()

	var iconInput string
	if m.selected == IconField {
		iconInput = focusedUnderlineStyle.Render(m.icon.View())
	} else {
		iconInput = blurredUnderlineStyle.Render(m.icon.View())
	}

	var colorInput string
	if m.selected == ColorField {
		colorInput = focusedUnderlineStyle.Render(m.color.View())
	} else {
		colorInput = blurredUnderlineStyle.Render(m.color.View())
	}

	iconText := lipgloss.JoinHorizontal(lipgloss.Top, " Icon: ", iconInput)
	iconColor := lipgloss.JoinHorizontal(lipgloss.Top, "# ", colorInput)
	icon := lipgloss.JoinHorizontal(lipgloss.Top, iconText, "     ", iconColor)

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
			m.Cycle(1)
		case tea.KeyShiftDown:
			m.Cycle(-1)

		}
	}

	if m.selected == NameField {
		return m, m.name.Focus()
	} else {
		m.name.Blur()
	}

	return m, nil
}

func (m *Model) Cycle(step int) {
	m.selected += step
	if m.selected < 0 {
		m.selected = 0
	} else {
		m.selected %= FieldCount
	}
}

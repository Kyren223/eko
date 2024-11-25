package networkcreation

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/ui/field"
)

var (
	style      = lipgloss.NewStyle().Border(lipgloss.ThickBorder())
	focusColor = lipgloss.Color("#5874FF")

	focusedStyle = lipgloss.NewStyle().Foreground(focusColor)

	fieldBlurredStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#007E8A"))
	fieldFocusedStyle = fieldBlurredStyle.BorderForeground(focusedStyle.GetForeground()).Border(lipgloss.ThickBorder())

	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#54D7A9"))

	blurredUnderlineStyle = lipgloss.NewStyle().Border(lipgloss.ThickBorder(), false, false, true, false).BorderForeground(lipgloss.Color("240"))
	focusedUnderlineStyle = blurredUnderlineStyle.BorderForeground(focusColor)
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

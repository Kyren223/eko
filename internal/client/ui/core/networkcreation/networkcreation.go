package networkcreation

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/ui/field"
)

var (
	style = lipgloss.NewStyle().Border(lipgloss.ThickBorder())

	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#5874FF"))

	fieldBlurredStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#007E8A"))
	fieldFocusedStyle = fieldBlurredStyle.BorderForeground(focusedStyle.GetForeground()).Border(lipgloss.ThickBorder())

	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#54D7A9"))

	iconStyle = lipgloss.NewStyle().Border(lipgloss.ThickBorder(), false, false, true, false)
)

type Model struct {
	Width  int
	Height int

	name  field.Model
	icon  textinput.Model
	color textinput.Model

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
		color: color,
		Style:  style,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	name := m.name.View()

	iconText := lipgloss.JoinHorizontal(lipgloss.Top, "icon: ", iconStyle.Render(m.icon.View()))
	iconColor := lipgloss.JoinHorizontal(lipgloss.Top, "# ", iconStyle.Render(m.color.View()))
	icon := lipgloss.JoinHorizontal(lipgloss.Top, iconText, "     ", iconColor)

	content := lipgloss.JoinVertical(lipgloss.Left, name, "\n", icon)
	popup := lipgloss.NewStyle().Width(m.Width).Height(m.Height).Render(content)
	return m.Style.Render(popup)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

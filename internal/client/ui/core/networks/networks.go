package networks

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/data"
)

var (
	style            = lipgloss.NewStyle().Border(lipgloss.ThickBorder(), true, false)
	partialIconStyle = lipgloss.NewStyle().Width(6).Height(3).PaddingTop(1).Margin(0, 1).
				Align(lipgloss.Center).
				Border(lipgloss.ThickBorder(), false, false)
	networks = []string{
		"󰜈 ",
		" ",
		"Kr",
		" ",
		// "Test",
		// "██████\n██████\n██████",
		// " ▄▄▄▄ \n █  █ \n ▀▀▀▀ ",
		// "      \n  󰜈  \n     ",
		// "Hmm",
		// "Another",
	}

	reverse = lipgloss.NewStyle().Foreground(lipgloss.Color("#20999D"))
	// Render("")

	label1 = lipgloss.NewStyle().
		Background(lipgloss.Color("#20999D")).Foreground(lipgloss.Color("#000000")).
		Padding(0, 1).
		Bold(true).
		Render("󰜈")
	label2 = reverse.Render("") + label1 + reverse.Render("")
	label3 = ""
	label4 = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#20999D")).
		Bold(true).
		Render("󰜈")

	name = lipgloss.NewStyle().
		Bold(true).
		Render("Kyren223")
	user = lipgloss.JoinHorizontal(lipgloss.Center, label4, " ", name)
)

func IconStyle(fg, bg lipgloss.Color) lipgloss.Style {
	return partialIconStyle.Foreground(fg).Background(bg)
}

type Model struct {
	networks []string
	a        data.Network
}

func New() Model {
	return Model{
		networks: networks,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	// border := lipgloss.ThickBorder()
	var builder strings.Builder

	// top := strings.Repeat(border.Top, 6)
	// builder.WriteString(fmt.Sprintf("%s%s%s\n", border.TopLeft, top, border.TopRight))
	builder.WriteString("\n")
	for _, network := range m.networks {
		builder.WriteString(IconStyle(lipgloss.Color("#ffbf00"), lipgloss.Color("#4a3d5c")).
			MarginBottom(1).Render(network))
		builder.WriteString("\n")
	}
	// bottom := strings.Repeat(border.Bottom, 6)
	// builder.WriteString(fmt.Sprintf("\n%s%s%s", border.BottomLeft, bottom, border.BottomRight))

	result := builder.String()

	sep := lipgloss.NewStyle().Border(lipgloss.ThickBorder(), false, true, false, false).Height(ui.Height).Width(0)
	result = lipgloss.JoinHorizontal(lipgloss.Top, result, sep.Render(""))

	return result + user
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

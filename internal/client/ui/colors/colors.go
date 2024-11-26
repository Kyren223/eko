package colors

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	Gray             = lipgloss.Color("#585858")
	LightBlue        = lipgloss.Color("#5874FF")
	Blue             = lipgloss.Color("#0029f5")
	DarkCyan         = lipgloss.Color("#007E8A")
	DarkMidnightBlue = lipgloss.Color("#1E1E2E")
	Turquoise        = lipgloss.Color("#54D7A9")
	Red              = lipgloss.Color("#F16265")
	White            = lipgloss.Color("#FFFFFF")
	Black            = lipgloss.Color("#000000")

	Background = DarkMidnightBlue
	Error      = Red
	Focus      = LightBlue
)

func ToHex(color lipgloss.Color) string {
	r, g, b, _ := color.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
}

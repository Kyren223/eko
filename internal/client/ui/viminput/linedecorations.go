package viminput

import (
	"strconv"

	"github.com/charmbracelet/lipgloss"
)

func LineNumberDecoration(style lipgloss.Style) LineDecoration {
	return func(lnum int, line string, cursorLnum int) string {
		return style.Render(strconv.FormatInt(int64(lnum), 10))
	}
}

func EmptyLineDecoration(lnum int, line string, cursorLnum int) string {
	return ""
}

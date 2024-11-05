package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ModelTransition struct {
	Model tea.Model
}

func AddBorderHeader(header string, headerOffset int, style lipgloss.Style, render string) string {
	b := style.GetBorderStyle()
	body := style.UnsetBorderTop().Render(render)

	bodyWidth, headerWidth := lipgloss.Width(body), lipgloss.Width(header)
	leftCornerWidth, rightCornerWidth := lipgloss.Width(b.TopLeft), lipgloss.Width(b.TopRight)
	topWidth := bodyWidth - leftCornerWidth - rightCornerWidth

	leftWidth := headerOffset
	rightWidth := topWidth - leftWidth - headerWidth

	topStyle := lipgloss.NewStyle().
		Background(style.GetBorderTopBackground()).
		Foreground(style.GetBorderTopForeground())

	left := b.TopLeft + strings.Repeat(b.Top, leftWidth)
	right := topStyle.Render(strings.Repeat(b.Top, rightWidth) + b.TopRight)

	borderTop := lipgloss.NewStyle().
		Inline(true).
		MaxWidth(bodyWidth).
		Background(style.GetBorderTopBackground()).
		Foreground(style.GetBorderTopForeground()).
		Render(fmt.Sprintf("%s%s%s", left, header, right))

	return lipgloss.JoinVertical(lipgloss.Left, borderTop, body)
}

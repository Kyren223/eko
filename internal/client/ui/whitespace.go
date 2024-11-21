package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// whitespace is a whitespace renderer.
type whitespace struct {
	re    *lipgloss.Renderer
	style termenv.Style
	chars string
}

// newWhitespace creates a new whitespace renderer. The order of the options
// matters, if you're using WithWhitespaceRenderer, make sure it comes first as
// other options might depend on it.
func newWhitespace(r *lipgloss.Renderer, opts ...WhitespaceOption) *whitespace {
	w := &whitespace{
		re:    r,
		style: r.ColorProfile().String(),
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Render whitespaces.
func (w whitespace) render(width int) string {
	if w.chars == "" {
		w.chars = " "
	}

	r := []rune(w.chars)
	j := 0
	b := strings.Builder{}

	// Cycle through runes and print them into the whitespace.
	for i := 0; i < width; {
		b.WriteRune(r[j])
		j++
		if j >= len(r) {
			j = 0
		}
		i += ansi.StringWidth(string(r[j]))
	}

	// Fill any extra gaps white spaces. This might be necessary if any runes
	// are more than one cell wide, which could leave a one-rune gap.
	short := width - ansi.StringWidth(b.String())
	if short > 0 {
		b.WriteString(strings.Repeat(" ", short))
	}

	return w.style.Styled(b.String())
}

// WhitespaceOption sets a styling rule for rendering whitespace.
type WhitespaceOption func(*whitespace)

// WithWhitespaceForeground sets the color of the characters in the whitespace.
func WithWhitespaceForeground(c lipgloss.TerminalColor) WhitespaceOption {
	return func(w *whitespace) {
		w.style = w.style.Foreground(ToTermenvColor(w.re, c))
	}
}

// WithWhitespaceBackground sets the background color of the whitespace.
func WithWhitespaceBackground(c lipgloss.TerminalColor) WhitespaceOption {
	return func(w *whitespace) {
		w.style = w.style.Background(ToTermenvColor(w.re, c))
	}
}

// WithWhitespaceChars sets the characters to be rendered in the whitespace.
func WithWhitespaceChars(s string) WhitespaceOption {
	return func(w *whitespace) {
		w.chars = s
	}
}

func ToTermenvColor(re *lipgloss.Renderer, c lipgloss.TerminalColor) termenv.Color {
	color := c.(lipgloss.Color)
	return re.ColorProfile().Color(string(color))
}

package ui

import (
	"bytes"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	charmansi "github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/ansi"
	"github.com/muesli/reflow/truncate"

	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/pkg/assert"
)

const (
	DEBUG = true

	MinWidth                   = 85
	Center   lipgloss.Position = 0.499
)

var (
	Width   int
	Height  int
	Program *tea.Program

	NormalMemberStyle        = lipgloss.NewStyle().Foreground(colors.Purple).SetString("󰀉")
	AdminMemberStyle         = lipgloss.NewStyle().Foreground(colors.Red).Bold(true).SetString("󰓏")
	OwnerMemberStyle         = AdminMemberStyle.Foreground(colors.Gold).SetString("󱟜")
	TrustedNormalMemberStyle = NormalMemberStyle.SetString("󰢏")
	TrustedAdminMemberStyle  = AdminMemberStyle.SetString("󱄻")
	TrustedOwnerMemberStyle  = OwnerMemberStyle.SetString("󱢼")
	UntrustedSymbol           = lipgloss.NewStyle().Foreground(colors.Red).Render("󱈸")
)

var NewAuth func() tea.Model

type ModelTransition struct {
	Model tea.Model
}

func Transition(model tea.Model) tea.Cmd {
	return func() tea.Msg {
		return ModelTransition{Model: model}
	}
}

type QuitMsg struct{}

func AddBorderHeader(header string, headerOffset int, style lipgloss.Style, render string) string {
	b := style.GetBorderStyle()
	body := style.UnsetBorderTop().Render(render)

	bodyWidth, headerWidth := lipgloss.Width(body), lipgloss.Width(header)
	leftCornerWidth, rightCornerWidth := lipgloss.Width(b.TopLeft), lipgloss.Width(b.TopRight)
	topWidth := bodyWidth - leftCornerWidth - rightCornerWidth

	leftWidth := headerOffset
	rightWidth := topWidth - leftWidth - headerWidth
	assert.Assert(leftWidth >= 0, "left width cannot be negative", "leftWidth", leftWidth)
	assert.Assert(rightWidth >= 0, "right width cannot be negative", "rightWidth", rightWidth)

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

// PlaceOverlay places fg on top of bg.
func PlaceOverlay(x, y int, fg, bg string, opts ...WhitespaceOption) string {
	fgLines, fgWidth := getLines(fg)
	bgLines, bgWidth := getLines(bg)
	bgHeight := len(bgLines)
	fgHeight := len(fgLines)

	if fgWidth >= bgWidth && fgHeight >= bgHeight {
		// FIXME: return fg or bg?
		return fg
	}
	// TODO: allow placement outside of the bg box?
	x = clamp(x, 0, bgWidth-fgWidth)
	y = clamp(y, 0, bgHeight-fgHeight)

	ws := &whitespace{}
	for _, opt := range opts {
		opt(ws)
	}

	var b strings.Builder
	for i, bgLine := range bgLines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i < y || i >= y+fgHeight {
			b.WriteString(bgLine)
			continue
		}

		pos := 0
		if x > 0 {
			left := truncate.String(bgLine, uint(x))
			pos = ansi.PrintableRuneWidth(left)
			b.WriteString(left)
			if pos < x {
				b.WriteString(ws.render(x - pos))
				pos = x
			}
		}

		fgLine := fgLines[i-y]
		b.WriteString(fgLine)
		pos += ansi.PrintableRuneWidth(fgLine)

		right := cutLeft(bgLine, pos)
		bgWidth := ansi.PrintableRuneWidth(bgLine)
		rightWidth := ansi.PrintableRuneWidth(right)
		if rightWidth <= bgWidth-pos {
			b.WriteString(ws.render(bgWidth - rightWidth - pos))
		}

		b.WriteString(right)
	}

	return b.String()
}

// cutLeft cuts printable characters from the left.
// This function is heavily based on muesli's ansi and truncate packages.
func cutLeft(s string, cutWidth int) string {
	var (
		pos    int
		isAnsi bool
		ab     bytes.Buffer
		b      bytes.Buffer
	)
	for _, c := range s {
		var w int
		if c == ansi.Marker || isAnsi {
			isAnsi = true
			ab.WriteRune(c)
			if ansi.IsTerminator(c) {
				isAnsi = false
				if bytes.HasSuffix(ab.Bytes(), []byte("[0m")) {
					ab.Reset()
				}
			}
		} else {
			w = runewidth.RuneWidth(c)
		}

		if pos >= cutWidth {
			if b.Len() == 0 {
				if ab.Len() > 0 {
					b.Write(ab.Bytes())
				}
				if pos-cutWidth > 1 {
					b.WriteByte(' ')
					continue
				}
			}
			b.WriteRune(c)
		}
		pos += w
	}
	return b.String()
}

func getLines(s string) (lines []string, widest int) {
	lines = strings.Split(s, "\n")

	for _, l := range lines {
		w := charmansi.StringWidth(l)
		if widest < w {
			widest = w
		}
	}

	return lines, widest
}

func clamp(v, lower, upper int) int {
	return min(max(v, lower), upper)
}

/*
🭊🭂██🭍🬿
██████
🭥🭓██🭞🭚

🭠🭘  🭣🭕

🭏🬽  🭈🭄
*/

func IconStyle(icon string, iconFg, iconBg, bg lipgloss.Color) lipgloss.Style {
	bgStyle := lipgloss.NewStyle().Background(iconBg).Foreground(bg)
	top := bgStyle.Render("🭠🭘  🭣🭕")
	middle := lipgloss.NewStyle().Width(6).Align(lipgloss.Center).
		Background(iconBg).Foreground(iconFg).Render(icon)
	bgStyle2 := lipgloss.NewStyle().Foreground(iconBg)
	bottom := bgStyle2.Render("🭥🭓██🭞🭚")
	combined := lipgloss.JoinVertical(lipgloss.Left, top, middle, bottom)
	return lipgloss.NewStyle().SetString(combined)
}

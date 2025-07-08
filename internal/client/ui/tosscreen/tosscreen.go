package tosscreen

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"

	// "github.com/charmbracelet/glamour"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/pkg/assert"
)

const MarginPercentage = 0.25

var style = func() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderBackground(colors.Background).
		Background(colors.Background).
		MarginBackground(colors.BackgroundDimmer).
		Padding(0, 3).
		Margin(1, int(float32(ui.Width)*MarginPercentage), 0)
}

type Model struct {
	content string
	vp      viewport.Model
}

func New(content string) Model {
	m := Model{
		content: content,
		vp:      viewport.New(ui.Width, ui.Height),
	}
	m.SetContent(content)

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	sepStyle := lipgloss.NewStyle().Background(colors.BackgroundDimmer).Foreground(colors.Purple)

	var b strings.Builder

	accept := lipgloss.NewStyle().
		Background(colors.BackgroundDimmer).
		Foreground(colors.Green).
		Render("󰳝 Accept")
	decline := lipgloss.NewStyle().
		Background(colors.BackgroundDimmer).
		Foreground(colors.Red).
		Render("^C Decline")

	b.WriteString(sepStyle.Render("-- "))
	b.WriteString(accept)
	b.WriteString(sepStyle.Render("  |  "))
	b.WriteString(decline)
	b.WriteString(sepStyle.Render("  |  ↑↓ or J/K Scroll  |  ^D/^U PgDn/PgUp --"))

	footer := b.String()

	fWidth := lipgloss.Width(footer)
	paddingLeft := (ui.Width - fWidth) / 2
	paddingRight := ui.Width - fWidth - paddingLeft
	footerStyle := lipgloss.NewStyle().
		Padding(0, paddingLeft, 0, paddingRight).
		Background(colors.BackgroundDimmer).
		MarginBackground(colors.BackgroundDimmer)
	footer = footerStyle.Render(footer)

	view := m.vp.View()

	return lipgloss.JoinVertical(lipgloss.Center, view, footer)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.vp.Width != ui.Width {
		m.vp.Width = ui.Width
		m.updateContent()
	}
	if m.vp.Height != ui.Height-1 {
		m.vp.Height = ui.Height - 1
		m.updateContent()
	}

	m.vp.Style = style()
	m.vp, cmd = m.vp.Update(msg)

	return m, cmd
}

func (m *Model) updateContent() {
	style := style()
	margin := style.GetMarginLeft() + style.GetMarginRight()
	padding := style.GetPaddingLeft() + style.GetPaddingRight()
	border := 2
	lineWidth := ui.Width - margin - padding - border

	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(MdStyle()),
		glamour.WithWordWrap(lineWidth),
	)
	content := m.content
	content = strings.ReplaceAll(content, "@", "") // HACK: workaround to remove mailto links
	content, err := r.Render(content)
	assert.NoError(err, "this should never error")
	content = strings.ReplaceAll(content, "", "@") // HACK: workaround to remove mailto links
	m.vp.SetContent(content)
}

func (m *Model) SetContent(content string) {
	m.content = content
	m.updateContent()
}

func MdStyle() ansi.StyleConfig {
	mdStyle := ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix:     "\n",
				BlockSuffix:     "\n",
				Color:           stringPtr(colors.White),
				BackgroundColor: stringPtr(colors.Background),
			},
			Margin: uintPtr(defaultMargin),
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{},
			Indent:         uintPtr(1),
			IndentToken:    stringPtr("│ "),
		},
		List: ansi.StyleList{
			LevelIndent: defaultListIndent,
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockSuffix: "\n",
				Color:       stringPtr(colors.LightBlue),
				Bold:        boolPtr(true),
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          " ",
				Suffix:          " ",
				Color:           stringPtr(colors.Black),
				BackgroundColor: stringPtr(colors.LightBlue),
				Bold:            boolPtr(true),
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "## ",
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "### ",
			},
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "#### ",
			},
		},
		H5: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "##### ",
			},
		},
		H6: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "###### ",
				Bold:   boolPtr(false),
			},
		},
		Strikethrough: ansi.StylePrimitive{
			CrossedOut: boolPtr(true),
		},
		Emph: ansi.StylePrimitive{
			Italic: boolPtr(true),
		},
		Strong: ansi.StylePrimitive{
			Color: stringPtr(colors.Orange),
			Bold:  boolPtr(true),
		},
		HorizontalRule: ansi.StylePrimitive{
			Color:  stringPtr(colors.Gray),
			Format: "\n--------\n",
		},
		Item: ansi.StylePrimitive{
			BlockPrefix: " ",
		},
		Enumeration: ansi.StylePrimitive{
			BlockPrefix: ". ",
		},
		Task: ansi.StyleTask{
			StylePrimitive: ansi.StylePrimitive{},
			Ticked:         "[✓] ",
			Unticked:       "[ ] ",
		},
		Link: ansi.StylePrimitive{},
		LinkText: ansi.StylePrimitive{
			Color:   stringPtr(colors.Gold),
			Bold:    boolPtr(true),
			Conceal: boolPtr(true),
			Faint:   boolPtr(true),
		},
		Image: ansi.StylePrimitive{
			Color:     stringPtr(colors.Purple),
			Underline: boolPtr(true),
		},
		ImageText: ansi.StylePrimitive{
			Color:  stringPtr(colors.LightGray),
			Format: "Image: {{.text}} →",
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          " ",
				Suffix:          " ",
				Color:           stringPtr(colors.Purple),
				BackgroundColor: stringPtr(colors.BackgroundHighlight),
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: stringPtr(colors.LightGray),
				},
				Margin: uintPtr(defaultMargin),
			},
			Chroma: &ansi.Chroma{
				Text: ansi.StylePrimitive{
					Color: stringPtr(colors.White),
				},
				Error: ansi.StylePrimitive{
					Color:           stringPtr(colors.White),
					BackgroundColor: stringPtr(colors.Red),
				},
				Comment: ansi.StylePrimitive{
					Color: stringPtr(colors.Gray),
				},
				CommentPreproc: ansi.StylePrimitive{
					Color: stringPtr(colors.Orange),
				},
				Keyword: ansi.StylePrimitive{
					Color: stringPtr(colors.Red),
				},
				KeywordReserved: ansi.StylePrimitive{
					Color: stringPtr(colors.Red),
				},
				KeywordNamespace: ansi.StylePrimitive{
					Color: stringPtr(colors.Red),
				},
				KeywordType: ansi.StylePrimitive{
					Color: stringPtr(colors.Turquoise),
				},
				Operator: ansi.StylePrimitive{
					Color: stringPtr(colors.Red),
				},
				Punctuation: ansi.StylePrimitive{
					Color: stringPtr(colors.White),
				},
				Name: ansi.StylePrimitive{
					Color: stringPtr(colors.Purple),
				},
				NameBuiltin: ansi.StylePrimitive{
					Color: stringPtr(colors.Red),
				},
				NameTag: ansi.StylePrimitive{
					Color: stringPtr(colors.Purple),
				},
				NameAttribute: ansi.StylePrimitive{
					Color: stringPtr(colors.LightBlue),
				},
				NameClass: ansi.StylePrimitive{
					Color:     stringPtr(colors.White),
					Underline: boolPtr(true),
					Bold:      boolPtr(true),
				},
				NameDecorator: ansi.StylePrimitive{
					Color: stringPtr(colors.Gold),
				},
				NameFunction: ansi.StylePrimitive{
					Color: stringPtr(colors.LightBlue),
				},
				LiteralNumber: ansi.StylePrimitive{
					Color: stringPtr(colors.Purple),
				},
				LiteralString: ansi.StylePrimitive{
					Color: stringPtr(colors.Orange),
				},
				LiteralStringEscape: ansi.StylePrimitive{
					Color: stringPtr(colors.Purple),
				},
				GenericDeleted: ansi.StylePrimitive{
					Color: stringPtr(colors.Red),
				},
				GenericEmph: ansi.StylePrimitive{
					Italic: boolPtr(true),
				},
				GenericInserted: ansi.StylePrimitive{
					Color: stringPtr(colors.Turquoise),
				},
				GenericStrong: ansi.StylePrimitive{
					Bold: boolPtr(true),
				},
				GenericSubheading: ansi.StylePrimitive{
					Color: stringPtr(colors.LightGray),
				},
				Background: ansi.StylePrimitive{
					BackgroundColor: stringPtr(colors.BackgroundDim),
				},
			},
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{},
			},
		},
		DefinitionDescription: ansi.StylePrimitive{
			BlockPrefix: "\n🠶 ",
		},
	}

	return mdStyle
}

const (
	defaultListIndent      = 2
	defaultListLevelIndent = 4
	defaultMargin          = 0
)

func stringPtr(c lipgloss.Color) *string { return (*string)(&c) }
func boolPtr(b bool) *bool               { return &b }
func uintPtr(u uint) *uint               { return &u }

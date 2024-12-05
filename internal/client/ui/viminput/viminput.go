package viminput

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/ui/colors"
)

var DefaultCursorStyle = lipgloss.NewStyle().Background(colors.White).Foreground(colors.Background)

type (
	LineDecoration = func(lnum int, m Model) string
)

const (
	NormalMode = iota
	InsertMode
	VisualMode
)

type Model struct {
	PlaceholderStyle lipgloss.Style
	PromptStyle      lipgloss.Style

	Placeholder    string
	LineDecoration LineDecoration

	lines        [][]rune
	cursorLine   int
	cursorColumn int
	goalColumn   int
	mode         int

	width  int
	height int
	focus  bool
}

func New(width, height int) Model {
	return Model{
		PlaceholderStyle: lipgloss.NewStyle(),
		PromptStyle:      lipgloss.NewStyle(),
		Placeholder:      "",
		LineDecoration:   EmptyLineDecoration,
		lines:            [][]rune{},
		cursorLine:       0,
		cursorColumn:     0,
		goalColumn:       -1,
		mode:             NormalMode,
		width:            width,
		height:           height,
		focus:            false,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	var lines [][]rune
	if len(m.lines) == 0 {
		placeholder := m.PlaceholderStyle.Render(m.Placeholder)
		lines = append(lines, []rune(placeholder))
	} else {
		lines = m.lines
	}

	var builder strings.Builder
	for i, line := range lines {
		lineDecoration := m.LineDecoration(i, m)
		builder.WriteString(lineDecoration)

		if m.CursorLine() != i {
			builder.WriteString(string(line))
		} else if m.CursorColumn() == len(m.lines[m.CursorLine()]) {
			builder.WriteString(string(line))
			builder.WriteString(DefaultCursorStyle.Render(" "))
		} else {
			builder.WriteString(string(line[:m.CursorColumn()]))
			builder.WriteString(DefaultCursorStyle.Render(string(line[m.CursorColumn()])))
			builder.WriteString(string(line[m.CursorColumn()+1:]))
		}

		builder.WriteByte('\n')
	}

	result := builder.String()
	result = lipgloss.NewStyle().Width(m.width).Height(m.height).Render(result)

	return result
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focus {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		cmd := m.handleKeys(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) SetWidth(width int) {
	m.width = width
}

func (m Model) Width() int {
	return m.width
}

func (m *Model) SetHeight(height int) {
	m.height = height
}

func (m Model) Height() int {
	return m.height
}

func (m *Model) Focus() {
	m.focus = true
}

func (m *Model) Blur() {
	m.focus = false
}

func (m *Model) SetLines(lines ...[]rune) {
	m.lines = lines
}

func (m *Model) SetLine(lnum int, line []rune) {
	m.lines[lnum] = line
}

func (m *Model) Line(lnum int) []rune {
	return m.lines[lnum]
}

func (m *Model) SetCursorColumn(col int) {
	m.cursorColumn = col
	m.goalColumn = -1
}

func (m *Model) SetCursorLine(line int) {
	fromLength := m.CursorColumn()
	toLength := len(m.lines[line])
	if fromLength > toLength && m.goalColumn == -1 {
		m.goalColumn = fromLength
	}
	if m.goalColumn != -1 {
		m.cursorColumn = min(toLength-1, m.goalColumn)
	}
	m.cursorLine = line
}

func (m *Model) CursorColumn() int {
	return m.cursorColumn
}

func (m *Model) CursorLine() int {
	return m.cursorLine
}

func (m *Model) handleKeys(key tea.KeyMsg) tea.Cmd {
	switch m.mode {
	case NormalMode:
		return m.handleNormalModeKeys(key)
	case InsertMode:
		return m.handleInsertModeKeys(key)
	}

	return nil
}

func (m *Model) handleNormalModeKeys(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "h":
		m.SetCursorColumn(max(m.CursorColumn()-1, 0))
	case "j":
		m.SetCursorLine(min(m.CursorLine()+1, len(m.lines)-1))
	case "k":
		m.SetCursorLine(max(m.CursorLine()-1, 0))
	case "l":
		m.SetCursorColumn(min(m.CursorColumn()+1, len(m.lines[m.CursorLine()])-1))
	case "i":
		m.mode = InsertMode
	case "a":
		m.mode = InsertMode
		m.SetCursorColumn(m.CursorColumn() + 1)
	case "0":
		m.SetCursorColumn(0)
	}

	return nil
}

func (m *Model) handleInsertModeKeys(key tea.KeyMsg) tea.Cmd {
	if key.Type == tea.KeyEscape {
		m.SetCursorColumn(max(m.CursorColumn()-1, 0))
		m.mode = NormalMode
		return nil
	}

	keyStr := key.String()
	length := len(keyStr)
	if length == 1 && 32 <= keyStr[0] && keyStr[0] <= 126 {
		line := m.lines[m.CursorLine()]
		m.lines[m.CursorLine()] = slices.Insert(line, m.CursorColumn(), rune(keyStr[0]))
		m.SetCursorColumn(m.CursorColumn() + 1)
	}

	return nil
}

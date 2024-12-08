package viminput

import (
	"slices"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/pkg/assert"
)

var DefaultCursorStyle = lipgloss.NewStyle().Background(colors.White).Foreground(colors.Background)

type (
	LineDecoration = func(lnum int, m Model) string
)

const Unchanged = -1

const (
	NormalMode = iota
	InsertMode
	VisualMode
	VisualLineMode
	OpendingMode
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
	pending      byte

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

		if m.cursorLine != i {
			builder.WriteString(string(line))
		} else if m.cursorColumn == len(m.lines[m.cursorLine]) {
			builder.WriteString(string(line))
			builder.WriteString(DefaultCursorStyle.Render(" "))
		} else {
			builder.WriteString(string(line[:m.cursorColumn]))
			builder.WriteString(DefaultCursorStyle.Render(string(line[m.cursorColumn])))
			builder.WriteString(string(line[m.cursorColumn+1:]))
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

func (m *Model) Lines() [][]rune {
	return m.lines
}

func (m *Model) SetCursorColumn(col int) {
	if len(m.lines[m.cursorLine]) == 0 {
		m.cursorColumn = 0
	} else {
		m.cursorColumn = max(col, 0)
	}
	m.goalColumn = -1
}

func (m *Model) SetCursorLine(line int) {
	m.cursorLine = min(max(line, 0), len(m.lines)-1)
	fromLength := m.cursorColumn
	toLength := len(m.lines[m.cursorLine])
	if fromLength > toLength && m.goalColumn == -1 {
		m.goalColumn = fromLength
	}
	if m.goalColumn != -1 {
		m.cursorColumn = max(min(toLength-1, m.goalColumn), 0)
	}
}

// func (m *Model) cursorColumn int {
// 	return m.cursorColumn
// }
//
// func (m *Model) cursorLine int {
// 	return m.cursorLine
// }

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
	line, col := m.Motion(key.String())
	if line != -1 {
		m.SetCursorLine(line)
	}
	if col != -1 {
		m.SetCursorColumn(col)
	}

	switch key.String() {
	case "d":
		fallthrough
	case "y":
		fallthrough
	case "c":
		fallthrough
	case "v":
		m.mode = OpendingMode
		m.pending = key.String()[0]

	case "i":
		m.mode = InsertMode
	case "a":
		m.mode = InsertMode
		m.SetCursorColumn(m.cursorColumn + 1)
	case "I":
		_, col := m.Motion("_")
		assert.Assert(col != -1, "Motion should exist", "motion", "_")
		m.SetCursorColumn(col)
		m.mode = InsertMode
	case "A":
		m.mode = InsertMode
		m.SetCursorColumn(len(m.lines[m.cursorLine]))

	case "x":
		line := m.lines[m.cursorLine]
		if len(line) != 0 {
			end := len(line) - 1
			copy(line[m.cursorColumn:], line[m.cursorColumn+1:])
			m.lines[m.cursorLine] = line[:end]
			if m.cursorColumn == end {
				m.SetCursorColumn(m.cursorColumn - 1)
			}
		}
	case "D":
		line := m.lines[m.cursorLine]
		if len(line) != 0 {
			m.lines[m.cursorLine] = line[:m.cursorColumn]
			m.SetCursorColumn(m.cursorColumn - 1)
		}
	case "o":
		m.lines = slices.Insert(m.lines, m.cursorLine+1, []rune(""))
		m.SetCursorLine(m.cursorLine + 1)
		m.SetCursorColumn(0)
		m.mode = InsertMode
	case "O":
		m.lines = slices.Insert(m.lines, m.cursorLine, []rune(""))
		m.SetCursorColumn(0)
		m.mode = InsertMode
	}
	return nil
}

func (m *Model) handleInsertModeKeys(key tea.KeyMsg) tea.Cmd {
	if key.Type == tea.KeyEscape {
		m.SetCursorColumn(m.cursorColumn - 1)
		m.mode = NormalMode
		return nil
	}

	if key.Type == tea.KeyBackspace {
		line := m.lines[m.cursorLine]
		if m.cursorColumn == 0 && m.cursorLine != 0 {
			lineBefore := m.lines[m.cursorLine-1]
			cursorColumn := len(lineBefore)

			m.lines = slices.Delete(m.lines, m.cursorLine, m.cursorLine+1)
			m.lines[m.cursorLine-1] = append(lineBefore, line...)

			m.SetCursorLine(m.cursorLine - 1)
			m.SetCursorColumn(cursorColumn)
		} else if len(line) != 0 {
			m.lines[m.cursorLine] = slices.Delete(line, m.cursorColumn-1, m.cursorColumn)
			m.SetCursorColumn(m.cursorColumn - 1)
		}
		return nil
	}

	if key.Type == tea.KeyEnter {
		line := m.lines[m.cursorLine]
		after := line[m.cursorColumn:]
		m.lines[m.cursorLine] = line[:m.cursorColumn]

		var newline []rune
		newline = append(newline, after...)
		m.lines = slices.Insert(m.lines, m.cursorLine+1, newline)

		m.SetCursorLine(m.cursorLine + 1)
		m.SetCursorColumn(0)

		return nil
	}

	keyStr := key.String()
	length := len(keyStr)
	if length == 1 && 32 <= keyStr[0] && keyStr[0] <= 126 {
		line := m.lines[m.cursorLine]
		m.lines[m.cursorLine] = slices.Insert(line, m.cursorColumn, rune(keyStr[0]))
		m.SetCursorColumn(m.cursorColumn + 1)
	}

	return nil
}

func (m *Model) RuneAtCursor() rune {
	return m.lines[m.cursorLine][m.cursorColumn]
}

func (m Model) Motion(motion string) (line, col int) {
	switch motion {
	case "h":
		return Unchanged, m.cursorColumn - 1
	case "j":
		return m.cursorLine + 1, Unchanged
	case "k":
		return m.cursorLine - 1, Unchanged
	case "l":
		return Unchanged, min(m.cursorColumn+1, len(m.lines[m.cursorLine])-1)

	case "0":
		return Unchanged, 0
	case "$":
		return Unchanged, len(m.lines[m.cursorLine]) - 1
	case "_":
		for i, r := range m.lines[m.cursorLine] {
			if !unicode.IsSpace(r) {
				return Unchanged, i
			}
		}
		return Unchanged, len(m.lines[m.cursorLine]) - 1
	case "-":
		if m.cursorLine-1 < 0 {
			return Unchanged, Unchanged
		}
		line := m.lines[m.cursorLine-1]
		for i, r := range line {
			if !unicode.IsSpace(r) {
				return m.cursorLine - 1, i
			}
		}
		return m.cursorLine - 1, len(line) - 1
	case "+":
		if m.cursorLine+1 >= len(m.lines) {
			return Unchanged, Unchanged
		}
		line := m.lines[m.cursorLine+1]
		for i, r := range line {
			if !unicode.IsSpace(r) {
				return m.cursorLine + 1, i
			}
		}
		return m.cursorLine + 1, len(line) - 1
	// case "E":
	// for {
	// 	line := m.lines[m.cursorLine]
	// 	m.SetCursorColumn(m.cursorColumn + 1)
	// 	for m.cursorColumn == len(line) {
	// 		cursorLine := m.cursorLine + 1
	// 		m.SetCursorLine(cursorLine)
	// 		m.SetCursorColumn(0)
	// 		if cursorLine == len(m.lines) {
	// 			return nil
	// 		}
	// 	}
	// 	if !unicode.IsSpace(m.RuneAtCursor()) {
	// 		break
	// 	}
	// }
	// for !unicode.IsSpace(m.RuneAtCursor()) {
	// 	line := m.lines[m.cursorLine]
	// 	m.SetCursorColumn(m.cursorColumn + 1)
	// 	if m.cursorColumn == len(line) {
	// 		break
	// 	}
	// }
	// m.SetCursorColumn(m.cursorColumn - 1)
	default:
		return Unchanged, Unchanged
	}
}

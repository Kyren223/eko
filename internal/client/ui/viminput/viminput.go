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

const (
	Unchanged   = -1
	InvalidGoal = -1
	NullChar    = 0
)

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

	register     string
	lines        [][]rune
	cursorLine   int
	cursorColumn int
	goalColumn   int
	mode         int
	pending      byte
	gmod         bool
	fchar        byte
	fmod         byte
	tlast        bool

	focus  bool
	width  int
	height int
}

func New(width, height int) Model {
	return Model{
		PlaceholderStyle: lipgloss.NewStyle(),
		PromptStyle:      lipgloss.NewStyle(),
		Placeholder:      "",
		LineDecoration:   EmptyLineDecoration,
		register:         "\nHMM\nYO",
		lines:            [][]rune{[]rune("")},
		cursorLine:       0,
		cursorColumn:     0,
		goalColumn:       InvalidGoal,
		mode:             NormalMode,
		pending:          NullChar,
		gmod:             false,
		fchar:            NullChar,
		fmod:             NullChar,
		tlast:            false,
		focus:            false,
		width:            width,
		height:           height,
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
		m.handleKeys(msg)
		return m, nil
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
	m.goalColumn = InvalidGoal
}

func (m *Model) SetCursorLine(line int) {
	m.cursorLine = min(max(line, 0), len(m.lines)-1)
	fromLength := m.cursorColumn
	toLength := len(m.lines[m.cursorLine])
	// In visual it's fine to be after the char
	if m.mode == NormalMode {
		toLength-- // In normal it's not (unless length == 0)
	}
	if fromLength > toLength && m.goalColumn == InvalidGoal {
		m.goalColumn = fromLength
	}
	if m.goalColumn != InvalidGoal {
		m.cursorColumn = max(min(toLength, m.goalColumn), 0)
	}
}

func (m *Model) handleKeys(key tea.KeyMsg) {
	switch m.mode {
	case NormalMode:
		m.handleNormalModeKeys(key)
	case InsertMode:
		m.handleInsertModeKeys(key)
	case OpendingMode:
		m.handleOpendingModeKeys(key)
	}
}

func (m *Model) handleNormalModeKeys(key tea.KeyMsg) {
	line, col := m.Motion(key.String())
	if line != Unchanged || col != Unchanged {
		if line != Unchanged {
			m.SetCursorLine(line)
		}
		if col != Unchanged {
			length := len(m.lines[m.cursorLine]) - 1
			m.SetCursorColumn(min(col, length))
		}
		return
	}

	switch key.String() {
	case "d":
		fallthrough
	case "y":
		fallthrough
	case "c":
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
			m.Yank(string(line[m.cursorColumn : m.cursorColumn+1]))
			copy(line[m.cursorColumn:], line[m.cursorColumn+1:])
			m.lines[m.cursorLine] = line[:len(line)-1]
			if m.cursorColumn == len(line)-1 {
				m.SetCursorColumn(m.cursorColumn - 1)
			}
		}
	case "C":
		m.mode = InsertMode
		fallthrough
	case "D":
		line := m.lines[m.cursorLine]
		if len(line) != 0 {
			m.Yank(string(line[m.cursorColumn:]))
			m.lines[m.cursorLine] = line[:m.cursorColumn]
			m.SetCursorColumn(m.cursorColumn - 1)
		}
	case "Y":
		line := m.lines[m.cursorLine]
		m.Yank(string(line[m.cursorColumn:]))

	case "o":
		m.lines = slices.Insert(m.lines, m.cursorLine+1, []rune(""))
		m.SetCursorLine(m.cursorLine + 1)
		m.SetCursorColumn(0)
		m.mode = InsertMode
	case "O":
		m.lines = slices.Insert(m.lines, m.cursorLine, []rune(""))
		m.SetCursorColumn(0)
		m.mode = InsertMode

	case "p":
		paste := m.Paste()
		newline := paste[len(paste)-1] == '\n'
		if newline {
			paste = paste[:len(paste)-1]
		}
		lines := strings.Split(paste, "\n")

		line := m.lines[m.cursorLine]

		if len(lines) == 1 && !newline {
			pastedLine := []rune(lines[0])
			if m.cursorColumn+1 >= len(line) {
				m.lines[m.cursorLine] = append(line, pastedLine...)
			} else {
				m.lines[m.cursorLine] = slices.Insert(line, m.cursorColumn+1, pastedLine...)
			}
			if len(line) == 0 {
				m.SetCursorColumn(m.cursorColumn - 1)
			}
			m.SetCursorColumn(m.cursorColumn + len(pastedLine))
		} else if newline {
			var runeLines [][]rune
			for _, line := range lines {
				runeLines = append(runeLines, []rune(line))
			}

			m.lines = slices.Insert(m.lines, m.cursorLine+1, runeLines...)

			m.SetCursorLine(m.cursorLine + 1)
			_, col := m.Motion("_")
			m.SetCursorColumn(col)
		} else {
			var runeLines [][]rune
			for _, line := range lines {
				runeLines = append(runeLines, []rune(line))
			}

			splittingCol := min(m.cursorColumn+1, len(line))
			line := m.lines[m.cursorLine]
			after := line[splittingCol:]
			lastLine := len(runeLines) - 1
			runeLines[lastLine] = append(runeLines[lastLine], after...)

			m.lines[m.cursorLine] = append(line[:splittingCol], runeLines[0]...)
			m.lines = slices.Insert(m.lines, m.cursorLine+1, runeLines[1:]...)

			m.SetCursorColumn(min(splittingCol, len(m.lines[m.cursorLine])-1))
		}
	case "P":
		paste := m.Paste()
		newline := paste[len(paste)-1] == '\n'
		if newline {
			paste = paste[:len(paste)-1]
		}

		lines := strings.Split(paste, "\n")
		line := m.lines[m.cursorLine]
		if len(lines) == 1 && !newline {
			pastedLine := []rune(lines[0])
			m.lines[m.cursorLine] = slices.Insert(line, m.cursorColumn, pastedLine...)
			m.SetCursorColumn(m.cursorColumn + len(pastedLine) - 1)
		} else if newline {
			var runeLines [][]rune
			for _, line := range lines {
				runeLines = append(runeLines, []rune(line))
			}
			m.lines = slices.Insert(m.lines, m.cursorLine, runeLines...)
			m.SetCursorLine(m.cursorLine)
			_, col := m.Motion("_")
			m.SetCursorColumn(col)
		} else {
			// TODO:
		}
	}
}

func (m *Model) handleInsertModeKeys(key tea.KeyMsg) {
	if key.Type == tea.KeyEscape {
		m.SetCursorColumn(m.cursorColumn - 1)
		m.mode = NormalMode
		return
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
		return
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

		return
	}

	keyStr := key.String()
	length := len(keyStr)
	if length == 1 && 32 <= keyStr[0] && keyStr[0] <= 126 {
		line := m.lines[m.cursorLine]
		m.lines[m.cursorLine] = slices.Insert(line, m.cursorColumn, rune(keyStr[0]))
		m.SetCursorColumn(m.cursorColumn + 1)
	}
}

func (m *Model) RuneAtCursor() rune {
	return m.lines[m.cursorLine][m.cursorColumn]
}

func (m *Model) Motion(motion string) (line, col int) {
	if motion == "g" && !m.gmod {
		m.gmod = true
		return Unchanged, Unchanged
	} else {
		m.gmod = false
	}

	isF := motion == "f" || motion == "t" || motion == "F" || motion == "T"
	if isF && m.fmod == NullChar {
		m.fmod = motion[0]
		return Unchanged, Unchanged
	}
	if m.fmod != NullChar {
		defer func(m *Model) { m.fmod = NullChar }(m)

		if len(motion) != 1 {
			return Unchanged, Unchanged
		}
		m.fchar = motion[0]

		dir := 1
		if m.fmod == 'F' || m.fmod == 'T' {
			dir = -1
		}
		line := m.lines[m.cursorLine]
		i := m.cursorColumn + dir

		index, ok := SearchChar(line, i, dir, rune(m.fchar))
		if !ok {
			return Unchanged, Unchanged
		}

		if m.fmod == 't' || m.fmod == 'T' {
			index -= dir
			m.tlast = true
		} else {
			m.tlast = false
		}
		return Unchanged, index
	}

	switch motion {
	case "h":
		return Unchanged, m.cursorColumn - 1
	case "j":
		return m.cursorLine + 1, Unchanged
	case "k":
		return m.cursorLine - 1, Unchanged
	case "l":
		return Unchanged, m.cursorColumn + 1

	case "0":
		return Unchanged, 0
	case "$":
		return Unchanged, len(m.lines[m.cursorLine])
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

	case "g":
		return 0, Unchanged
	case "G":
		return len(m.lines) - 1, Unchanged

	case ",":
		dir := -1
		line := m.lines[m.cursorLine]
		i := m.cursorColumn + dir
		if m.tlast {
			i += dir
		}
		index, ok := SearchChar(line, i, dir, rune(m.fchar))
		if !ok {
			return Unchanged, Unchanged
		}
		if m.tlast {
			index -= dir
		}
		return Unchanged, index
	case ";":
		dir := 1
		line := m.lines[m.cursorLine]
		i := m.cursorColumn + dir
		if m.tlast {
			i += dir
		}
		index, ok := SearchChar(line, i, dir, rune(m.fchar))
		if !ok {
			return Unchanged, Unchanged
		}
		if m.tlast {
			index -= dir
		}
		return Unchanged, index

	case "E":
		lnum := m.cursorLine
		col := m.cursorColumn
		for {
			line := m.lines[lnum]
			isLastLine := lnum == len(m.lines)-1
			isColAtEnd := col == len(line)-1

			// Skip if empty
			if len(line) == 0 && !isLastLine {
				lnum++
				col = 0
				continue
			}

			// Search next whitespace
			i, ok := SearchCharFunc(line, col+1, 1, unicode.IsSpace)
			if !ok {
				if isLastLine || !isColAtEnd {
					return lnum, len(line) - 1
				}
				lnum++
				col = 0
				continue
			}

			// Found next word - Simple case
			if i-1 != col {
				return lnum, i - 1
			}

			// Search start of next word
			i, ok = SearchCharFunc(line, col+1, 1, func(c rune) bool {
				return !unicode.IsSpace(c)
			})
			if !ok {
				if isLastLine {
					return lnum, len(line) - 1
				}
				lnum++
				col = 0
				continue
			}

			// Next word exists, find it's end
			i, ok = SearchCharFunc(line, i, 1, unicode.IsSpace)
			if !ok {
				if isLastLine || !isColAtEnd {
					return lnum, len(line) - 1
				}
				// If at the end, continue to next line (if it exists)
				lnum++
				col = 0
				continue
			}
			return lnum, i - 1
		}
	case "B":
		lnum := m.cursorLine
		col := m.cursorColumn
		for {
			line := m.lines[lnum]
			isFirstLine := lnum == 0
			isColAtStart := col == 0

			// Skip if empty
			if len(line) == 0 && !isFirstLine {
				lnum--
				col = len(m.lines[lnum]) - 1
				continue
			}

			// Search previous whitespace
			i, ok := SearchCharFunc(line, col-1, -1, unicode.IsSpace)
			if !ok {
				if isFirstLine || !isColAtStart {
					return lnum, 0
				}
				lnum--
				col = len(m.lines[lnum]) - 1
				continue
			}

			// Found next word - Simple case
			if i+1 != col {
				return lnum, i + 1
			}

			// Search end of previous word
			i, ok = SearchCharFunc(line, col-1, -1, func(c rune) bool {
				return !unicode.IsSpace(c)
			})
			if !ok {
				if isFirstLine {
					return lnum, 0
				}
				lnum--
				col = len(m.lines[lnum]) - 1
				continue
			}

			// Previous word exists, find it's start
			i, ok = SearchCharFunc(line, i, -1, unicode.IsSpace)
			if !ok {
				if isFirstLine || !isColAtStart {
					return lnum, 0
				}
				// If at the start, continue to previous line (if it exists)
				lnum--
				col = len(m.lines[lnum]) - 1
				continue
			}
			return lnum, i + 1
		}

	default:
		return Unchanged, Unchanged
	}
}

func (m *Model) handleOpendingModeKeys(key tea.KeyMsg) {
	if key.Type == tea.KeyEscape {
		m.ResetOpending()
		return
	}

	ftmod := m.fmod == 'f' || m.fmod == 't'
	lnum, col := m.Motion(key.String())
	if m.gmod || m.fmod != NullChar {
		return
	}
	if lnum != Unchanged || col != Unchanged {
		if lnum == Unchanged {
			lnum = m.cursorLine
		}
		if col == Unchanged {
			col = m.cursorColumn
		}
		if ftmod {
			col++ // Adjust due to upper bound being exclusive
			// For F/T (backwards), being exclusive is the correct
		}

		if lnum == m.cursorLine {
			line := m.lines[m.cursorLine]
			length := len(line)
			lower := min(col, m.cursorColumn, length)
			upper := min(max(col, m.cursorColumn), length)
			value := line[lower:upper]

			m.Yank(string(value))

			if m.pending == 'd' || m.pending == 'c' {
				line = slices.Delete(line, lower, upper)
				m.lines[m.cursorLine] = line
			}

			m.SetCursorColumn(min(lower, len(line)-1))
			if ftmod && m.pending == 'c' {
				m.SetCursorColumn(lower) // Ignore bound
				// Only ok because of insert mode (same as NVIM)
			}

		} else {
			lower := min(lnum, m.cursorLine)
			upper := max(lnum, m.cursorLine) + 1
			if upper > len(m.lines) || lower < 0 {
				m.ResetOpending()
				return
			}

			var builder strings.Builder
			for i := lower; i < upper; i++ {
				builder.WriteString(string(m.lines[i]))
				builder.WriteRune('\n')
			}

			m.Yank(builder.String())
			m.SetCursorLine(lower)

			switch m.pending {
			case 'd':
				m.lines = slices.Delete(m.lines, lower, upper)
				if len(m.lines) == 0 {
					m.lines = append(m.lines, []rune(""))
				}
				m.SetCursorLine(min(m.cursorLine, len(m.lines)-1))

				end := len(m.lines[m.cursorLine]) - 1
				m.SetCursorColumn(min(m.cursorColumn, end))

			case 'y':
				end := len(m.lines[m.cursorLine]) - 1
				m.SetCursorColumn(min(m.cursorColumn, end))

			case 'c':
				m.lines = slices.Delete(m.lines, lower+1, upper)
				m.lines[lower] = []rune("")
				m.SetCursorColumn(0)
			}

		}

		m.ResetOpending()
		if m.pending == 'c' {
			m.mode = InsertMode
		}
		return
	}

	switch key.String() {
	case "c":
		m.Yank(string(m.lines[m.cursorLine]) + "\n")
		m.lines[m.cursorLine] = []rune("")
		m.SetCursorColumn(0)

		m.ResetOpending()
		if m.pending == 'c' {
			m.mode = InsertMode
		}

	case "d":
		m.Yank(string(m.lines[m.cursorLine]) + "\n")
		if len(m.lines) == 1 {
			m.lines[0] = []rune("")
			m.SetCursorColumn(0)
		} else {
			m.lines = slices.Delete(m.lines, m.cursorLine, m.cursorLine+1)
			m.SetCursorLine(m.cursorLine)
			m.SetCursorColumn(m.cursorColumn)
		}

		m.ResetOpending()
		if m.pending == 'c' {
			m.mode = InsertMode
		}

	case "y":
		m.Yank(string(m.lines[m.cursorLine]) + "\n")

		m.ResetOpending()
		if m.pending == 'c' {
			m.mode = InsertMode
		}

	default:
		m.ResetOpending()
	}
}

func (m *Model) ResetOpending() {
	m.mode = NormalMode
}

func (m *Model) Yank(s string) {
	m.register = s
}

func (m Model) Paste() string {
	return m.register
}

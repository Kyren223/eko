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

var (
	CursorStyle = lipgloss.NewStyle().Background(colors.White).Foreground(colors.Background)
	VisualStyle = lipgloss.NewStyle().Background(colors.DarkGray)
)

const (
	Unchanged   = -1
	InvalidGoal = -1
	NullChar    = 0
	NoCount
)

type State struct {
	lines        [][]rune
	cursorLine   int
	cursorColumn int
}

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

	Placeholder  string
	register     string
	lines        [][]rune
	undoStack    []State
	redoStack    []State
	cursorLine   int
	cursorColumn int
	goalColumn   int
	mode         int
	count        int
	vline        int
	vcol         int
	pending      byte
	gmod         bool
	fchar        byte
	fmod         byte
	tlast        bool
	imod         bool // only in O-pending
	amod         bool // only in O-pending

	focus     bool
	width     int
	height    int
	maxHeight int
	offset    int
}

func New(width, maxHeight int) Model {
	return Model{
		PlaceholderStyle: lipgloss.NewStyle(),
		PromptStyle:      lipgloss.NewStyle(),
		Placeholder:      "",
		register:         "",
		lines:            [][]rune{[]rune("")},
		undoStack:        []State{{[][]rune{[]rune("")}, 0, 0}},
		redoStack:        []State{},
		cursorLine:       0,
		cursorColumn:     0,
		goalColumn:       InvalidGoal,
		mode:             NormalMode,
		count:            NoCount,
		vline:            0,
		vcol:             0,
		pending:          NullChar,
		gmod:             false,
		fchar:            NullChar,
		fmod:             NullChar,
		tlast:            false,
		imod:             false,
		amod:             false,
		focus:            false,
		width:            width,
		height:           1,
		maxHeight:        maxHeight,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	lines := m.lines

	var builder strings.Builder
	switch m.mode {
	case NormalMode, InsertMode, OpendingMode:
		for i, line := range lines {
			if i < m.offset || i >= m.offset+m.height {
				continue
			}

			if len(m.lines) == 1 && len(line) == 0 && m.Placeholder != "" {
				cursorChar := m.PlaceholderStyle.Render(m.Placeholder[0:1])
				rest := m.PlaceholderStyle.Render(m.Placeholder[1:])
				builder.WriteString(CursorStyle.Render(cursorChar))
				builder.WriteString(rest)
				builder.WriteByte('\n')
				continue
			}

			if m.cursorLine != i {
				builder.WriteString(string(line))
			} else if m.cursorColumn == len(line) {
				builder.WriteString(string(line))
				builder.WriteString(CursorStyle.Render(" "))
			} else {
				builder.WriteString(string(line[:m.cursorColumn]))
				builder.WriteString(CursorStyle.Render(string(line[m.cursorColumn])))
				builder.WriteString(string(line[m.cursorColumn+1:]))
			}

			builder.WriteByte('\n')
		}

	case VisualMode:
		isAnchorBefore := m.cursorLine > m.vline
		for i, line := range lines {
			if i < m.offset || i >= m.offset+m.height {
				continue
			}

			if len(m.lines) == 1 && len(line) == 0 && m.Placeholder != "" {
				cursorChar := m.PlaceholderStyle.Render(m.Placeholder[0:1])
				rest := m.PlaceholderStyle.Render(m.Placeholder[1:])
				builder.WriteString(CursorStyle.Render(cursorChar))
				builder.WriteString(rest)
				builder.WriteByte('\n')
				continue
			}

			before := isAnchorBefore && m.vline < i && i < m.cursorLine
			after := !isAnchorBefore && m.cursorLine < i && i < m.vline
			isInBetween := before || after
			if isInBetween {
				// Entire line highlighted
				builder.WriteString(VisualStyle.Render(string(line)))
				if len(line) == 0 {
					builder.WriteString(VisualStyle.Render(" "))
				}
			} else if m.cursorLine != i && m.vline != i {
				// Normal line
				builder.WriteString(string(line))
			} else if m.cursorLine == i && m.vline == i {
				// Both anchor and cursor are on the same line
				if m.cursorColumn == m.vcol {
					builder.WriteString(string(line[:m.cursorColumn]))
					if len(line) != 0 {
						cursorChar := string(line[m.cursorColumn])
						builder.WriteString(CursorStyle.Render(cursorChar))
						builder.WriteString(string(line[m.cursorColumn+1:]))
					} else {
						builder.WriteString(CursorStyle.Render(" "))
					}
					builder.WriteByte('\n')
					continue
				}

				lower := min(m.cursorColumn, m.vcol)
				upper := max(m.cursorColumn, m.vcol)
				builder.WriteString(string(line[:lower]))
				if lower == m.cursorColumn && m.cursorColumn < len(line) {
					builder.WriteString(CursorStyle.Render(string(line[m.cursorColumn])))
					lower++
				}
				safeUpper := min(upper, len(line))
				builder.WriteString(VisualStyle.Render(string(line[lower:safeUpper])))
				if m.cursorColumn == safeUpper && m.cursorColumn < len(line) {
					builder.WriteString(CursorStyle.Render(string(line[safeUpper])))
				} else if m.vcol == safeUpper && m.vcol < len(line) {
					builder.WriteString(VisualStyle.Render(string(line[safeUpper])))
				}
				safeLower := min(safeUpper+1, len(line))
				builder.WriteString(string(line[safeLower:]))

				if m.cursorColumn == len(line) {
					builder.WriteString(CursorStyle.Render(" "))
				} else if m.vcol == len(line) {
					builder.WriteString(VisualStyle.Render(" "))
				}

			} else if m.cursorLine == i && isAnchorBefore {
				// cursor line, highlight before cursor col
				if m.cursorColumn == len(line) {
					builder.WriteString(VisualStyle.Render(string(line[:m.cursorColumn])))
					builder.WriteString(CursorStyle.Render(" "))
				} else {
					builder.WriteString(VisualStyle.Render(string(line[:m.cursorColumn])))
					builder.WriteString(CursorStyle.Render(string(line[m.cursorColumn])))
					builder.WriteString(string(line[m.cursorColumn+1:]))
				}
			} else if m.cursorLine == i {
				// cursor line, highlight after cursor col
				builder.WriteString(string(line[:m.cursorColumn]))
				if m.cursorColumn == len(line) {
					builder.WriteString(CursorStyle.Render(" "))
				} else {
					builder.WriteString(CursorStyle.Render(string(line[m.cursorColumn])))
					builder.WriteString(VisualStyle.Render(string(line[m.cursorColumn+1:])))
				}
			} else if m.vline == i && isAnchorBefore {
				// anchor line highlight after col
				builder.WriteString(string(line[:m.vcol]))
				builder.WriteString(VisualStyle.Render(string(line[m.vcol:])))
				if m.vcol == len(line) {
					builder.WriteString(VisualStyle.Render(" "))
				}
			} else if m.vline == i {
				// anchor line, highlight before col
				if m.vcol == len(line) {
					builder.WriteString(VisualStyle.Render(string(line[:m.vcol])))
					builder.WriteString(VisualStyle.Render(" "))
				} else {
					builder.WriteString(VisualStyle.Render(string(line[:m.vcol+1])))
					builder.WriteString(string(line[m.vcol+1:]))
				}
			}

			builder.WriteByte('\n')
		}

	case VisualLineMode:
		isAnchorBefore := m.cursorLine > m.vline
		for i, line := range lines {
			if i < m.offset || i >= m.offset+m.height {
				continue
			}

			if len(m.lines) == 1 && len(line) == 0 && m.Placeholder != "" {
				cursorChar := m.PlaceholderStyle.Render(m.Placeholder[0:1])
				rest := m.PlaceholderStyle.Render(m.Placeholder[1:])
				builder.WriteString(CursorStyle.Render(cursorChar))
				builder.WriteString(rest)
				builder.WriteByte('\n')
				continue
			}

			before := isAnchorBefore && m.vline <= i && i < m.cursorLine
			after := !isAnchorBefore && m.cursorLine < i && i <= m.vline
			isInBetween := before || after
			if isInBetween {
				builder.WriteString(VisualStyle.Render(string(line)))
				if len(line) == 0 {
					builder.WriteString(VisualStyle.Render(" "))
				}
			} else if m.cursorLine != i {
				builder.WriteString(string(line))
			} else if m.cursorColumn == len(m.lines[m.cursorLine]) {
				builder.WriteString(VisualStyle.Render(string(line)))
				builder.WriteString(CursorStyle.Render(" "))
			} else {
				builder.WriteString(VisualStyle.Render(string(line[:m.cursorColumn])))
				builder.WriteString(CursorStyle.Render(string(line[m.cursorColumn])))
				builder.WriteString(VisualStyle.Render(string(line[m.cursorColumn+1:])))
			}

			builder.WriteByte('\n')
		}

	default:
		assert.Never("cannot display mode", "mode", m.mode)
	}

	result := builder.String()
	result = result[:len(result)-1] // Remove last \n
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
	}

	m.height = min(m.maxHeight, len(m.lines))
	diff := m.cursorLine - m.offset
	if diff >= m.height {
		m.offset += 1 + diff - m.height
	}
	if diff < 0 {
		m.offset += diff
	}

	if m.offset+m.height > len(m.lines) {
		m.offset -= (m.offset + m.height) - len(m.lines)
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
		m.cursorColumn = min(max(col, 0), len(m.lines[m.cursorLine]))
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
		m.Save()
		m.handleNormalModeKeys(key)
		m.Save()
	case InsertMode:
		// Note: we don't want to save on insert mode
		// That'd mean it'd save on every keystroke which will
		// be very annoying
		m.handleInsertModeKeys(key)
	case OpendingMode:
		m.Save()
		m.handleOpendingModeKeys(key)
		m.Save()
	case VisualMode:
		m.Save()
		m.handleVisualModeKeys(key)
		m.Save()
	case VisualLineMode:
		m.Save()
		m.handleVisualLineModeKeys(key)
		m.Save()
	}
}

func (m *Model) handleNormalModeKeys(key tea.KeyMsg) {
	if key.Type == tea.KeyEscape {
		m.count = NoCount
		return
	}

	motion := key.String()

	count := 1
	if m.count != NoCount && (motion[0] < '0' || motion[0] > '9') {
		count = m.count
		m.count = NoCount
	}

	shouldReturn := false
	for i := 0; i < count; i++ {
		line, col := m.Motion(motion)
		if line != Unchanged {
			m.SetCursorLine(line)
			shouldReturn = true
		}
		if col != Unchanged {
			length := len(m.lines[m.cursorLine]) - 1
			m.SetCursorColumn(min(col, length))
			shouldReturn = true
		}
	}
	if shouldReturn {
		return
	}

	switch key.String() {
	case "u":
		m.Undo()
	case "ctrl+r", "U":
		m.Redo()

	case "v":
		m.mode = VisualMode
		m.vline = m.cursorLine
		m.vcol = m.cursorColumn
	case "V":
		m.mode = VisualLineMode
		m.vline = m.cursorLine
		m.vcol = m.cursorColumn

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
		m.SetCursorColumn(m.cursorColumn + 1)
		m.mode = InsertMode
	case "I":
		_, col := m.Motion("_")
		assert.Assert(col != -1, "Motion should exist", "motion", "_")
		m.SetCursorColumn(col)
		m.mode = InsertMode
	case "A":
		m.SetCursorColumn(len(m.lines[m.cursorLine]))
		m.mode = InsertMode

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

			if len(line) != 0 {
				m.lines = slices.Insert(m.lines, m.cursorLine+1, runeLines...)
				m.SetCursorLine(m.cursorLine + 1)
				_, col := m.Motion("_")
				m.SetCursorColumn(col)
			} else {
				m.lines[m.cursorLine] = runeLines[0]
				m.lines = slices.Insert(m.lines, m.cursorLine+1, runeLines[1:]...)
				_, col := m.Motion("_")
				m.SetCursorColumn(col)
			}

		} else {
			var runeLines [][]rune
			for _, line := range lines {
				runeLines = append(runeLines, []rune(line))
			}

			// Note: caps to line length, needed for len=0 col=0
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
			var runeLines [][]rune
			for _, line := range lines {
				runeLines = append(runeLines, []rune(line))
			}

			splittingCol := m.cursorColumn
			line := m.lines[m.cursorLine]
			after := line[splittingCol:]
			lastLine := len(runeLines) - 1
			runeLines[lastLine] = append(runeLines[lastLine], after...)

			m.lines[m.cursorLine] = append(line[:splittingCol], runeLines[0]...)
			m.lines = slices.Insert(m.lines, m.cursorLine+1, runeLines[1:]...)

			m.SetCursorColumn(min(splittingCol, len(m.lines[m.cursorLine])-1))
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
	} else if m.gmod {
		m.gmod = false
		motion = "g" + motion
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

	if m.count != NoCount && '0' <= motion[0] && motion[0] <= '9' {
		m.count *= 10
		m.count += int(motion[0] - '0')
		return Unchanged, Unchanged
	}

	if '1' <= motion[0] && motion[0] <= '9' {
		m.count = int(motion[0] - '0')
		return Unchanged, Unchanged
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

	case "ctrl+d":
		return m.cursorLine + m.height/2, Unchanged
	case "ctrl+u":
		return m.cursorLine - m.height/2, Unchanged

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

	case "gg":
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
			i, ok = SearchCharFunc(line, i, 1, func(c rune) bool {
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
	case "e":
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

			// Search next non-matching char
			if len(line) == 0 {
				return lnum, 0
			}
			if col >= len(line) {
				return lnum, len(line) - 1
			}
			isKeyword := IsKeyword(line[col])
			isWhitespace := unicode.IsSpace(line[col])
			i, ok := SearchCharFunc(line, col+1, 1, func(c rune) bool {
				return unicode.IsSpace(c) || isWhitespace || isKeyword != IsKeyword(c)
			})
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
			i, ok = SearchCharFunc(line, i, 1, func(c rune) bool {
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
			isKeyword = IsKeyword(line[i])
			isWhitespace = unicode.IsSpace(line[i])
			i, ok = SearchCharFunc(line, i, 1, func(c rune) bool {
				return unicode.IsSpace(c) || isWhitespace || isKeyword != IsKeyword(c)
			})
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
			i, ok = SearchCharFunc(line, i, -1, func(c rune) bool {
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
	case "b":
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

			// Search previous non-matching char
			if col < 0 || len(line) == 0 {
				return lnum, 0
			}
			isKeyword := IsKeyword(line[col])
			isWhitespace := unicode.IsSpace(line[col])
			i, ok := SearchCharFunc(line, col-1, -1, func(c rune) bool {
				return unicode.IsSpace(c) || isWhitespace || isKeyword != IsKeyword(c)
			})
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
			i, ok = SearchCharFunc(line, i, -1, func(c rune) bool {
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
			isKeyword = IsKeyword(line[i])
			isWhitespace = unicode.IsSpace(line[i])
			i, ok = SearchCharFunc(line, i, -1, func(c rune) bool {
				return unicode.IsSpace(c) || isWhitespace || isKeyword != IsKeyword(c)
			})
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
	case "W":
		lnum := m.cursorLine
		col := m.cursorColumn
		remember := false
		for {
			line := m.lines[lnum]
			isLastLine := lnum == len(m.lines)-1

			// Skip if empty
			if len(line) == 0 && !isLastLine {
				// What happens if this is the current line
				if lnum == m.cursorLine {
					lnum++
					col = 0
					remember = true
					continue
				}
				return lnum, 0
			}

			if remember {
				i, ok := SearchCharFunc(line, col, 1, func(c rune) bool {
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
				return lnum, i
			}

			// Search next whitespace
			i, ok := SearchCharFunc(line, col, 1, unicode.IsSpace)
			if !ok {
				if isLastLine {
					return lnum, len(line) - 1
				}
				lnum++
				col = 0
				remember = true
				continue
			}

			// Search start of next word
			i, ok = SearchCharFunc(line, i, 1, func(c rune) bool {
				return !unicode.IsSpace(c)
			})
			if !ok {
				if isLastLine {
					return lnum, len(line) - 1
				}
				lnum++
				col = 0
				remember = true
				continue
			}
			return lnum, i
		}
	case "w":
		lnum := m.cursorLine
		col := m.cursorColumn
		remember := false
		for {
			line := m.lines[lnum]
			isLastLine := lnum == len(m.lines)-1

			// Skip if empty
			if len(line) == 0 && !isLastLine {
				// What happens if this is the current line
				if lnum == m.cursorLine {
					lnum++
					col = 0
					remember = true
					continue
				}
				return lnum, 0
			}

			if remember {
				i, ok := SearchCharFunc(line, col, 1, func(c rune) bool {
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
				return lnum, i
			}

			// Search previous non-matching char
			if col < 0 || len(line) == 0 {
				return lnum, 0
			}
			isKeyword := IsKeyword(line[col])
			isWhitespace := unicode.IsSpace(line[col])
			i, ok := SearchCharFunc(line, col, 1, func(c rune) bool {
				return unicode.IsSpace(c) || isWhitespace || isKeyword != IsKeyword(c)
			})
			if !ok {
				if remember {
					return lnum, 0
				}
				if isLastLine {
					return lnum, len(line) - 1
				}
				lnum++
				col = 0
				remember = true
				continue
			}

			// Search start of next word
			i, ok = SearchCharFunc(line, i, 1, func(c rune) bool {
				return !unicode.IsSpace(c)
			})
			if !ok {
				if isLastLine {
					return lnum, len(line) - 1
				}
				lnum++
				col = 0
				remember = true
				continue
			}
			return lnum, i
		}
	case "gE":
		lnum := m.cursorLine
		col := m.cursorColumn
		remember := false
		for {
			line := m.lines[lnum]
			isFirstLine := lnum == 0

			// Skip if empty
			if len(line) == 0 && !isFirstLine {
				// What happens if this is the current line
				if lnum == m.cursorLine {
					lnum--
					col = len(m.lines[lnum]) - 1
					remember = true
					continue
				}
				return lnum, 0
			}

			if remember {
				i, ok := SearchCharFunc(line, col, -1, func(c rune) bool {
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
				return lnum, i
			}

			// Search next whitespace
			i, ok := SearchCharFunc(line, col, -1, unicode.IsSpace)
			if !ok {
				if isFirstLine {
					return lnum, 0
				}
				lnum--
				col = len(m.lines[lnum]) - 1
				remember = true
				continue
			}

			// Search start of next word
			i, ok = SearchCharFunc(line, i, -1, func(c rune) bool {
				return !unicode.IsSpace(c)
			})
			if !ok {
				if isFirstLine {
					return lnum, 0
				}
				lnum--
				col = len(m.lines[lnum]) - 1
				remember = true
				continue
			}
			return lnum, i
		}
	case "ge":
		lnum := m.cursorLine
		col := m.cursorColumn
		remember := false
		for {
			line := m.lines[lnum]
			isFirstLine := lnum == 0

			// Skip if empty
			if len(line) == 0 && !isFirstLine {
				// What happens if this is the current line
				if lnum == m.cursorLine {
					lnum--
					col = len(m.lines[lnum]) - 1
					remember = true
					continue
				}
				return lnum, 0
			}

			if remember {
				i, ok := SearchCharFunc(line, col, -1, func(c rune) bool {
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
				return lnum, i
			}

			// Search previous non-matching char
			if col < 0 || len(line) == 0 {
				return lnum, 0
			}
			isKeyword := IsKeyword(line[col])
			isWhitespace := unicode.IsSpace(line[col])
			i, ok := SearchCharFunc(line, col, -1, func(c rune) bool {
				return unicode.IsSpace(c) || isWhitespace || isKeyword != IsKeyword(c)
			})
			if !ok {
				if isFirstLine {
					return lnum, 0
				}
				lnum--
				col = len(m.lines[lnum]) - 1
				remember = true
				continue
			}

			// Search start of next word
			i, ok = SearchCharFunc(line, i, -1, func(c rune) bool {
				return !unicode.IsSpace(c)
			})
			if !ok {
				if isFirstLine {
					return lnum, 0
				}
				lnum--
				col = len(m.lines[lnum]) - 1
				remember = true
				continue
			}
			return lnum, i
		}

	default:
		return Unchanged, Unchanged
	}
}

func (m *Model) handleOpendingModeKeys(key tea.KeyMsg) {
	if key.Type == tea.KeyEscape {
		m.ResetOpending(false)
		return
	}

	motion := key.String()

	// Text objects
	if motion == "i" && !m.imod {
		m.imod = true
		return
	} else if m.imod {
		m.imod = false
		motion = "i" + motion
	} else {
		m.imod = false
	}
	if motion == "a" && !m.amod {
		m.amod = true
		return
	} else if m.amod {
		m.amod = false
		motion = "a" + motion
	} else {
		m.amod = false
	}

	ftmod := m.fmod == 'f' || m.fmod == 't'
	lnum, col := m.Motion(motion)
	if m.gmod || m.fmod != NullChar {
		return
	}

	// For now hardcoding is fine
	// Should include all the "vertical" movement (j k - + gg G)
	isVerticalMotion := col == Unchanged || motion[0] == '-' || motion[0] == '+'

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
		if motion == "e" || motion == "E" {
			col++ // Adjust due to upper bound being exclusive
		}

		if lnum == m.cursorLine {
			line := m.lines[m.cursorLine]
			lower := min(col, m.cursorColumn, len(line))
			upper := min(max(col, m.cursorColumn), len(line))
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

		} else if isVerticalMotion {
			lower := min(lnum, m.cursorLine)
			upper := max(lnum, m.cursorLine) + 1
			if upper > len(m.lines) || lower < 0 {
				m.ResetOpending(false)
				return
			}

			var builder strings.Builder
			for i := lower; i < upper; i++ {
				builder.WriteString(string(m.lines[i]))
				builder.WriteByte('\n')
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

		} else {
			// Multiline but not horizontal movement (such as "word" motions)

			// Known issues (won't be fixed most likely, PRs are welcome):
			// "dw" doesn't delete last char if it's the only word in the last line
			// "dge" doesn't delete current char, in NVIM it does
			// "dge" doesn't combine lines in certain scenarios where NVIM does

			line := m.lines[m.cursorLine]

			isLastLine := len(m.lines)-1 == m.cursorLine
			if len(line) == 0 && len(m.lines) > 1 && !isLastLine {
				m.lines = slices.Delete(m.lines, m.cursorLine, m.cursorLine+1)
				m.ResetOpending(true)
				return
			}

			if lnum > m.cursorLine {
				col = len(line) // end
			} else {
				col = 0 // start
			}

			lower := min(col, m.cursorColumn, len(line))
			upper := min(max(col, m.cursorColumn), len(line))
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
		}

		m.ResetOpending(true)
		return
	}

	switch motion {
	case "c":
		m.Yank(string(m.lines[m.cursorLine]) + "\n")
		m.lines[m.cursorLine] = []rune("")
		m.SetCursorColumn(0)
		m.ResetOpending(true)

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
		m.ResetOpending(true)

	case "y":
		m.Yank(string(m.lines[m.cursorLine]) + "\n")
		m.ResetOpending(true)

	case "aW":
		fallthrough
	case "iW":
		line := m.lines[m.cursorLine]
		if len(line) == 0 {
			m.ResetOpending(true)
			return
		}

		isWhitespace := unicode.IsSpace(line[m.cursorColumn])
		lower := m.cursorColumn
		upper := m.cursorColumn + 1
		searchFunc := func(c rune) bool {
			return isWhitespace != unicode.IsSpace(c)
		}

		i, ok := SearchCharFunc(line, lower, -1, searchFunc)
		if ok {
			lower = i + 1
		} else {
			lower = 0
		}

		i, ok = SearchCharFunc(line, upper, 1, searchFunc)
		if ok {
			upper = i
		} else {
			upper = len(line)
		}

		value := line[lower:upper]
		m.Yank(string(value))

		if m.pending == 'd' || m.pending == 'c' {
			line = slices.Delete(line, lower, upper)
			m.lines[m.cursorLine] = line
			m.SetCursorColumn(min(lower, len(line)-1))
		}

		m.ResetOpending(true)

	case "aw":
		fallthrough
	case "iw":
		line := m.lines[m.cursorLine]
		if len(line) == 0 {
			m.ResetOpending(true)
			return
		}

		isKeyword := IsKeyword(line[m.cursorColumn])
		isWhitespace := unicode.IsSpace(line[m.cursorColumn])
		lower := m.cursorColumn
		upper := m.cursorColumn + 1
		searchFunc := func(c rune) bool {
			return isKeyword != IsKeyword(c) || isWhitespace != unicode.IsSpace(c)
		}

		i, ok := SearchCharFunc(line, lower, -1, searchFunc)
		if ok {
			lower = i + 1
		} else {
			lower = 0
		}

		i, ok = SearchCharFunc(line, upper, 1, searchFunc)
		if ok {
			upper = i
		} else {
			upper = len(line)
		}

		value := line[lower:upper]
		m.Yank(string(value))

		if m.pending == 'd' || m.pending == 'c' {
			line = slices.Delete(line, lower, upper)
			m.lines[m.cursorLine] = line
			m.SetCursorColumn(min(lower, len(line)-1))
		}

		m.ResetOpending(true)

	default:
		m.ResetOpending(false)
	}
}

func (m *Model) ResetOpending(allowInsert bool) {
	if m.pending == 'c' && allowInsert {
		m.mode = InsertMode
	} else {
		m.mode = NormalMode
	}
	m.imod = false
	m.amod = false
}

func (m *Model) Yank(s string) {
	m.register = s
}

func (m Model) Paste() string {
	return m.register
}

func (m *Model) handleVisualModeKeys(key tea.KeyMsg) {
	if key.Type == tea.KeyEscape {
		m.SetCursorColumn(min(m.cursorColumn, len(m.lines[m.cursorLine])-1))
		m.mode = NormalMode
		m.count = NoCount
		return
	}

	motion := key.String()
	if motion == "o" {
		line, col := m.vline, m.vcol
		m.vline = m.cursorLine
		m.vcol = m.cursorColumn
		m.SetCursorLine(line)
		m.SetCursorColumn(col)
		return
	}

	count := 1
	if m.count != NoCount && (motion[0] < '0' || motion[0] > '9') {
		count = m.count
		m.count = NoCount
	}

	shouldReturn := false
	for i := 0; i < count; i++ {
		line, col := m.Motion(motion)
		if line != Unchanged {
			m.SetCursorLine(line)
			shouldReturn = true
		}
		if col != Unchanged {
			m.SetCursorColumn(col)
			shouldReturn = true
		}
	}
	if shouldReturn {
		return
	}

	if motion != "x" && motion != "d" && motion != "c" && motion != "y" &&
		motion != "p" && motion != "P" {
		return
	}

	paste := ""
	if motion == "p" || motion == "P" {
		paste = m.Paste()
	}

	lower := min(m.cursorLine, m.vline)
	upper := max(m.cursorLine, m.vline)

	if m.cursorLine == m.vline {
		line := m.lines[m.cursorLine]
		lowerCol := min(m.cursorColumn, m.vcol)
		upperCol := max(m.cursorColumn, m.vcol)
		if lowerCol == len(line) {
			m.Yank("\n")
		} else if upperCol == len(line) {
			m.Yank(string(line[lowerCol:upperCol]) + "\n")
		} else {
			m.Yank(string(line[lowerCol : upperCol+1]))
		}
	} else {
		lowerCol := m.vcol
		upperCol := m.cursorColumn
		if m.vline > m.cursorLine {
			lowerCol = m.cursorColumn
			upperCol = m.vcol
		}

		var builder strings.Builder
		builder.WriteString(string(m.lines[lower][lowerCol:]))
		builder.WriteByte('\n')
		for i := lower + 1; i < upper; i++ {
			builder.WriteString(string(m.lines[i]))
			builder.WriteByte('\n')
		}
		builder.WriteString(string(m.lines[upper][:upperCol]))
		if upperCol == len(m.lines[upper]) {
			builder.WriteByte('\n')
		} else {
			builder.WriteRune(m.lines[upper][upperCol])
		}
		m.Yank(builder.String())
	}

	if motion == "y" {
		m.mode = NormalMode
		m.SetCursorLine(lower)
		col := min(m.vcol, m.cursorColumn)
		if m.vline > m.cursorLine {
			col = m.cursorColumn
		} else if m.vline < m.cursorLine {
			col = m.vcol
		}
		m.SetCursorColumn(col)
		return
	}

	if m.cursorLine == m.vline {
		line := m.lines[m.cursorLine]
		lowerCol := min(m.cursorColumn, m.vcol)
		upperCol := max(m.cursorColumn, m.vcol)
		if lowerCol == len(line) {
			if m.cursorLine != len(m.lines)-1 {
				line = append(line, m.lines[m.cursorLine+1]...)
				m.lines[m.cursorLine] = line
				m.lines = slices.Delete(m.lines, m.cursorLine+1, m.cursorLine+2)
			}
		} else if upperCol == len(line) {
			line = line[:lowerCol]
			if m.cursorLine != len(m.lines)-1 {
				line = append(line, m.lines[m.cursorLine+1]...)
				m.lines[m.cursorLine] = line
				m.lines = slices.Delete(m.lines, m.cursorLine+1, m.cursorLine+2)
			}
			m.SetCursorColumn(lowerCol)
		} else {
			line = slices.Delete(line, lowerCol, upperCol+1)
			m.lines[m.cursorLine] = line
			m.SetCursorColumn(lowerCol)
		}
	} else {
		lowerCol := m.vcol
		upperCol := m.cursorColumn
		if m.vline > m.cursorLine {
			lowerCol = m.cursorColumn
			upperCol = m.vcol
		}
		lowerLine := m.lines[lower]
		upperLine := m.lines[upper]

		lowerLine = lowerLine[:lowerCol]
		if upperCol != len(upperLine) {
			lowerLine = append(lowerLine, upperLine[upperCol+1:]...)
		} else if upper != len(m.lines)-1 {
			lowerLine = append(lowerLine, m.lines[upper+1]...)
			m.lines = slices.Delete(m.lines, upper+1, upper+2)
		}
		m.lines = slices.Delete(m.lines, lower+1, upper+1)

		m.lines[lower] = lowerLine
		m.SetCursorLine(lower)
		m.SetCursorColumn(lowerCol)
	}

	if motion == "c" {
		m.mode = InsertMode
	} else {
		m.mode = NormalMode
		m.SetCursorColumn(min(m.cursorColumn, len(m.lines[m.cursorLine])-1))
	}

	if motion == "p" || motion == "P" {
		copyPaste := m.Paste()
		m.Yank(paste)
		key.Runes[0] = 'P' // In visual it should always use backwards P
		m.handleNormalModeKeys(key)

		if motion == "p" {
			m.Yank(copyPaste) // Restore
		}
		// Note: we don't restore for "P" because we use it like
		// <leader>p (we paste over and keep what we had)
		// Shift+P in visual mode is pretty useless/uncommon
		// So using that instead of adding <leader>p makes more sense
	}
}

func (m *Model) handleVisualLineModeKeys(key tea.KeyMsg) {
	if key.Type == tea.KeyEscape {
		m.SetCursorColumn(min(m.cursorColumn, len(m.lines[m.cursorLine])-1))
		m.mode = NormalMode
		m.count = NoCount
		return
	}

	motion := key.String()
	if motion == "o" {
		line, col := m.vline, m.vcol
		m.vline = m.cursorLine
		m.vcol = m.cursorColumn
		m.SetCursorLine(line)
		m.SetCursorColumn(col)
		return
	}

	count := 1
	if m.count != NoCount && (motion[0] < '0' || motion[0] > '9') {
		count = m.count
		m.count = NoCount
	}

	shouldReturn := false
	for i := 0; i < count; i++ {
		line, col := m.Motion(motion)
		if line != Unchanged {
			m.SetCursorLine(line)
			shouldReturn = true
		}
		if col != Unchanged {
			m.SetCursorColumn(col)
			shouldReturn = true
		}
	}
	if shouldReturn {
		return
	}

	if motion != "x" && motion != "d" && motion != "c" && motion != "y" &&
		motion != "p" && motion != "P" {
		return
	}

	lower := min(m.cursorLine, m.vline)
	upper := max(m.cursorLine, m.vline) + 1

	paste := ""
	if motion == "p" || motion == "P" {
		paste = m.Paste()
	}

	var builder strings.Builder
	for i := lower; i < upper; i++ {
		builder.WriteString(string(m.lines[i]))
		builder.WriteByte('\n')
	}
	m.Yank(builder.String())

	if motion == "y" {
		m.mode = NormalMode
		if lower == m.vline {
			m.SetCursorLine(m.vline)
			m.SetCursorColumn(0)
		}
		return
	}

	if motion == "c" {
		m.lines = slices.Delete(m.lines, lower+1, upper)
		m.lines[lower] = []rune{}

		m.mode = InsertMode
		m.SetCursorLine(lower)
		m.SetCursorColumn(0)
	}

	if motion == "d" || motion == "x" || motion == "p" || motion == "P" {
		includesLastLine := upper == len(m.lines)
		m.lines = slices.Delete(m.lines, lower, upper)
		if len(m.lines) == 0 {
			m.lines = append(m.lines, []rune{})
		}
		m.SetCursorLine(min(lower, len(m.lines)-1))
		m.SetCursorColumn(min(m.cursorColumn, len(m.lines[m.cursorLine])-1))
		m.mode = NormalMode

		if motion == "p" || motion == "P" {
			if includesLastLine {
				m.lines = append(m.lines, []rune{})
				m.SetCursorLine(len(m.lines) - 1)
				m.SetCursorColumn(0)
			} else {
				m.lines = slices.Insert(m.lines, m.cursorLine, []rune{})
				m.SetCursorColumn(0)
			}

			copyPaste := m.Paste()
			m.Yank(paste)
			key.Runes[0] = 'p' // In visual line it should always use forward p
			// This is bcz we are pasting on a blank line
			m.handleNormalModeKeys(key)

			if motion == "p" {
				m.Yank(copyPaste) // Restore
			}
			// Note: we don't restore for "P" because we use it like
			// <leader>p (we paste over and keep what we had)
			// Shift+P in visual mode is pretty useless/uncommon
			// So using that instead of adding <leader>p makes more sense
		}
	}
}

func (m Model) Mode() int {
	return m.mode
}

func (m *Model) SetMode(mode int) {
	m.mode = mode
}

func (m *Model) Save() {
	if len(m.undoStack) != 0 {
		last := m.undoStack[len(m.undoStack)-1]
		equal := slices.EqualFunc(last.lines, m.lines, slices.Equal)
		if equal {
			return
		}
	}

	copyLines := make([][]rune, len(m.lines))
	for i, line := range m.lines {
		copyLine := make([]rune, len(line))
		copy(copyLine, line)
		copyLines[i] = copyLine
	}

	m.undoStack = append(m.undoStack, State{
		lines:        copyLines,
		cursorLine:   m.cursorLine,
		cursorColumn: m.cursorColumn,
	})
	m.redoStack = nil
}

func (m *Model) Undo() {
	for {
		if len(m.undoStack) == 0 {
			return
		}
		last := len(m.undoStack) - 1
		equal := slices.EqualFunc(m.undoStack[last].lines, m.lines, slices.Equal)
		if equal {
			if len(m.undoStack) == 1 {
				return
			}
			m.undoStack = m.undoStack[:last]
		} else {
			break
		}
	}

	last := len(m.undoStack) - 1
	m.redoStack = append(m.redoStack, State{
		lines:        m.lines,
		cursorLine:   m.cursorLine,
		cursorColumn: m.cursorColumn,
	})

	lines := m.undoStack[last].lines
	copyLines := make([][]rune, len(lines))
	for i, line := range lines {
		copyLine := make([]rune, len(line))
		copy(copyLine, line)
		copyLines[i] = copyLine
	}
	m.lines = copyLines
	m.cursorLine = m.undoStack[last].cursorLine
	m.cursorColumn = m.undoStack[last].cursorColumn
}

func (m *Model) Redo() {
	if len(m.redoStack) == 0 {
		return
	}

	m.undoStack = append(m.undoStack, State{
		lines:        m.lines,
		cursorLine:   m.cursorLine,
		cursorColumn: m.cursorColumn,
	})

	m.lines = m.redoStack[len(m.redoStack)-1].lines
	m.cursorLine = m.redoStack[len(m.redoStack)-1].cursorLine
	m.cursorColumn = m.redoStack[len(m.redoStack)-1].cursorColumn

	m.redoStack = m.redoStack[:len(m.redoStack)-1]

	copyLines := make([][]rune, len(m.lines))
	for i, line := range m.lines {
		copyLine := make([]rune, len(line))
		copy(copyLine, line)
		copyLines[i] = copyLine
	}
	m.undoStack = append(m.undoStack, State{
		lines:        copyLines,
		cursorLine:   m.cursorLine,
		cursorColumn: m.cursorColumn,
	})
}

func (m *Model) Count() int {
	sum := 0
	for _, line := range m.lines {
		sum += len(line)
	}
	return sum + len(m.lines) - 1
}

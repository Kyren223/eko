package loadscreen

import (
	"time"

	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/pkg/assert"
)

var (
	style         = lipgloss.NewStyle().Border(lipgloss.ThickBorder()).Padding(1, 3).MarginTop(2)
	loadingFrames = circleTrail(4, 4, 0, true, "  ", "░░", "▒▒", "▓▓", "██")
	loading       = spinner.Spinner{
		Frames: loadingFrames,
		FPS:    time.Second / time.Duration(len(loadingFrames)),
	}
)

func circleTrail(width int, height int, offset int, clockwise bool, bg string, trail ...string) (states []string) {
	slots := (width+height)*2 - 4
	slotWidth := lipgloss.Width(bg)
	slotHeight := lipgloss.Height(bg)

	assert.Assert(slots-len(trail) >= 0, "not enough space to fit trail")
	assert.AddData("slotWidth", slotWidth)
	assert.AddData("slotHeight", slotHeight)
	defer assert.RemoveData("slotWidth")
	defer assert.RemoveData("slotHeight")
	for _, elem := range trail {
		w := lipgloss.Width(elem)
		h := lipgloss.Height(elem)
		assert.AddData("elemWidth", w)
		assert.AddData("elemHeight", h)
		assert.Assert(slotWidth == w, "bg and all trails must have the same height")
		assert.Assert(slotHeight == h, "bg and all trails must have the same height")
		assert.RemoveData("elemWidth")
		assert.RemoveData("elemHeight")
	}

	matrix := make([][]string, height)
	for i := range matrix {
		matrix[i] = make([]string, width)
	}

	for i := 0; i < slots; i++ {
		for y := range matrix {
			for x := range matrix[y] {
				matrix[y][x] = bg
			}
		}

		x, y := 0, 0
		for j := 0; j < i+offset; j++ {
			if clockwise {
				if y == 0 && x != width-1 {
					x++
				} else if x == width-1 && y != height-1 {
					y++
				} else if y == height-1 && x != 0 {
					x--
				} else if x == 0 && y != 0 {
					y--
				}
			} else {
				if x == 0 && y != height-1 {
					y++
				} else if y == height-1 && x != width-1 {
					x++
				} else if x == width-1 && y != 0 {
					y--
				} else if y == 0 && x != 0 {
					x--
				}
			}
		}

		for _, trail := range trail {
			matrix[y][x] = trail
			if clockwise {
				if y == 0 && x != width-1 {
					x++
				} else if x == width-1 && y != height-1 {
					y++
				} else if y == height-1 && x != 0 {
					x--
				} else if x == 0 && y != 0 {
					y--
				}
			} else {
				if x == 0 && y != height-1 {
					y++
				} else if y == height-1 && x != width-1 {
					x++
				} else if x == width-1 && y != 0 {
					y--
				} else if y == 0 && x != 0 {
					x--
				}
			}
		}

		var rows []string
		for _, row := range matrix {
			rows = append(rows, lipgloss.JoinHorizontal(0, row...))
		}
		state := lipgloss.JoinVertical(0, rows...)
		states = append(states, state)
	}

	return states
}

type Updater func(msg tea.Msg) tea.Cmd

type Model struct {
	sp      spinner.Model
	content string
	updater Updater
}

func New(content string, updater Updater) Model {
	width := lipgloss.Width(content)
	height := lipgloss.Height(content)
	content = lipgloss.NewStyle().Width(width).Height(height).Render(content)
	return Model{
		sp:      spinner.New(spinner.WithSpinner(loading)),
		content: style.Render(content),
	}
}

func (m Model) Init() tea.Cmd {
	return m.sp.Tick
}

func (m Model) View() string {
	return lipgloss.Place(
		ui.Width, ui.Height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, m.sp.View(), m.content),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		key := msg.Type
		switch key {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.sp, cmd = m.sp.Update(msg)
		return m, cmd
	}

	return m, m.updater(msg)
}

package client

import (
	"log"
	"reflect"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"

	"github.com/kyren223/eko/internal/client/config"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/auth"
	"github.com/kyren223/eko/pkg/assert"
)

type BubbleTeaCloser struct {
	program *tea.Program
}

func (c BubbleTeaCloser) Close() error {
	c.program.Kill()
	return nil
}

func Run() {
	log.Println("client started")

	err := config.Load()
	if err != nil {
		log.Fatalf("Config file at '%v' was unable to load successfully\n%v\n", config.ConfigFile, err)
	}

	// HACK: Sets the terminal color and keeps it that way
	// I am not aware of a way to query the current background color (to revert this)
	// Ideally bubbletea will handle the setting/reventing, for now it's fine
	// It only changes the current pane so new terminal panes/windows are not affected.
	termenv.DefaultOutput().SetBackgroundColor(termenv.RGBColor(ui.BackgroundColor))

	program := tea.NewProgram(initialModel(), tea.WithAltScreen())
	assert.AddFlush(BubbleTeaCloser{program})
	ui.Program = program

	if _, err := program.Run(); err != nil {
		log.Println(err)
	}
}

type model struct {
	model tea.Model
}

func initialModel() model {
	return model{auth.New()}
}

func (m model) Init() tea.Cmd {
	return m.model.Init()
}

func (m model) View() string {
	return m.model.View()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ui.Width, ui.Height = msg.Width, msg.Height
		return m, nil

	case ui.ModelTransition:
		log.Println("Transition model from", reflect.TypeOf(m.model).String(), "to", reflect.TypeOf(msg.Model).String())
		m.model = msg.Model
		return m, m.model.Init()

	default:
		var cmd tea.Cmd
		m.model, cmd = m.model.Update(msg)
		return m, cmd
	}
}

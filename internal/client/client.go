package client

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/davecgh/go-spew/spew"

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
	var dump *os.File
	if ui.DEBUG {
		var err error
		dump, err = os.OpenFile("messages.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			os.Exit(1)
		}
	}

	log.Println("client started")

	err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Config file at '%v' was unable to load successfully\n%v\n", config.ConfigFile, err)
		os.Exit(1)
	}
	err = config.LoadCache()
	if err != nil {
		fmt.Printf("Cache file at '%v' was unable to load successfully\n%v\n", config.CacheFile, err)
		os.Exit(1)
	}

	program := tea.NewProgram(initialModel(dump), tea.WithAltScreen())
	assert.AddFlush(BubbleTeaCloser{program})
	ui.Program = program

	if _, err := program.Run(); err != nil {
		log.Println(err)
	}
}

type model struct {
	dump  io.Writer
	model tea.Model
}

func initialModel(dump io.WriteCloser) model {
	return model{
		dump:  dump,
		model: auth.New(),
	}
}

func (m model) Init() tea.Cmd {
	return m.model.Init()
}

func (m model) View() string {
	return m.model.View()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.dump != nil {
		spew.Fdump(m.dump, msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, func() tea.Msg { return ui.QuitMsg{} }
		}

	case tea.WindowSizeMsg:
		ui.Width, ui.Height = msg.Width, msg.Height

	case ui.ModelTransition:
		log.Println("Transition model from", reflect.TypeOf(m.model).String(), "to", reflect.TypeOf(msg.Model).String())
		m.model = msg.Model
		return m, m.model.Init()

	}
	var cmd tea.Cmd
	m.model, cmd = m.model.Update(msg)

	if _, ok := msg.(ui.QuitMsg); ok {
		return m, tea.Quit
	}

	return m, cmd
}

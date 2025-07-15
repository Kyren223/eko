package client

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/davecgh/go-spew/spew"

	"github.com/kyren223/eko/internal/client/config"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/auth"
	"github.com/kyren223/eko/internal/client/ui/colors"
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
	if IsTooSmall() {
		return TooSmallView()
	}

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

	if _, ok := msg.(ui.QuitMsg); ok {
		m.model, _ = m.model.Update(msg)
		return m, tea.Quit
	}

	var cmd tea.Cmd
	if !IsTooSmall() {
		m.model, cmd = m.model.Update(msg)
	}

	return m, cmd
}

func IsTooSmall() bool {
	return ui.Width < ui.MinWidth || ui.Height < ui.MinHeight
}

func TooSmallView() string {
	style := lipgloss.NewStyle().Background(colors.Background).Foreground(colors.White).Bold(true)

	wcolor := colors.Green
	if ui.Width < ui.MinWidth {
		wcolor = colors.Red
	}

	hcolor := colors.Green
	if ui.Height < ui.MinHeight {
		hcolor = colors.Red
	}

	var b strings.Builder

	b.WriteString(style.Render("Terminal size too small:"))
	b.WriteByte('\n')
	b.WriteString(style.Render("Width = "))
	b.WriteString(style.Foreground(wcolor).Render(strconv.FormatInt(int64(ui.Width), 10)))
	b.WriteString(style.Render(" Height = "))
	b.WriteString(style.Foreground(hcolor).Render(strconv.FormatInt(int64(ui.Height), 10)))

	b.WriteByte('\n')
	b.WriteByte('\n')

	b.WriteString(style.Render("Minimum size required:"))
	b.WriteByte('\n')
	b.WriteString(style.Render("Width = "))
	b.WriteString(style.Render(strconv.FormatInt(int64(ui.MinWidth), 10)))
	b.WriteString(style.Render(" Height = "))
	b.WriteString(style.Render(strconv.FormatInt(int64(ui.MinHeight), 10)))

	content := b.String()

	// return content
	return lipgloss.Place(ui.Width, ui.Height, lipgloss.Center, lipgloss.Center, content, lipgloss.WithWhitespaceBackground(colors.Background))
}

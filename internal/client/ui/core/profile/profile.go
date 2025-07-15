package profile

import (
	"crypto/ed25519"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
	"golang.org/x/crypto/ssh"
)

var width = 70

type Model struct {
	id snowflake.ID
}

func New(userId snowflake.ID) Model {
	return Model{
		id: userId,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	user := state.State.Users[m.id]

	var builder strings.Builder

	headerStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		Width(width).
		Background(colors.Background).
		Foreground(colors.Focus)

	name := user.Name
	name = lipgloss.NewStyle().
		MarginBackground(colors.Background).
		Background(colors.Gold).
		Foreground(colors.Background).
		Bold(true).
		Render(name)
	paddingLeft := lipgloss.NewStyle().
		MarginLeft(2).
		Background(colors.Background).
		Foreground(colors.Gold).
		Render("")
	paddingRight := lipgloss.NewStyle().
		Background(colors.Background).
		Foreground(colors.Gold).
		Render("")
	builder.WriteString(paddingLeft)
	builder.WriteString(name)
	builder.WriteString(paddingRight)
	builder.WriteByte('\n')
	builder.WriteByte('\n')

	publicDMs := "Private DMs"
	if user.IsPublicDM {
		publicDMs = "Public DMs"
	}

	id := strconv.FormatInt(int64(user.ID), 10)
	info := lipgloss.NewStyle().
		PaddingLeft(2).
		Width(width).
		Background(colors.Background).
		Foreground(colors.LightGray).
		Render("ID: " + id + " | " + publicDMs)
	builder.WriteString(info)
	builder.WriteByte('\n')
	builder.WriteByte('\n')

	publicKeyHeader := headerStyle.Render("Public Key (ssh-ed25519)")
	publicKey := lipgloss.NewStyle().
		PaddingLeft(2).
		Width(width).
		Render(publicKeyToSshString(user.PublicKey))
	builder.WriteString(publicKeyHeader)
	builder.WriteByte('\n')
	builder.WriteByte('\n')
	builder.WriteString(publicKey)
	builder.WriteByte('\n')

	aboutMeHeader := headerStyle.Render("About me")
	builder.WriteString(aboutMeHeader)
	builder.WriteByte('\n')
	builder.WriteByte('\n')

	textBg := colors.BackgroundHighlight
	textFg := colors.White
	bg := colors.Background

	description := user.Description
	if description == "" {
		description = lipgloss.NewStyle().
			Foreground(colors.Purple).
			Render("No description was provided")
	}

	excessWidth := 4 - 2
	bgStyle := lipgloss.NewStyle().Background(textBg).Foreground(bg)
	topMiddle := strings.Repeat(" ", width-excessWidth)
	top := bgStyle.Render("🭠🭘" + topMiddle + "🭣🭕")
	middle := lipgloss.NewStyle().Width(width+2).Padding(0, 2).
		Background(textBg).Foreground(textFg).Render(description)
	bgStyle2 := lipgloss.NewStyle().Foreground(textBg)
	bottomMiddle := strings.Repeat("█", width-excessWidth)
	bottom := bgStyle2.Render("🭥🭓" + bottomMiddle + "🭞🭚")
	aboutMe := lipgloss.JoinVertical(lipgloss.Left, top, middle, bottom)

	builder.WriteString(aboutMe)

	content := builder.String()

	return lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		Padding(1, 2).
		Align(lipgloss.Left, lipgloss.Center).
		BorderBackground(colors.Background).
		BorderForeground(colors.White).
		Background(colors.Background).
		Foreground(colors.White).
		Render(content)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func publicKeyToSshString(pub ed25519.PublicKey) string {
	sshPubKey, err := ssh.NewPublicKey(pub)
	assert.NoError(err, "converting to ssh key should never fail")
	sshPubKeyBytes := ssh.MarshalAuthorizedKey(sshPubKey)
	return strings.Split(string(sshPubKeyBytes), " ")[1]
}

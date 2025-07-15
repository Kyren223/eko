package core

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/colors"
)

const (
	HelpNetworkList = iota
	HelpSignalList
	HelpFrequencyList
	HelpChat
	HelpMemberList
	HelpGlobal
	HelpVim
	HelpBanList
	HelpMax
)

type Keymap struct {
	key         string
	description string
}

type HelpPopup struct {
	help   int
	global bool
}

func NewHelpPopup(help int) *HelpPopup {
	return &HelpPopup{
		help:   help,
		global: false,
	}
}

func (m HelpPopup) Init() tea.Cmd {
	return nil
}

func (m HelpPopup) View() string {
	if m.global {
		m.help = HelpGlobal
	}

	title := lipgloss.NewStyle().
		Background(colors.Background).
		Foreground(colors.Focus).
		Border(lipgloss.ThickBorder(), false, false, true).
		Padding(0, 4, 1).MarginBottom(1).
		Align(lipgloss.Center, lipgloss.Center).
		Render(m.Title()+" Keybindings Cheatsheet") + "\n"

	descriptionStyle := lipgloss.NewStyle().
		PaddingLeft(1).
		Background(colors.Background).
		Foreground(colors.White)

	keyStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Background(colors.DarkerCyan).
		Foreground(colors.White)

	keymapLists := [][]Keymap{}
	switch m.help {
	case HelpNetworkList:
		keymapLists = m.HelpNetworkList()
	case HelpSignalList:
		keymapLists = m.HelpSignalList()
	case HelpFrequencyList:
		keymapLists = m.HelpFrequencyList()
	case HelpChat:
		keymapLists = m.HelpChat()
	case HelpMemberList:
		keymapLists = m.HelpMemberList()
	case HelpGlobal:
		keymapLists = m.HelpGlobal()
	case HelpVim:
		keymapLists = m.HelpVim()
	case HelpBanList:
		keymapLists = m.HelpBanList()
	}

	helps := []string{}

	// Must known max size beforehand due to JoinHorizontal issue
	// See https://github.com/charmbracelet/lipgloss/issues/209
	largestSize := 0
	for _, keymaps := range keymapLists {
		if len(keymaps) > largestSize {
			largestSize = len(keymaps)
		}
	}

	for i, keymaps := range keymapLists {
		var builder strings.Builder

		for _, keymap := range keymaps {
			builder.WriteString(keyStyle.Render(keymap.key))
			builder.WriteString(descriptionStyle.Render(keymap.description))
			builder.WriteString("\n\n")
		}

		help := lipgloss.NewStyle().
			Height(largestSize*2 + 1).
			Background(colors.Background).
			Render(builder.String())
		if i != len(keymapLists)-1 {
			help = lipgloss.NewStyle().
				PaddingRight(2).
				Background(colors.Background).
				Render(builder.String())
		}

		helps = append(helps, help)

	}

	content := lipgloss.JoinHorizontal(lipgloss.Top, helps...) + "\n"

	toggleHint := lipgloss.NewStyle().Background(colors.Background).Foreground(colors.Gold).
		Render("-- Press space to toggle between global/local keybindings --")

	return lipgloss.NewStyle().
		MaxWidth(ui.MinWidth).
		Border(lipgloss.ThickBorder()).
		Padding(1, 2).
		Align(lipgloss.Center, lipgloss.Center).
		BorderBackground(colors.Background).
		BorderForeground(colors.White).
		Background(colors.Background).
		Foreground(colors.White).
		Render(title + content + toggleHint)
}

func (m HelpPopup) Update(msg tea.Msg) (HelpPopup, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeySpace {
			m.global = !m.global
		}
	}

	return m, nil
}

func (m HelpPopup) Title() string {
	switch m.help {
	case HelpNetworkList:
		return "Network"
	case HelpSignalList:
		return "Signals"
	case HelpFrequencyList:
		return "Frequency"
	case HelpChat:
		return "Chat"
	case HelpMemberList:
		return "Member"
	case HelpGlobal:
		return "Global"
	case HelpVim:
		return "VIM"
	case HelpBanList:
		return "Ban List"
	}
	return "Unknown"
}

func (m HelpPopup) HelpGlobal() [][]Keymap {
	return [][]Keymap{{
		{"ctrl+c", "Exit eko"},
		{"H", "Move focus to the left"},
		{"L", "Move focus to the right"},
		{"s", "User profile settings"},
		{"?", "Show a help popup"},
	}, {
		{"esc", "Close popup"},
		{"tab", "cycle to the next option"},
		{"shift+tab", "cycle to the previous option"},
		{"enter", "confirm the selected option"},
	}}
}

func (m HelpPopup) HelpNetworkList() [][]Keymap {
	return [][]Keymap{{
		{"k", "Move up a network"},
		{"j", "Move down a network"},
		{"K", "Move the selected network up"},
		{"J", "Move the selected network down"},
		{"Q", "Leave the selected network"},
	}, {
		{"n", "Create a new network"},
		{"e", "Edit the selected network"},
		{"a", "Join a new network from an invite code"},
		{"i", "Copy the selected network's invite code"},
		{"D", "Delete the selected network"},
	}}
}

func (m HelpPopup) HelpFrequencyList() [][]Keymap {
	return [][]Keymap{{
		{"k", "Move up a frequency"},
		{"j", "Move down a frequency"},
		{"ctrl+u", "Move half a page up"},
		{"ctrl+d", "Move half a page down"},
		{"g", "Move to the first frequency"},
		{"G", "Move to the last frequency"},
	}, {
		{"n", "Create a new frequency"},
		{"e", "Edit the selected frequency"},
		{"x", "Delete the selected frequency"},
		{"K", "Move the selected frequency up"},
		{"J", "Move the selected frequency down"},
		{"i", "Copy the network's invite code"},
	}}
}

func (m HelpPopup) HelpSignalList() [][]Keymap {
	return [][]Keymap{{
		{"k", "Move up a user"},
		{"j", "Move down a user"},
		{"ctrl+u", "Move half a page up"},
		{"ctrl+d", "Move half a page down"},
		{"g", "Move to the top"},
		{"G", "Move to the bottom"},
	}, {
		{"a", "Add new user signal"},
		{"c", "Close user signal"},
		{"T", "Trust/untrust user"},
		{"B", "Block user"},
		{"U", "Unblock user"},
		{"i", "Copy your user ID"},
	}}
}

func (m HelpPopup) HelpChat() [][]Keymap {
	return [][]Keymap{{
		{"k", "Move up a message"},
		{"j", "Move down a message"},
		{"ctrl+u", "Move half a page up"},
		{"ctrl+d", "Move half a page down"},
		{"G", "Snap to the bottom message"},
		{"i", "Start typing a message"},
		{"enter", "Snap to bottom and type"},
		{"ctrl+q", "Exit typing mode"},

		{"x", "Delete selected message"},
		{"e", "Edit selected message"},
	}, {
		{"K", "Kick message sender"},
		{"M", "Mute message sender"},
		{"U", "Unmute message sender"},
		{"B", "Ban message sender"},
		{"P", "Promote sender to admin"},
		{"D", "Demote sender from admin"},
		{"b", "Block user"},
		{"u", "Unblock user"},
		{"T", "Trust/Untrust user"},
		{"p", "View user profile"},
	}}
}

func (m HelpPopup) HelpMemberList() [][]Keymap {
	return [][]Keymap{{
		{"k", "Move up by one"},
		{"j", "Move down by one"},
		{"ctrl+u", "Move half a page up"},
		{"ctrl+d", "Move half a page down"},
		{"g", "Move to the top"},
		{"G", "Move to the bottom"},

		{"p", "View member profile"},
		{"T", "Trust/untrust member"},
	}, {
		{"K", "Kick selected member"},
		{"M", "Mute selected member"},
		{"U", "Unmute selected member"},
		{"B", "Ban selected member"},
		{"b", "Switch to banlist view"},
		{"P", "Promote member to admin"},
		{"D", "Demote member from admin"},
		{"ctrl+t", "Transfer ownership"}, // TODO: implement
	}}
}

func (m HelpPopup) HelpVim() [][]Keymap {
	return [][]Keymap{{
		{"esc", "Go into normal mode from insert mode"},
		{"q", "Exit typing from normal mode"},
		{"ctrl+q", "Exit typing from insert mode"},
		{"other", "The rest of the vim keys work as usual"},
	}, {}}
}

func (m HelpPopup) HelpBanList() [][]Keymap {
	return [][]Keymap{{
		{"k", "Move up by one"},
		{"j", "Move down by one"},
		{"ctrl+u", "Move half a page up"},
		{"ctrl+d", "Move half a page down"},
		{"g", "Move to the top"},
		{"G", "Move to the bottom"},
	}, {
		{"p", "View user profile"},
		{"U", "Unban user"},
		{"V", "View ban reason"},
		{"b", "Switch to member list view"},
	}}
}

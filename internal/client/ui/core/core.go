package core

import (
	"crypto/ed25519"
	"fmt"
	"log"
	"math"
	"slices"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/client/ui/colors"
	"github.com/kyren223/eko/internal/client/ui/core/banreason"
	"github.com/kyren223/eko/internal/client/ui/core/banview"
	"github.com/kyren223/eko/internal/client/ui/core/chat"
	"github.com/kyren223/eko/internal/client/ui/core/frequencycreation"
	"github.com/kyren223/eko/internal/client/ui/core/frequencylist"
	"github.com/kyren223/eko/internal/client/ui/core/frequencyupdate"
	"github.com/kyren223/eko/internal/client/ui/core/memberlist"
	"github.com/kyren223/eko/internal/client/ui/core/networkcreation"
	"github.com/kyren223/eko/internal/client/ui/core/networkjoin"
	"github.com/kyren223/eko/internal/client/ui/core/networklist"
	"github.com/kyren223/eko/internal/client/ui/core/networkupdate"
	"github.com/kyren223/eko/internal/client/ui/core/profile"
	"github.com/kyren223/eko/internal/client/ui/core/signaladd"
	"github.com/kyren223/eko/internal/client/ui/core/signallist"
	"github.com/kyren223/eko/internal/client/ui/core/state"
	"github.com/kyren223/eko/internal/client/ui/core/usersettings"
	"github.com/kyren223/eko/internal/client/ui/loadscreen"
	"github.com/kyren223/eko/internal/client/ui/viminput"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

var (
	connectingToServer = "Connecting to server.."
	connectionFailed   = "Connection failed - retrying in %d sec..."
	connectionTimeout  = 5 * time.Second
	initialTimeout     = 3750 * time.Millisecond
	timerInterval      = 50 * time.Millisecond
)

const (
	NetworkWidth      = 9
	SidebarPercentage = 0.20
	MinSidebarWidth   = 16
)

const (
	FocusNetworkList = iota
	FocusLeftSidebar
	FocusChat
	FocusRightSidebar
	FocusMax
)

type Model struct {
	name    string
	privKey ed25519.PrivateKey

	loading   loadscreen.Model
	timer     timer.Model
	timeout   time.Duration
	connected bool

	helpPopup              *HelpPopup
	userSettingsPopup      *usersettings.Model
	networkCreationPopup   *networkcreation.Model
	networkUpdatePopup     *networkupdate.Model
	networkJoinPopup       *networkjoin.Model
	frequencyCreationPopup *frequencycreation.Model
	frequencyUpdatePopup   *frequencyupdate.Model
	banReasonPopup         *banreason.Model
	banViewPopup           *banview.Model
	signalAddPopup         *signaladd.Model
	profilePopup           *profile.Model
	networkList            networklist.Model
	signalList             signallist.Model
	frequencyList          frequencylist.Model
	memberList             memberlist.Model
	chat                   chat.Model
	focus                  int
}

func New(privKey ed25519.PrivateKey, name string) Model {
	m := Model{
		name:                   name,
		privKey:                privKey,
		loading:                loadscreen.New(connectingToServer),
		timer:                  newTimer(initialTimeout),
		timeout:                initialTimeout,
		connected:              false,
		helpPopup:              nil,
		userSettingsPopup:      nil,
		networkCreationPopup:   nil,
		networkUpdatePopup:     nil,
		networkJoinPopup:       nil,
		frequencyCreationPopup: nil,
		frequencyUpdatePopup:   nil,
		banReasonPopup:         nil,
		banViewPopup:           nil,
		signalAddPopup:         nil,
		networkList:            networklist.New(),
		signalList:             signallist.New(),
		frequencyList:          frequencylist.New(),
		memberList:             memberlist.New(),
		chat:                   chat.New(),
		focus:                  FocusNetworkList,
	}
	m.move(0) // Update focus

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(gateway.Connect(m.privKey, connectionTimeout), m.loading.Init())
}

func (m Model) View() string {
	if !m.connected {
		return m.loading.View()
	}

	if m.HasPopup() {
		colors.Darken()
	}

	networkList := m.networkList.View()
	var leftSidebar string
	if m.networkList.Index() == networklist.SignalsIndex {
		leftSidebar = m.signalList.View()
	} else {
		leftSidebar = m.frequencyList.View()
	}
	chat := m.chat.View()
	rightSidebar := m.memberList.View()
	result := lipgloss.JoinHorizontal(lipgloss.Top, networkList, leftSidebar, chat, rightSidebar)

	result = lipgloss.Place(
		ui.Width, ui.Height,
		lipgloss.Left, lipgloss.Top,
		result,
	)

	if m.HasPopup() {
		colors.Restore()
		var popup string
		if m.helpPopup != nil {
			popup = m.helpPopup.View()
		} else if m.userSettingsPopup != nil {
			popup = m.userSettingsPopup.View()
		} else if m.networkCreationPopup != nil {
			popup = m.networkCreationPopup.View()
		} else if m.networkUpdatePopup != nil {
			popup = m.networkUpdatePopup.View()
		} else if m.frequencyCreationPopup != nil {
			popup = m.frequencyCreationPopup.View()
		} else if m.frequencyUpdatePopup != nil {
			popup = m.frequencyUpdatePopup.View()
		} else if m.networkJoinPopup != nil {
			popup = m.networkJoinPopup.View()
		} else if m.banReasonPopup != nil {
			popup = m.banReasonPopup.View()
		} else if m.banViewPopup != nil {
			popup = m.banViewPopup.View()
		} else if m.signalAddPopup != nil {
			popup = m.signalAddPopup.View()
		} else if m.profilePopup != nil {
			popup = m.profilePopup.View()
		} else {
			assert.Never("missing handling of a popup!")
		}

		x := (ui.Width - lipgloss.Width(popup)) / 2
		y := (ui.Height - lipgloss.Height(popup)) / 2
		result = ui.PlaceOverlay(x, y, popup, result)
	}

	return result
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.connected {
		cmd := m.updateConnected(msg)
		return m, cmd
	} else {
		cmd := m.updateNotConnected(msg)
		return m, cmd
	}
}

func (m *Model) updateNotConnected(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case gateway.ConnectionEstablished:
		state.UserID = (*snowflake.ID)(&msg)
		m.connected = true
		m.timeout = initialTimeout

		var setName tea.Cmd
		if m.name != "" {
			setName = gateway.Send(&packet.SetUserData{
				Data: nil,
				User: &data.User{
					Name:        m.name,
					Description: "",
					IsPublicDM:  true,
				},
			})
			m.name = ""
		}

		return tea.Batch(m.timer.Stop(), setName)

	case gateway.ConnectionFailed:
		log.Println("failed to connect:", msg)
		m.timer = newTimer(m.timeout)
		m.updateLoadScreenContent()
		return m.timer.Start()

	case timer.TimeoutMsg:
		m.timeout = min(m.timeout*2, time.Minute)
		m.loading.SetContent(connectingToServer)
		return gateway.Connect(m.privKey, connectionTimeout)

	case timer.StartStopMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return cmd

	case timer.TickMsg:
		m.updateLoadScreenContent()
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return cmd

	case spinner.TickMsg:
		var loadscreenCmd tea.Cmd
		m.loading, loadscreenCmd = m.loading.Update(msg)
		return loadscreenCmd

	default:
		return nil
	}
}

func (m *Model) updateConnected(message tea.Msg) tea.Cmd {
	totalWidth := max(ui.Width, ui.MinWidth)
	totalWidth -= NetworkWidth
	sidebarWidth := int(math.Round(float64(totalWidth) * SidebarPercentage))
	sidebarWidth = max(sidebarWidth, MinSidebarWidth)
	chatWidth := totalWidth - (2 * (sidebarWidth + 1))

	// log.Println("Widths:", ui.Width)
	// log.Println("sidebarWidth:", sidebarWidth)
	// log.Println("chatWidth:", chatWidth)

	m.signalList.SetWidth(sidebarWidth)
	m.frequencyList.SetWidth(sidebarWidth)
	m.memberList.SetWidth(sidebarWidth)
	m.chat.SetWidth(chatWidth)

	switch msg := message.(type) {
	case ui.QuitMsg:
		state.SendFinalData() // blocks
		gateway.Disconnect()

	case gateway.ConnectionLost:
		state.UserID = nil
		m.connected = false
		m.timeout = initialTimeout
		return tea.Batch(gateway.Connect(m.privKey, connectionTimeout), m.loading.Init())

	case *packet.Error:
		err := "new connection from another location, closing this one"
		if err == msg.Error {
			return ui.Transition(ui.NewAuth())
		}

	case *packet.SetUserData:
		if msg.User != nil {
			state.State.Users[msg.User.ID] = *msg.User
		}
		if msg.Data != nil {
			state.FromJsonUserData(*msg.Data)
		}

	case *packet.NetworksInfo:
		state.UpdateNetworks(msg)
		networkId := state.NetworkId(m.networkList.Index())
		if networkId == nil && m.networkList.Index() != networklist.SignalsIndex {
			m.networkList.SetIndex(m.networkList.Index() - 1)
			m.frequencyList.SetNetworkIndex(m.networkList.Index())
			m.memberList.SetNetworkAndFrequency(m.networkList.Index(), m.frequencyList.Index())
			m.chat.SetFrequency(m.networkList.Index(), m.frequencyList.Index())
		}

	case *packet.MembersInfo:
		state.UpdateMembers(msg)
		networkId := state.NetworkId(m.networkList.Index())
		if networkId != nil {
			members := state.State.Members[*networkId]
			index := m.memberList.Index()
			if index >= len(members) {
				m.memberList.SetIndex(m.memberList.Index())
			}
		}

	case *packet.FrequenciesInfo:
		state.UpdateFrequencies(msg)
		networkId := state.NetworkId(m.networkList.Index())
		if networkId == nil {
			return nil
		}
		index := m.frequencyList.Index()
		length := len(state.State.Frequencies[*networkId])
		if index >= length {
			m.frequencyList.SetIndex(index - 1)
			m.memberList.SetNetworkAndFrequency(m.networkList.Index(), m.frequencyList.Index())
			m.chat.SetFrequency(m.networkList.Index(), m.frequencyList.Index())
		}

	case *packet.MessagesInfo:
		if len(msg.Messages) == 1 {
			m.chat.OnNewMessageReceived(msg) // MUST BE BEFORE STATE UPDATE
		}
		state.UpdateMessages(msg)

	case *packet.TrustInfo:
		state.UpdateTrustedUsers(msg)

	case *packet.BlockInfo:
		state.UpdateBlockedUsers(msg)

	case *packet.UsersInfo:
		state.UpdateUsersInfo(msg)

	case *packet.NotificationsInfo:
		signals := state.UpdateNotifications(msg)
		if m.networkList.Index() == networklist.SignalsIndex {
			m.chat.SetReceiver(-1)
			state.Data.Signals = slices.Insert(state.Data.Signals, 0, signals...)
			return m.chat.SetReceiver(m.signalList.Index())
		} else {
			state.Data.Signals = slices.Insert(state.Data.Signals, 0, signals...)
		}

	case ui.BanReasonPopupMsg:
		popup := banreason.New(msg.User, msg.Network)
		m.banReasonPopup = &popup

	case ui.BanViewPopupMsg:
		popup := banview.New(msg.User, msg.Network)
		m.banViewPopup = &popup

	case ui.ProfilePopupMsg:
		popup := profile.New(msg.User)
		m.profilePopup = &popup

	case tea.KeyMsg:
		switch msg.String() {
		case "n":
			switch m.focus {
			case FocusNetworkList:
				if !m.HasPopup() {
					popup := networkcreation.New()
					m.networkCreationPopup = &popup
					message = ui.EmptyMsg{}
				}
			case FocusLeftSidebar:
				IsFrequenciesSidebar := m.networkList.Index() != networklist.SignalsIndex
				if !m.HasPopup() && IsFrequenciesSidebar {
					networkId := state.NetworkId(m.networkList.Index())
					if networkId == nil {
						return nil
					}
					member := state.State.Members[*networkId][*state.UserID]
					if !member.IsAdmin {
						return nil
					}
					popup := frequencycreation.New(*networkId)
					m.frequencyCreationPopup = &popup
					message = ui.EmptyMsg{}
				}
			}

		case "a":
			if !m.HasPopup() {
				if m.focus == FocusNetworkList {
					popup := networkjoin.New()
					m.networkJoinPopup = &popup
					message = ui.EmptyMsg{}
				}
				IsSignalsSidebar := m.networkList.Index() == networklist.SignalsIndex
				if m.focus == FocusLeftSidebar && IsSignalsSidebar {
					popup := signaladd.New()
					m.signalAddPopup = &popup
					message = ui.EmptyMsg{}
				}
			}

		case "i":
			index := m.networkList.Index()
			isNetworkListFocused := m.focus == FocusNetworkList
			isFrequenciesSidebar := index != networklist.SignalsIndex
			if !m.HasPopup() && isNetworkListFocused && isFrequenciesSidebar {
				networkId := state.NetworkId(index)
				if networkId != nil {
					_ = clipboard.WriteAll(networkId.String())
				}
				return nil
			}

		case "e":
			index := m.networkList.Index()
			networkFocus := m.focus == FocusNetworkList
			frequencyFocus := m.focus == FocusLeftSidebar
			if !m.HasPopup() && networkFocus && index != networklist.SignalsIndex {
				networkId := state.NetworkId(index)
				if networkId == nil {
					return nil
				}
				if state.State.Networks[*networkId].OwnerID != *state.UserID {
					return nil
				}
				popup := networkupdate.New(*networkId)
				m.networkUpdatePopup = &popup
				message = ui.EmptyMsg{}
			} else if !m.HasPopup() && frequencyFocus {
				networkId := state.NetworkId(index)
				if networkId == nil {
					return nil
				}
				member := state.State.Members[*networkId][*state.UserID]
				if !member.IsAdmin {
					return nil
				}
				index := m.frequencyList.Index()
				popup := frequencyupdate.New(*networkId, index)
				m.frequencyUpdatePopup = &popup
				message = ui.EmptyMsg{}
			}

		// user [s]ettings
		case "s":
			isChatLocked := m.focus == FocusChat && m.chat.Locked()
			if !m.HasPopup() && !isChatLocked {
				popup := usersettings.New()
				m.userSettingsPopup = &popup
				message = ui.EmptyMsg{}
			}

		case "?":
			normalMode := m.chat.Mode() == viminput.NormalMode
			if !m.HasPopup() && (m.focus != FocusChat || !m.chat.Locked() || normalMode) {
				switch m.focus {
				case FocusNetworkList:
					m.helpPopup = NewHelpPopup(HelpNetworkList)
				case FocusLeftSidebar:
					if m.networkList.Index() == networklist.SignalsIndex {
						m.helpPopup = NewHelpPopup(HelpSignalList)
					} else {
						m.helpPopup = NewHelpPopup(HelpFrequencyList)
					}
				case FocusChat:
					if m.chat.Locked() {
						m.helpPopup = NewHelpPopup(HelpVim)
					} else {
						m.helpPopup = NewHelpPopup(HelpChat)
					}
				case FocusRightSidebar:
					if m.memberList.IsBanList() {
						m.helpPopup = NewHelpPopup(HelpBanList)
					} else {
						m.helpPopup = NewHelpPopup(HelpMemberList)
					}
				}
			}

		case "esc":
			if m.HasPopup() {
				m.helpPopup = nil
				m.userSettingsPopup = nil
				m.networkCreationPopup = nil
				m.networkUpdatePopup = nil
				m.frequencyCreationPopup = nil
				m.frequencyUpdatePopup = nil
				m.networkJoinPopup = nil
				m.banReasonPopup = nil
				m.banViewPopup = nil
				m.signalAddPopup = nil
				m.profilePopup = nil
			}

		case "enter":
			if m.helpPopup != nil {
				m.helpPopup = nil
			} else if m.userSettingsPopup != nil {
				cmd := m.userSettingsPopup.Select()
				if cmd != nil {
					m.userSettingsPopup = nil
				}
				return cmd
			} else if m.networkCreationPopup != nil {
				cmd := m.networkCreationPopup.Select()
				if cmd != nil {
					m.networkCreationPopup = nil
				}
				return cmd
			} else if m.networkUpdatePopup != nil {
				cmd := m.networkUpdatePopup.Select()
				if cmd != nil {
					m.networkUpdatePopup = nil
				}
				return cmd
			} else if m.frequencyCreationPopup != nil {
				cmd := m.frequencyCreationPopup.Select()
				if cmd != nil {
					m.frequencyCreationPopup = nil
				}
				return cmd
			} else if m.frequencyUpdatePopup != nil {
				cmd := m.frequencyUpdatePopup.Select()
				if cmd != nil {
					m.frequencyUpdatePopup = nil
				}
				return cmd
			} else if m.networkJoinPopup != nil {
				cmd := m.networkJoinPopup.Select()
				if cmd != nil {
					m.networkJoinPopup = nil
				}
				return cmd
			} else if m.banReasonPopup != nil {
				cmd := m.banReasonPopup.Select()
				if cmd != nil {
					m.banReasonPopup = nil
				}
				return cmd
			} else if m.banViewPopup != nil {
				m.banViewPopup = nil
			} else if m.signalAddPopup != nil {
				cmd, i := m.signalAddPopup.Select()
				if i != -1 {
					m.signalAddPopup = nil
					m.signalList.SetIndex(i)
				}
				return cmd
			} else if m.profilePopup != nil {
				m.profilePopup = nil
			}

		default:
			isChatLocked := m.focus == FocusChat && m.chat.Locked()
			if !m.HasPopup() && !isChatLocked {
				left := msg.String() == "H"
				right := msg.String() == "L"
				direction := 0
				if left {
					direction = -1
				} else if right {
					direction = 1
				}
				m.move(direction)
			}
		}
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd

	if m.HasPopup() {
		cmd := m.updatePopups(message)
		cmds = append(cmds, cmd)
		message = ui.EmptyMsg{}
		colors.Darken()
	}

	m.networkList, cmd = m.networkList.Update(message)
	cmds = append(cmds, cmd)

	if m.networkList.Index() == networklist.SignalsIndex {
		m.signalList, cmd = m.signalList.Update(message)
		cmds = append(cmds, cmd)

		cmd = m.chat.SetReceiver(m.signalList.Index())
		cmds = append(cmds, cmd)
	} else {
		m.frequencyList.SetNetworkIndex(m.networkList.Index())
		m.frequencyList, cmd = m.frequencyList.Update(message)
		cmds = append(cmds, cmd)

		cmd = m.chat.SetFrequency(m.networkList.Index(), m.frequencyList.Index())
		cmds = append(cmds, cmd)
	}

	m.memberList.SetNetworkAndFrequency(m.networkList.Index(), m.frequencyList.Index())
	m.memberList, cmd = m.memberList.Update(message)
	cmds = append(cmds, cmd)

	m.chat, cmd = m.chat.Update(message)
	cmds = append(cmds, cmd)

	calculateNotifications()

	if m.HasPopup() {
		colors.Restore()
	}

	return tea.Batch(cmds...)
}

func (m *Model) updateLoadScreenContent() {
	seconds := m.timer.Timeout.Round(time.Second) / time.Second
	m.loading.SetContent(fmt.Sprintf(connectionFailed, seconds))
}

func newTimer(timeout time.Duration) timer.Model {
	return timer.NewWithInterval(timeout.Truncate(time.Second)+(time.Second/2), timerInterval)
}

func (m *Model) move(direction int) {
	focus := m.focus + direction
	m.focus = max(0, min(FocusMax-1, focus))

	switch m.focus {
	case FocusNetworkList:
		m.signalList.Blur()
		m.frequencyList.Blur()
		m.memberList.Blur()
		m.chat.Blur()
		m.networkList.Focus()
	case FocusLeftSidebar:
		m.networkList.Blur()
		m.memberList.Blur()
		m.chat.Blur()
		if m.networkList.Index() == networklist.SignalsIndex {
			m.frequencyList.Blur()
			m.signalList.Focus()
		} else {
			m.signalList.Blur()
			m.frequencyList.Focus()
		}
	case FocusChat:
		m.signalList.Blur()
		m.networkList.Blur()
		m.memberList.Blur()
		m.frequencyList.Blur()
		m.chat.Focus()
	case FocusRightSidebar:
		m.signalList.Blur()
		m.networkList.Blur()
		m.frequencyList.Blur()
		m.chat.Blur()
		m.memberList.Focus()
	default:
		assert.Never("missing switch statement field in move", "focus", m.focus)
	}
}

func (m *Model) updatePopups(msg tea.Msg) tea.Cmd {
	if m.helpPopup != nil {
		popup, cmd := m.helpPopup.Update(msg)
		m.helpPopup = &popup
		return cmd
	} else if m.userSettingsPopup != nil {
		popup, cmd := m.userSettingsPopup.Update(msg)
		m.userSettingsPopup = &popup
		return cmd
	} else if m.networkCreationPopup != nil {
		popup, cmd := m.networkCreationPopup.Update(msg)
		m.networkCreationPopup = &popup
		return cmd
	} else if m.networkUpdatePopup != nil {
		popup, cmd := m.networkUpdatePopup.Update(msg)
		m.networkUpdatePopup = &popup
		return cmd
	} else if m.frequencyCreationPopup != nil {
		popup, cmd := m.frequencyCreationPopup.Update(msg)
		m.frequencyCreationPopup = &popup
		return cmd
	} else if m.frequencyUpdatePopup != nil {
		popup, cmd := m.frequencyUpdatePopup.Update(msg)
		m.frequencyUpdatePopup = &popup
		return cmd
	} else if m.networkJoinPopup != nil {
		popup, cmd := m.networkJoinPopup.Update(msg)
		m.networkJoinPopup = &popup
		return cmd
	} else if m.banReasonPopup != nil {
		popup, cmd := m.banReasonPopup.Update(msg)
		m.banReasonPopup = &popup
		return cmd
	} else if m.banViewPopup != nil {
		popup, cmd := m.banViewPopup.Update(msg)
		m.banViewPopup = &popup
		return cmd
	} else if m.signalAddPopup != nil {
		popup, cmd := m.signalAddPopup.Update(msg)
		m.signalAddPopup = &popup
		return cmd
	} else if m.profilePopup != nil {
		popup, cmd := m.profilePopup.Update(msg)
		m.profilePopup = &popup
		return cmd
	}
	return nil
}

func (m *Model) HasPopup() bool {
	return m.helpPopup != nil ||
		m.userSettingsPopup != nil ||
		m.networkCreationPopup != nil ||
		m.networkUpdatePopup != nil ||
		m.frequencyCreationPopup != nil ||
		m.frequencyUpdatePopup != nil ||
		m.networkJoinPopup != nil ||
		m.banReasonPopup != nil ||
		m.banViewPopup != nil ||
		m.signalAddPopup != nil ||
		m.profilePopup != nil
}

func calculateNotifications() {
	for _, signalId := range state.Data.Signals {
		if _, ok := state.State.Messages[signalId]; !ok {
			continue
		}

		pings := getSignalNotification(signalId)
		state.State.LocalNotifications[signalId] = pings
	}

	for networkId := range state.State.Networks {
		for _, frequency := range state.State.Frequencies[networkId] {
			if _, ok := state.State.Messages[frequency.ID]; !ok {
				continue
			}

			pings, hasNotif := getFrequencyNotification(networkId, frequency.ID)
			if hasNotif {
				state.State.LocalNotifications[frequency.ID] = pings
			} else {
				delete(state.State.LocalNotifications, frequency.ID)
			}
		}
	}
}

func getFrequencyNotification(networkId, frequencyId snowflake.ID) (_ int, _ bool) {
	lastReadMsg := state.State.LastReadMessages[frequencyId]
	if lastReadMsg == nil {
		return 0, false
	}

	btree := state.State.Messages[frequencyId]
	if btree == nil {
		return 0, false
	}

	pings := 0
	hasNotif := false

	isAdmin := state.State.Members[networkId][*state.UserID].IsAdmin

	btree.AscendGreaterOrEqual(data.Message{ID: *lastReadMsg + 1}, func(item data.Message) bool {
		hasNotif = true

		if item.Ping == nil {
			return true
		}

		if *item.Ping == packet.PingEveryone {
			pings++
		} else if *item.Ping == packet.PingAdmins && isAdmin {
			pings++
		} else if *item.Ping == *state.UserID {
			pings++
		}

		// No need to continue if we have 10 pings
		return pings < 10
	})

	return pings, hasNotif
}

func getSignalNotification(signal snowflake.ID) int {
	lastReadMsg := state.State.LastReadMessages[signal]
	if lastReadMsg == nil {
		return 0
	}

	btree := state.State.Messages[signal]
	if btree == nil {
		return 0
	}

	pings := 0

	btree.AscendGreaterOrEqual(data.Message{ID: *lastReadMsg + 1}, func(item data.Message) bool {
		pings++

		// No need to continue if we have 10 pings
		return pings < 10
	})

	return pings
}

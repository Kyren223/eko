package state

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"log"
	"slices"
	"time"

	"github.com/google/btree"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

type ChatState struct {
	IncompleteMessage string
	Base              int
	MaxHeight         int
}

type state struct {
	ChatState     map[snowflake.ID]ChatState    // key is frequency id or receiver id
	LastFrequency map[snowflake.ID]snowflake.ID // key is network id

	Messages      map[snowflake.ID]*btree.BTreeG[data.Message]  // key is frequency id or receiver id
	Networks      map[snowflake.ID]data.Network                 // key is network id
	Frequencies   map[snowflake.ID][]data.Frequency             // key is network id
	Members       map[snowflake.ID]map[snowflake.ID]data.Member // key is network id then user id
	Users         map[snowflake.ID]data.User                    // key is user id
	TrustedUsers  map[snowflake.ID]ed25519.PublicKey            // key is user id
	BlockedUsers  map[snowflake.ID]struct{}                     // key is user id
	BlockingUsers map[snowflake.ID]struct{}                     // key is user id

	LastReadMessages    map[snowflake.ID]*snowflake.ID // key is frequency id or receiver id
	RemoteNotifications map[snowflake.ID]int           // key is frequency id or receiver id
	LocalNotifications  map[snowflake.ID]int           // key is frequency id or receiver id
}

var State state = state{
	ChatState:           map[snowflake.ID]ChatState{},
	LastFrequency:       map[snowflake.ID]snowflake.ID{},
	Messages:            map[snowflake.ID]*btree.BTreeG[data.Message]{},
	Networks:            map[snowflake.ID]data.Network{},
	Frequencies:         map[snowflake.ID][]data.Frequency{},
	Members:             map[snowflake.ID]map[snowflake.ID]data.Member{},
	Users:               map[snowflake.ID]data.User{},
	TrustedUsers:        map[snowflake.ID]ed25519.PublicKey{},
	BlockedUsers:        map[snowflake.ID]struct{}{},
	BlockingUsers:       map[snowflake.ID]struct{}{},
	LastReadMessages:    map[snowflake.ID]*snowflake.ID{},
	RemoteNotifications: map[snowflake.ID]int{},
	LocalNotifications:  map[snowflake.ID]int{},
}

type UserData struct {
	Networks []snowflake.ID
	Signals  []snowflake.ID
}

var Data UserData = UserData{
	Networks: []snowflake.ID{},
	Signals:  []snowflake.ID{},
}

var UserID *snowflake.ID = nil

func UpdateNetworks(info *packet.NetworksInfo) {
	networks := State.Networks

	for _, removedNetworkId := range info.RemovedNetworks {
		delete(networks, removedNetworkId)
		delete(State.Frequencies, removedNetworkId)
		delete(State.Members, removedNetworkId)

		for i, network := range Data.Networks {
			if network == removedNetworkId {
				copy(Data.Networks[i:], Data.Networks[i+1:])
				Data.Networks = Data.Networks[:len(Data.Networks)-1]
				break
			}
		}
	}

	for _, network := range info.Networks {
		if !slices.Contains(Data.Networks, network.ID) {
			Data.Networks = append(Data.Networks, network.ID)
		}

		networks[network.ID] = network.Network

		if info.Partial {
			continue
		}

		State.Frequencies[network.ID] = network.Frequencies

		for _, member := range network.Members {
			if State.Members[network.ID] == nil {
				State.Members[network.ID] = map[snowflake.ID]data.Member{}
			}
			State.Members[network.ID][member.UserID] = member
		}
		for _, user := range network.Users {
			State.Users[user.ID] = user
		}
	}

	// Remove any unrecognized networks
	Data.Networks = slices.DeleteFunc(Data.Networks, func(id snowflake.ID) bool {
		_, ok := State.Networks[id]
		return !ok
	})
}

func UpdateFrequencies(info *packet.FrequenciesInfo) {
	frequencies := State.Frequencies[info.Network]

	frequencies = slices.DeleteFunc(frequencies, func(frequency data.Frequency) bool {
		return slices.Contains(info.RemovedFrequencies, frequency.ID)
	})
	for i, frequency := range frequencies {
		frequency.Position = int64(i)
		frequencies[i] = frequency
	}

	for _, newFrequency := range info.Frequencies {
		position := int(newFrequency.Position)
		if len(frequencies) == position {
			frequencies = append(frequencies, newFrequency)
		} else if position < len(frequencies) {
			frequencies[position] = newFrequency
		}
	}

	State.Frequencies[info.Network] = frequencies
}

func UpdateMessages(info *packet.MessagesInfo) {
	for _, id := range info.RemovedMessages {
		for _, btree := range State.Messages {
			btree.Delete(data.Message{ID: id})
		}
	}

	unknownUsers := []snowflake.ID{}
	for _, message := range info.Messages {
		msgSource := message.FrequencyID
		if msgSource == nil {
			msgSource = message.ReceiverID
			if *message.ReceiverID == *UserID {
				msgSource = &message.SenderID
			}
		}
		bt := State.Messages[*msgSource]
		if bt == nil {
			bt = btree.NewG(2, func(a, b data.Message) bool {
				return a.ID < b.ID
			})
			State.Messages[*msgSource] = bt
		}
		bt.ReplaceOrInsert(message)

		if _, ok := State.Users[message.SenderID]; !ok {
			unknownUsers = append(unknownUsers, message.SenderID)
		}
	}

	gateway.SendAsync(&packet.GetUsers{
		Users: unknownUsers,
	})

	// Note: this is a naive approach
	// Ideally we check each message that was added/removed
	// For the frequency/receiver/sender id and only remove that
	// But it can be very slow when there are thousands of messages
	// TODO: when msg chunking is implemented, consider doing it per-msg
	// TODO: consider checking messages count and for small counts use
	// the per message approach
	for id, state := range State.ChatState {
		state.MaxHeight = -1
		State.ChatState[id] = state
	}
}

func UpdateMembers(info *packet.MembersInfo) {
	for _, member := range info.Members {
		if State.Members[info.Network] == nil {
			State.Members[info.Network] = map[snowflake.ID]data.Member{}
		}
		State.Members[info.Network][member.UserID] = member
	}
	for _, removedMember := range info.RemovedMembers {
		delete(State.Members[info.Network], removedMember)

		if removedMember != *UserID {
			continue
		}
		delete(State.Networks, info.Network)
		delete(State.Frequencies, info.Network)
		delete(State.Members, info.Network)

		for i, network := range Data.Networks {
			if network == info.Network {
				copy(Data.Networks[i:], Data.Networks[i+1:])
				Data.Networks = Data.Networks[:len(Data.Networks)-1]
				break
			}
		}
	}
	for _, user := range info.Users {
		State.Users[user.ID] = user
	}
}

func NetworkId(index int) *snowflake.ID {
	if 0 <= index && index < len(Data.Networks) {
		return &Data.Networks[index]
	}
	return nil
}

func JsonUserData() string {
	bytes, err := json.Marshal(Data)
	assert.NoError(err, "marshling should never fail")
	return string(bytes)
}

func FromJsonUserData(s string) {
	var data UserData
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		return
	}

	// log.Println("Previous user data:", Data)
	if data.Networks != nil {
		Data.Networks = data.Networks
	}
	if data.Signals != nil {
		Data.Signals = data.Signals
	}
	log.Println("Updated user data:", Data)
}

func UpdateTrustedUsers(info *packet.TrustInfo) {
	for _, removed := range info.RemovedTrustedUsers {
		delete(State.TrustedUsers, removed)
	}

	for i, trusted := range info.TrustedUsers {
		State.TrustedUsers[trusted] = info.TrustedPublicKeys[i]
	}
}

func GetLastMessage(id snowflake.ID) *snowflake.ID {
	btree := State.Messages[id]
	if btree == nil {
		return nil
	}
	msg, ok := btree.Max()
	if !ok {
		return nil
	}
	return &msg.ID
}

func IsFrequency(id snowflake.ID) bool {
	// OPTIMIZE: this is very expensive and inefficient
	// A map is better but as most of the time frequencies are iterated
	// over based on a network id, this would add overhead
	// And this function is only used once in notifications
	// Maybe there should be a special bit in the snowflake to determine this?
	// This could work if frequencies, user IDs, network IDs etc would all have
	// their own "pool" to generate from (using the "machine id" bits)

	for _, frequencies := range State.Frequencies {
		for _, frequency := range frequencies {
			if id == frequency.ID {
				return true
			}
		}
	}
	return false
}

func UpdateNotifications(info *packet.NotificationsInfo) []snowflake.ID {
	signals := []snowflake.ID{}

	for i := 0; i < len(info.Source); i++ {
		source := info.Source[i]
		lastRead := snowflake.ID(info.LastRead[i])
		State.LastReadMessages[source] = &lastRead

		ping := info.Pings[i]
		if ping != nil {
			State.RemoteNotifications[source] = int(*ping)

			// When someone messages you, and you don't have a signal with him
			// already, add a signal with him so you see his messages
			// PERF: IsFrequency is expensive so contains is checked first
			if !slices.Contains(Data.Signals, source) && !IsFrequency(source) {
				signals = append(signals, source)
			}
		} else {
			delete(State.RemoteNotifications, info.Source[i])
		}
	}

	return signals
}

func SendFinalData() {
	data := JsonUserData()
	// ch1 := gateway.SendAsync(&packet.SetUserData{
	// 	Data: &data,
	// 	User: nil,
	// })

	sources := []snowflake.ID{}

	sources = append(sources, Data.Signals...)
	for _, frequencies := range State.Frequencies {
		for _, frequency := range frequencies {
			sources = append(sources, frequency.ID)
		}
	}

	lastReads := make([]int64, 0, len(sources))
	for _, source := range sources {
		if lastRead := State.LastReadMessages[source]; lastRead != nil {
			lastReads = append(lastReads, int64(*lastRead))
		} else {
			lastReads = append(lastReads, 0)
		}
	}

	ch1 := gateway.SendAsync(&packet.SetUserData{
		Data: &data,
		User: nil,
	})
	ch2 := gateway.SendAsync(&packet.SetLastReadMessages{
		Source:   sources,
		LastRead: lastReads,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		<-ch1
		<-ch2
		close(done)
	}()

	select {
	case <-ctx.Done():
		log.Println("Timeout while waiting for final writes to complete")
	case <-done:
		log.Println("All final writes completed successfully")
	}

	// HACK: Give a small grace period for the writes to be processed
	// Tweak this value as needed
	time.Sleep(20 * time.Millisecond)

	// TODO:
	// I think the issue is that it's random which of these 2 requests goes
	// first (bcz it only does the first request after the client disconnects)
	// I could fix it server side but eh maybe not
	// this should instead use a method that blocks until a response was
	// received, which may need a new gateway method to do that as currently
	// it just always sends to the prograg
	// If this is tedious enough it might be worth it to just do it on the
	// server side

	// Also later on I should probably remove the calculate notifs in core
	// it can be replaced with just diff-ing incoming notifs
	// this will most likely work fine although there are some issues with
	// scopes like becoming an admin/no longer being admin or gaining
	// or losing access to frequencies and of course msg deletions
	// But it's probably the right approach (also WAYYYYYY faster)

	// Then there is also the issue of when switching to a frequency
	// not yet receiving the history so it says "no keep" bcz it's not loaded
	// but with history it would've said "yes keep" so the solutin would
	// be to rework it quite a bit to make it stateless or smthing
	// Then that should be most issues when it comes to notifications
	// just need to make sure local/remote notifs reset properly when
	// reaching the bottom

	// log.Println("BLOCKING...")
	// <-ctx.Done()
	// log.Println("CTX DANZO")

	// TODO: remove this before release
	// assert.NoError(ctx.Err(), "context has ran out of time!")
}

func UpdateBlockedUsers(info *packet.BlockInfo) {
	for _, removed := range info.RemovedBlockedUsers {
		delete(State.BlockedUsers, removed)
	}

	for _, blocked := range info.BlockedUsers {
		State.BlockedUsers[blocked] = struct{}{}
		delete(State.TrustedUsers, blocked)
	}

	for _, removed := range info.RemovedBlockingUsers {
		delete(State.BlockingUsers, removed)
	}

	for _, blocking := range info.BlockingUsers {
		State.BlockingUsers[blocking] = struct{}{}
	}
}

func UpdateUsersInfo(info *packet.UsersInfo) {
	for _, user := range info.Users {
		State.Users[user.ID] = user
	}
}

func MergedNotification(chatId snowflake.ID) (pings int, ok bool) {
	remotePings, remoteOk := State.RemoteNotifications[chatId]
	localPings, localOk := State.LocalNotifications[chatId]
	return remotePings + localPings, remoteOk || localOk
}

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

	Messages     map[snowflake.ID]*btree.BTreeG[data.Message]  // key is frequency id or receiver id
	Networks     map[snowflake.ID]data.Network                 // key is network id
	Frequencies  map[snowflake.ID][]data.Frequency             // key is network id
	Members      map[snowflake.ID]map[snowflake.ID]data.Member // key is network id then user id
	Users        map[snowflake.ID]data.User                    // key is user id
	TrustedUsers map[snowflake.ID]ed25519.PublicKey            // key is user id

	LastReadMessages map[snowflake.ID]*snowflake.ID // key is frequency id or receiver id
	Notifications    map[snowflake.ID]int           // key is frequency id or receiver id
}

var State state = state{
	ChatState:        map[snowflake.ID]ChatState{},
	LastFrequency:    map[snowflake.ID]snowflake.ID{},
	Messages:         map[snowflake.ID]*btree.BTreeG[data.Message]{},
	Networks:         map[snowflake.ID]data.Network{},
	Frequencies:      map[snowflake.ID][]data.Frequency{},
	Members:          map[snowflake.ID]map[snowflake.ID]data.Member{},
	Users:            map[snowflake.ID]data.User{},
	TrustedUsers:     map[snowflake.ID]ed25519.PublicKey{},
	Notifications:    map[snowflake.ID]int{},
	LastReadMessages: map[snowflake.ID]*snowflake.ID{},
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
	}

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
	// Note this is very expensive and inefficient
	// A map is better but as most of the time frequencies are iterated
	// over based on a network id, this would add overhead
	// And this function is only used once in notifications

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
			// log.Println(source, *ping)
			State.Notifications[source] = int(*ping)
			if !IsFrequency(source) && !slices.Contains(Data.Signals, source) {
				signals = append(signals, source)
				// log.Println("Signals:", Data.Signals)
			}
		} else {
			// log.Println(source, "deleted")
			delete(State.Notifications, info.Source[i])
		}
	}

	return signals
}

func SendFinalData() {
	data := JsonUserData()
	ch1 := gateway.SendAsync(&packet.SetUserData{
		Data: &data,
		User: nil,
	})

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

	ch2 := gateway.SendAsync(&packet.SetLastReadMessages{
		Source:   sources,
		LastRead: lastReads,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
	case <-ch1:
	}

	select {
	case <-ctx.Done():
	case <-ch2:
	}
}

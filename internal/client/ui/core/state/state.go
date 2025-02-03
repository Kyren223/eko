package state

import (
	"crypto/ed25519"
	"encoding/json"
	"slices"

	"github.com/google/btree"
	"github.com/kyren223/eko/internal/client/gateway"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

type FrequencyState struct {
	IncompleteMessage string
	Base              int
	MaxHeight         int
}

type state struct {
	ChatState     map[snowflake.ID]FrequencyState // key is frequency id or receiver id
	LastFrequency map[snowflake.ID]snowflake.ID   // key is network id

	Messages    map[snowflake.ID]*btree.BTreeG[data.Message]  // key is frequency id or receiver id
	Networks    map[snowflake.ID]data.Network                 // key is network id
	Frequencies map[snowflake.ID][]data.Frequency             // key is network id
	Members     map[snowflake.ID]map[snowflake.ID]data.Member // key is network id then user id
	Users       map[snowflake.ID]data.User                    // key is user id
	Trusteds    map[snowflake.ID]ed25519.PublicKey            // key is user id

	Notifications map[snowflake.ID]int // key is frequency id or receiver id
}

var State state = state{
	ChatState:     map[snowflake.ID]FrequencyState{},
	LastFrequency: map[snowflake.ID]snowflake.ID{},
	Messages:      map[snowflake.ID]*btree.BTreeG[data.Message]{},
	Networks:      map[snowflake.ID]data.Network{},
	Frequencies:   map[snowflake.ID][]data.Frequency{},
	Members:       map[snowflake.ID]map[snowflake.ID]data.Member{},
	Users:         map[snowflake.ID]data.User{},
	Trusteds:      map[snowflake.ID]ed25519.PublicKey{},
	Notifications: map[snowflake.ID]int{},
}

type UserData struct {
	LastReadMessage map[snowflake.ID]*snowflake.ID
	Networks        []snowflake.ID
	Peers           []snowflake.ID
}

var Data UserData = UserData{
	LastReadMessage: map[snowflake.ID]*snowflake.ID{},
	Networks:        []snowflake.ID{},
	Peers:           []snowflake.ID{},
}

var UserID *snowflake.ID = nil

var (
	ReceivedData     = false
	ReceivedNetworks = false
	AskedForNotifs   = false
)

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

	data := JsonUserData()
	gateway.SendAsync(&packet.SetUserData{
		Data: &data,
		User: nil,
	})

	ReceivedNetworks = true
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
	Data = data
	if Data.LastReadMessage == nil {
		Data.LastReadMessage = map[snowflake.ID]*snowflake.ID{}
	}

	ReceivedData = true
}

func UpdateTrusteds(info *packet.TrustInfo) {
	for _, removed := range info.RemovedTrusteds {
		delete(State.Trusteds, removed)
	}

	for i, trusted := range info.Trusteds {
		State.Trusteds[trusted] = info.TrustedPublicKeys[i]
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

func SendUserDatUpdate() {
	data := JsonUserData()
	gateway.SendAsync(&packet.SetUserData{
		Data: &data,
		User: nil,
	})
}

func UpdateNotifications(info *packet.NotificationsInfo) {
	for i := 0; i < len(info.Source); i++ {
		ping := info.Pings[i]
		if ping != nil {
			State.Notifications[info.Source[i]] = int(*ping)
		} else {
			delete(State.Notifications, info.Source[i])
		}
	}
}

func AskForNotifs() {
	source := []snowflake.ID{}

	source = append(source, Data.Peers...)
	for networkId := range State.Networks {
		for _, frequency := range State.Frequencies[networkId] {
			source = append(source, frequency.ID)
		}
	}

	lastReadId := make([]snowflake.ID, 0, len(source))

	for _, source := range source {
		id := Data.LastReadMessage[source]
		if id == nil {
			lastReadId = append(lastReadId, 0)
		} else {
			lastReadId = append(lastReadId, *id)
		}
	}

	gateway.SendAsync(&packet.GetNotifications{
		Source:     source,
		LastReadId: lastReadId,
	})
}

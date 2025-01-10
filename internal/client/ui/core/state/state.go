package state

import (
	"encoding/json"
	"slices"

	"github.com/google/btree"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

type FrequencyState struct {
	IncompleteMessage string
	Offset            int
	MaxHeight         int
}

type state struct {
	UserID *snowflake.ID

	// Key is either a frequency or receiver
	FrequencyState map[snowflake.ID]FrequencyState // key is frequency id
	LastFrequency  map[snowflake.ID]snowflake.ID   // key is network id

	Messages    map[snowflake.ID]*btree.BTreeG[data.Message]  // key is frequency id or receiver id
	Networks    map[snowflake.ID]data.Network                 // key is network id
	Frequencies map[snowflake.ID][]data.Frequency             // key is network id
	Members     map[snowflake.ID]map[snowflake.ID]data.Member // key is network id then user id
	Users       map[snowflake.ID]data.User                    // key is user id
}

var State state = state{
	UserID:         nil,
	FrequencyState: map[snowflake.ID]FrequencyState{},
	LastFrequency:  map[snowflake.ID]snowflake.ID{},
	Messages:       map[snowflake.ID]*btree.BTreeG[data.Message]{},
	Networks:       map[snowflake.ID]data.Network{},
	Frequencies:    map[snowflake.ID][]data.Frequency{},
	Members:        map[snowflake.ID]map[snowflake.ID]data.Member{},
	Users:          map[snowflake.ID]data.User{},
}

type UserData struct {
	Networks []snowflake.ID
}

var Data UserData = UserData{
	Networks: []snowflake.ID{},
}

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
		if _, ok := networks[network.ID]; !ok {
			Data.Networks = append(Data.Networks, network.ID)
		}

		networks[network.ID] = network.Network
		State.Frequencies[network.ID] = network.Frequencies
		for _, member := range network.Members {
			State.Members[network.ID][member.UserID] = member
		}

		for _, user := range network.Users {
			State.Users[user.ID] = user
		}
	}
}

func UpdateFrequencies(info *packet.FrequenciesInfo) {
	// TODO: check this
	frequencies := State.Frequencies[info.Network]
	for _, newFrequency := range info.Frequencies {
		add := true
		for i, existingFrequency := range frequencies {
			if existingFrequency.ID == newFrequency.ID {
				add = false
				if newFrequency.Position == -1 {
					newFrequency.Position = existingFrequency.Position
				}
				frequencies[i] = newFrequency
				break
			}
		}
		if add {
			frequencies = append(frequencies, newFrequency)
		}
	}

	frequencies = slices.DeleteFunc(frequencies, func(frequency data.Frequency) bool {
		return slices.Contains(info.RemovedFrequencies, frequency.ID)
	})
	slices.SortFunc(frequencies, func(a, b data.Frequency) int {
		return int(a.Position - b.Position)
	})
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
	members := State.Members[info.Network]
	for _, member := range info.Members {
		members[member.UserID] = member
	}
	for _, removedMember := range info.RemovedMembers {
		delete(members, removedMember)

		if removedMember != *State.UserID {
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

func FromJsonUserDat(s string) {
	var data UserData
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		return
	}
	Data = data
}

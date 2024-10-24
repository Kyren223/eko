package data

import (
	"github.com/kyren223/eko/pkg/snowflake"
	"github.com/kyren223/eko/pkg/utils"
)

// Represents an Eko Network, equivalent to a "discord server".
type Network struct {
	id          snowflake.ID
	name        string
	frequencies []snowflake.ID
}

// Represents an Eko Frequency, equivalent to a "discord channel" within a "discord server".
type Frequency struct {
	id   snowflake.ID
	name string
}

// Represents an Eko Signal, equivalent to a "discord message" between only 2 people.
type Signal struct {
	userId1 snowflake.ID
	userId2 snowflake.ID
}

// Represents a message
type Message struct {
	Id          snowflake.ID
	SenderId    snowflake.ID
	FrequencyId snowflake.ID
	NetworkId   snowflake.ID
	Contents    string
}

func (a Message) CmpTimestamp(b Message) int {
	cmpTime := a.Id.Time() - b.Id.Time()
	if cmpTime != 0 {
		return int(utils.Clamp(cmpTime, -1, 1))
	}
	cmpStep := a.Id.Step() - b.Id.Step()
	if cmpStep != 0 {
		return int(utils.Clamp(cmpStep, -1, 1))
	}
	cmpNode := a.Id.Node() - b.Id.Node()
	if cmpNode != 0 {
		return int(utils.Clamp(cmpNode, -1, 1))
	}
	return 0
}

// Represents an Eko User
type User struct {
	id snowflake.ID
}

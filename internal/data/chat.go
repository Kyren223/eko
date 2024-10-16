package data

import "github.com/kyren223/eko/pkg/snowflake"

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
	Id       snowflake.ID
	SenderId snowflake.ID
	FrequencyId snowflake.ID
	NetworkId snowflake.ID
	Contents string
}

// Represents an Eko User
type User struct {
	id snowflake.ID
}

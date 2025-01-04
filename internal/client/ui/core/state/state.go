package state

import (
	"github.com/google/btree"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/snowflake"
)

type Frequency struct {
	IncompleteMessage string
	Offset            int
	MaxHeight         int
}

type state struct {
	UserID *snowflake.ID

	// Key is either a frequency or receiver
	FrequencyState map[snowflake.ID]Frequency
	LastFrequency  map[snowflake.ID]snowflake.ID // key is network

	Messages map[snowflake.ID]*btree.BTreeG[data.Message]
	Networks []packet.FullNetwork
}

var State state = state{
	UserID:         nil,
	FrequencyState: map[snowflake.ID]Frequency{},
	LastFrequency:  map[snowflake.ID]snowflake.ID{},
	Messages:       map[snowflake.ID]*btree.BTreeG[data.Message]{},
	Networks:       []packet.FullNetwork{},
}

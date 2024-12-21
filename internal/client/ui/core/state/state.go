package state

import (
	"github.com/google/btree"
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/snowflake"
)

type state struct {
	// Key is either a frequency or receiver
	IncompleteMessages map[snowflake.ID]string

	Messages map[snowflake.ID]*btree.BTreeG[data.Message]
	Networks []packet.FullNetwork
}

var State state = state{
	Messages:           map[snowflake.ID]*btree.BTreeG[data.Message]{},
	Networks:           []packet.FullNetwork{},
	IncompleteMessages: map[snowflake.ID]string{},
}

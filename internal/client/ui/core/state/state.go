package state

import (
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/snowflake"
)

type state struct {
	Networks map[snowflake.ID]packet.FullNetwork
}

var State state = state{
	Networks: make(map[snowflake.ID]packet.FullNetwork),
}

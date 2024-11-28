package state

import (
	"github.com/kyren223/eko/internal/packet"
)

type state struct {
	Networks []packet.FullNetwork
}

var State state = state{}

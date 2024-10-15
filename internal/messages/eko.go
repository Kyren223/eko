package messages

import "github.com/kyren223/eko/internal/packet"

type EkoMessage struct {
	Message string `msgpack:"message"`
}

func (m EkoMessage) Type() packet.PacketType {
	return packet.TypeEko
}

type ErrorMessage struct {
	Error string `msgpack:"error"`
}

func (m ErrorMessage) Type() packet.PacketType {
	return packet.TypeError
}

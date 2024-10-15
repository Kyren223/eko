package packet

type EkoMessage struct {
	Message string `msgpack:"message"`
}

func (m *EkoMessage) Type() PacketType {
	return TypeEko
}

type ErrorMessage struct {
	Error string `msgpack:"error"`
}

func (m *ErrorMessage) Type() PacketType {
	return TypeError
}

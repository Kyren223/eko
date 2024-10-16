package packet

import (
	"github.com/kyren223/eko/internal/data"
)

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

type GetMessagesMessage struct {
	Since *int64
	UpTo *int64
}

func (m *GetMessagesMessage) Type() PacketType {
	return TypeGetMessages
}

type SendMessageMessage struct {
	Content string
}

func (m *SendMessageMessage) Type() PacketType {
	return TypeSendMessage
}

type MessagesMessage struct {
	Messages []data.Message
}

func (m *MessagesMessage) Type() PacketType {
	return TypeMessages
}

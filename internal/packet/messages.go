package packet

import (
	"github.com/kyren223/eko/internal/data"
)

type ErrorMessage struct {
	Error string `msgpack:"error"`
}

func NewOkMessage() *ErrorMessage {
	return &ErrorMessage{}
}

func (m *ErrorMessage) Type() PacketType {
	return PacketError
}

func (m *ErrorMessage) IsOk() bool {
	return m.Error == ""
}

type SendMessage struct {
	Content string
}

func (m *SendMessage) Type() PacketType {
	return PacketSendMessage
}

type Messages struct {
	Messages []data.Message
}

func (m *Messages) Type() PacketType {
	return PacketMessages
}

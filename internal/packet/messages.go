package packet

import (
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/pkg/snowflake"
)

type ErrorMessage struct {
	Error string `msgpack:"error"`
}

func (m *ErrorMessage) Type() PacketType {
	return PacketError
}

type SendMessage struct {
	ReceiverID  *snowflake.ID
	FrequencyID *snowflake.ID
	Content     string
}

func (m *SendMessage) Type() PacketType {
	return PacketSendMessage
}

type PushedMessages struct {
	Messages []data.Message
}

func (m *PushedMessages) Type() PacketType {
	return PacketPushedMessages
}

type Messages struct {
	Messages []data.Message
}

func (m *Messages) Type() PacketType {
	return PacketMessages
}

type GetMessagesRange struct {
	FrequencyID *snowflake.ID
	ReceiverID  *snowflake.ID
	From        *int64
	To          *int64
}

func (m *GetMessagesRange) Type() PacketType {
	return PacketGetMessageRange
}

type GetUserByID struct {
	UserID snowflake.ID
}

func (m *GetUserByID) Type() PacketType {
	return PacketGetUserById
}

type Users struct {
	Users []data.User
}

func (m *Users) Type() PacketType {
	return PacketUsers
}

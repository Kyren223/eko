package packet

import (
	"encoding/json"

	"github.com/kyren223/eko/pkg/assert"
	"github.com/vmihailenco/msgpack/v5"
)

type TypedMessage interface {
	Type() PacketType
}

type defaultPacketEncoder struct {
	data       []byte
	encoding   Encoding
	packetType PacketType
}

func (e defaultPacketEncoder) Encoding() Encoding {
	return e.encoding
}

func (e defaultPacketEncoder) Type() PacketType {
	return e.packetType
}

func (e defaultPacketEncoder) Payload() []byte {
	return e.data
}

func NewJsonEncoder(message TypedMessage) PacketEncoder {
	data, err := json.Marshal(message)
	assert.NoError(err, "encoding a message with JSON should never fail")

	return defaultPacketEncoder{
		data:       data,
		encoding:   EncodingJson,
		packetType: message.Type(),
	}
}

func NewMsgPackEncoder(message TypedMessage) PacketEncoder {
	data, err := msgpack.Marshal(message)
	assert.NoError(err, "encoding a message with msg pack should never fail")

	return defaultPacketEncoder{
		data:       data,
		encoding:   EncodingMsgPack,
		packetType: message.Type(),
	}
}

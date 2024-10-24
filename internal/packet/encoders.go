package packet

import (
	"encoding/json"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/kyren223/eko/pkg/assert"
)

type Payload interface {
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

func NewJsonEncoder(payload Payload) PacketEncoder {
	data, err := json.Marshal(payload)
	assert.NoError(err, "encoding a message with JSON should never fail")

	return defaultPacketEncoder{
		data:       data,
		encoding:   EncodingJson,
		packetType: payload.Type(),
	}
}

func NewMsgPackEncoder(payload Payload) PacketEncoder {
	data, err := msgpack.Marshal(payload)
	assert.NoError(err, "encoding a message with msg pack should never fail")

	return defaultPacketEncoder{
		data:       data,
		encoding:   EncodingMsgPack,
		packetType: payload.Type(),
	}
}

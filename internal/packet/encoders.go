package packet

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/vmihailenco/msgpack/v5"
)

type TypedMessage interface{
	Type() PacketType
}

type defaultPacketEncoder struct {
	io.Reader
	encoding Encoding
	packetType PacketType
}

func (e defaultPacketEncoder) Encoding() Encoding {
	return e.encoding
}

func (e defaultPacketEncoder) Type() PacketType {
	return e.packetType
}

func NewJsonEncoder(message TypedMessage) (PacketEncoder, error) {
	data, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	return defaultPacketEncoder{
		Reader:     bytes.NewReader(data),
		encoding:   EncodingJson,
		packetType: message.Type(),
	}, nil
}

func NewMsgPackEncoder(message TypedMessage) (PacketEncoder, error) {
	data, err := msgpack.Marshal(message)
	if err != nil {
		return nil, err
	}

	return defaultPacketEncoder{
		Reader:     bytes.NewReader(data),
		encoding:   EncodingJson,
		packetType: message.Type(),
	}, nil

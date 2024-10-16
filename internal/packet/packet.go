package packet

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/kyren223/eko/pkg/assert"
)

type Encoding uint8

func (e Encoding) String() string {
	switch e {
	case EncodingJson:
		return "EncodingJson"
	case EncodingMsgPack:
		return "EncodingMsgPack"
	case EncodingUnused1:
		return "EncodingUnused1"
	case EncodingUnused2:
		return "EncodingUnused2"
	default:
		return fmt.Sprintf("EncodingInvalid(%v)", byte(e))
	}
}

func (e Encoding) IsSupported() bool {
	switch e {
	case EncodingJson, EncodingMsgPack:
		return true
	default:
		return false
	}
}

const (
	EncodingJson Encoding = iota
	EncodingMsgPack
	EncodingUnused1
	EncodingUnused2
)

type PacketType uint8

func (t PacketType) String() string {
	switch t {
	case TypeEko:
		return "PacketTypeEko"
	case TypeError:
		return "PacketTypeError"
	case TypeGetMessages:
		return "PacketTypeGetMessages"
	case TypeSendMessage:
		return "PacketTypeSendMessage"
	case TypeMessages:
		return "PacketTypeMessages"
	default:
		return fmt.Sprintf("PacketTypeInvalid(%v)", byte(t))
	}
}

func (e PacketType) IsSupported() bool {
	switch e {
	case TypeEko, TypeError, TypeGetMessages, TypeSendMessage, TypeMessages:
		return true
	default:
		return false
	}
}

const (
	TypeEko PacketType = iota
	TypeError
	TypeGetMessages
	TypeSendMessage
	TypeMessages
)

const (
	VERSION          = byte(1)
	PACKET_MAX_SIZE  = ^uint16(0)
	PAYLOAD_MAX_SIZE = PACKET_MAX_SIZE - HEADER_SIZE
	HEADER_SIZE      = 4
	VERSION_OFFSET   = 0
	TYPE_OFFSET      = 1
	ENCODING_OFFSET  = 1
	LENGTH_OFFSET    = 2
)

type PacketEncoder interface {
	io.Reader
	Encoding() Encoding
	Type() PacketType
}

// The following diagram shows the packet structure:
//
//	 0                   1                   2                   3
//	 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|    Version    |En.|    Type   |         Payload Length        |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|              Payload... Payload Length bytes ...              |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
type Packet struct {
	data []byte
}

func NewPacket(encoder PacketEncoder) Packet {
	data := make([]byte, PACKET_MAX_SIZE)

	n, err := encoder.Read(data[HEADER_SIZE:])
	assert.NoError(err, "packet encoder should never error when reading")

	binary.BigEndian.PutUint16(data[LENGTH_OFFSET:], uint16(n))

	data[VERSION_OFFSET] = VERSION

	packetType, encoding := byte(encoder.Type()), byte(encoder.Encoding())
	assert.Assert(packetType <= 63, "packet type exceeded allowed size type=%v", packetType)
	assert.Assert(encoding <= 3, "encoding exceeded allowed permutations encoding=%v", encoding)
	data[TYPE_OFFSET] = packetType | encoding<<6

	return Packet{data[:HEADER_SIZE+n]}
}

func (p Packet) Version() uint8 {
	return p.data[VERSION_OFFSET]
}

func (p Packet) Type() PacketType {
	return PacketType(p.data[TYPE_OFFSET] & 63) // 2^6-1
}

func (p Packet) Encoding() Encoding {
	return Encoding(p.data[ENCODING_OFFSET] >> 6)
}

func (p Packet) PayloadLength() uint16 {
	return binary.BigEndian.Uint16(p.data[LENGTH_OFFSET:])
}

func (p Packet) String() string {
	return fmt.Sprintf("{v%v %v %v %v: %v}", p.data[0], p.Encoding().String(), p.Type().String(), p.PayloadLength(), p.Payload())
}

// The payload data, caller must not modify the returned slice, even temporarily
func (p Packet) Payload() []byte {
	return p.data[HEADER_SIZE:]
}

func (p Packet) DecodePayload(v TypedMessage) error {
	if p.Type() != v.Type() {
		return fmt.Errorf("type mismatch: want %v got %v", p.Type(), v.Type())
	}
	switch p.Encoding() {
	case EncodingJson:
		return json.Unmarshal(p.Payload(), v)
	case EncodingMsgPack:
		return msgpack.Unmarshal(p.Payload(), v)
	case EncodingUnused1:
		fallthrough
	case EncodingUnused2:
		return fmt.Errorf("unsupported encoding: %v", p.Encoding().String())
	default:
		assert.Unreachable("encoding from packet should always be valid encoding=%v", p.Encoding())
		return nil
	}
}

func (p Packet) Into(writer io.Writer) error {
	_, err := writer.Write(p.data[:len(p.data)])
	return err
}

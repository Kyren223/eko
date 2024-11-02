package packet

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/kyren223/eko/pkg/assert"
)

type Encoding uint8

const (
	EncodingJson Encoding = iota
	EncodingMsgPack
	EncodingUnused1
	EncodingUnused2
)

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

type PacketType uint8

const (
	PacketError PacketType = iota
	PacketSendMessage
	PacketPushedMessages
	PacketGetMessageRange
	PacketMessages
)

func (t PacketType) String() string {
	switch t {
	case PacketError:
		return "PacketError"
	case PacketSendMessage:
		return "PacketSendMessage"
	case PacketPushedMessages:
		return "PacketPushedMessages"
	case PacketGetMessageRange:
		return "PacketGetMessageRange"
	case PacketMessages:
		return "PacketMessages"
	default:
		return fmt.Sprintf("PacketInvalidType(%v)", byte(t))
	}
}

func (e PacketType) IsSupported() bool {
	switch e {
	case PacketError, PacketSendMessage, PacketPushedMessages, PacketGetMessageRange, PacketMessages:
		return true
	default:
		return false
	}
}

// True for all packets that a server may push passively to the client.
func (e PacketType) IsPush() bool {
	switch e {
	case PacketError, PacketSendMessage, PacketGetMessageRange, PacketMessages:
		return false
	case PacketPushedMessages:
		return true
	default:
		assert.Never("should never happen")
		return false
	}
}

const (
	VERSION          = byte(1)
	PACKET_MAX_SIZE  = math.MaxUint16
	PAYLOAD_MAX_SIZE = PACKET_MAX_SIZE - HEADER_SIZE
	HEADER_SIZE      = 4
	VERSION_OFFSET   = 0
	TYPE_OFFSET      = 1
	ENCODING_OFFSET  = 1
	LENGTH_OFFSET    = 2
)

type PacketEncoder interface {
	Payload() []byte
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
	payload := encoder.Payload()
	n := len(payload)
	assert.Assert(0 <= n && n <= PAYLOAD_MAX_SIZE, "size of payload must be valid", "size", n)

	data := make([]byte, HEADER_SIZE+n)

	data[VERSION_OFFSET] = VERSION

	packetType, encoding := byte(encoder.Type()), byte(encoder.Encoding())
	assert.Assert(packetType <= 63, "packet type exceeded allowed size", "type", packetType)
	assert.Assert(encoding <= 3, "encoding exceeded allowed size", "encoding", encoding)
	data[TYPE_OFFSET] = packetType | encoding<<6

	binary.BigEndian.PutUint16(data[LENGTH_OFFSET:], uint16(n))

	copy(data[HEADER_SIZE:], payload)

	return Packet{data}
}

func (p Packet) Version() uint8 {
	return p.data[VERSION_OFFSET]
}

func (p Packet) Type() PacketType {
	return PacketType(p.data[TYPE_OFFSET] & 63)
}

func (p Packet) Encoding() Encoding {
	return Encoding(p.data[ENCODING_OFFSET] >> 6)
}

func (p Packet) PayloadLength() uint16 {
	return binary.BigEndian.Uint16(p.data[LENGTH_OFFSET:])
}

func (p Packet) Payload() []byte {
	return p.data[HEADER_SIZE:]
}

func (p Packet) String() string {
	return fmt.Sprintf("Packet(v%v %v %v [%v bytes...])", p.Version(), p.Encoding().String(), p.Type().String(), p.PayloadLength())
}

func (p Packet) Into(writer io.Writer) (int, error) {
	return writer.Write(p.data)
}

func (p Packet) DecodePayloadInto(v Payload) error {
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
		assert.Never("encoding from packet should always be valid", "encoding", p.Encoding())
		return nil
	}
}

func (p Packet) DecodedPayload() (Payload, error) {
	var payload Payload
	switch p.Type() {
	case PacketError:
		payload = &ErrorMessage{}
	case PacketPushedMessages:
		payload = &PushedMessages{}
	case PacketSendMessage:
		payload = &SendMessage{}
	case PacketGetMessageRange:
		payload = &GetMessagesRange{}
	case PacketMessages:
		payload = &Messages{}
	default:
		assert.Never("packet type of a packet struct must always be valid")
	}
	err := p.DecodePayloadInto(payload)
	return payload, err
}

var (
	PacketUnsupportedVersion  error = errors.New("packet error: unsupported version")
	PacketUnsupportedEncoding error = errors.New("packet error: unsupported encoding")
	PacketUnsupportedType     error = errors.New("packet error: unsupported type")
)

type PacketFramer struct {
	buffer []byte
	Out    chan Packet
}

func NewFramer() PacketFramer {
	return PacketFramer{
		Out: make(chan Packet, 10),
	}
}

func (f *PacketFramer) Push(ctx context.Context, data []byte) error {
	f.buffer = append(f.buffer, data...)

	for {
		packet, err := f.parse()
		if packet == nil || err != nil {
			return err
		}
		select {
		case f.Out <- *packet:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (f *PacketFramer) parse() (*Packet, error) {
	if len(f.buffer) < HEADER_SIZE {
		return nil, nil
	}

	if f.buffer[VERSION_OFFSET] != VERSION {
		return nil, PacketUnsupportedVersion
	}

	encoding := Encoding(f.buffer[ENCODING_OFFSET] >> 6)
	if !encoding.IsSupported() {
		return nil, PacketUnsupportedEncoding
	}

	packetType := PacketType(f.buffer[TYPE_OFFSET] & 63)
	if !packetType.IsSupported() {
		return nil, PacketUnsupportedType
	}

	length := binary.BigEndian.Uint16(f.buffer[LENGTH_OFFSET:])
	if len(f.buffer)-HEADER_SIZE < int(length) {
		// Wait for more data to arrive
		return nil, nil
	}

	fullLength := HEADER_SIZE + length
	packetBuffer := make([]byte, fullLength)
	copy(packetBuffer, f.buffer[:fullLength])
	copy(f.buffer, f.buffer[fullLength:])
	f.buffer = f.buffer[:len(f.buffer)-int(fullLength)]

	return &Packet{packetBuffer}, nil
}

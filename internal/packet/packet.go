package packet

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
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

	PacketTosInfo
	PacketAcceptTos

	PacketGetNonce
	PacketNonceInfo

	PacketAuthenticate

	PacketUsersInfo
	PacketGetUsers

	PacketSetUserData
	PacketGetUserData

	PacketCreateNetwork
	PacketUpdateNetwork
	PacketTransferNetwork
	PacketDeleteNetwork
	PacketNetworksInfo

	PacketCreateFrequency
	PacketUpdateFrequency
	PacketDeleteFrequency
	PacketSwapFrequencies
	PacketFrequenciesInfo

	PacketSendMessage
	PacketEditMessage
	PacketDeleteMessage
	PacketRequestMessages
	PacketMessagesInfo

	PacketGetBannedMembers
	PacketSetMember
	PacketMembersInfo

	PacketTrustUser
	PacketTrustInfo

	PacketSetLastReadMessages
	PacketNotificationsInfo

	PacketBlockUser
	PacketBlockInfo

	PacketMax
)

var packetNames = map[PacketType]string{
	PacketError: "PacketError",

	PacketTosInfo:   "PacketTosInfo",
	PacketAcceptTos: "PacketAcceptTos",

	PacketGetNonce:  "PacketGetNonce",
	PacketNonceInfo: "PacketNonceInfo",

	PacketAuthenticate: "PacketAuthenticate",

	PacketSetUserData: "PacketSetUserData",
	PacketGetUserData: "PacketGetUserData",

	PacketCreateNetwork:   "PacketCreateNetwork",
	PacketUpdateNetwork:   "PacketUpdateNetwork",
	PacketTransferNetwork: "PacketTransferNetwork",
	PacketDeleteNetwork:   "PacketDeleteNetwork",
	PacketNetworksInfo:    "PacketNetworksInfo",

	PacketCreateFrequency: "PacketCreateFrequency",
	PacketUpdateFrequency: "PacketUpdateFrequency",
	PacketDeleteFrequency: "PacketDeleteFrequency",
	PacketSwapFrequencies: "PacketSwapFrequencies",
	PacketFrequenciesInfo: "PacketFrequenciesInfo",

	PacketSendMessage:     "PacketSendMessage",
	PacketEditMessage:     "PacketEditMessage",
	PacketDeleteMessage:   "PacketDeleteMessage",
	PacketRequestMessages: "PacketRequestMessages",
	PacketMessagesInfo:    "PacketMessagesInfo",

	PacketGetBannedMembers: "PacketGetBannedMembers",
	PacketSetMember:        "PacketSetMember",
	PacketMembersInfo:      "PacketMembersInfo",

	PacketTrustUser: "PacketTrustUser",
	PacketTrustInfo: "PacketTrustInfo",

	PacketSetLastReadMessages: "PacketSetLastReadMessages",
	PacketNotificationsInfo:   "PacketNotificationsInfo",

	PacketBlockUser: "PacketBlockUser",
	PacketBlockInfo: "PacketBlockInfo",

	PacketGetUsers:  "PacketGetUsers",
	PacketUsersInfo: "PacketUsersInfo",
}

func init() {
	assert.Assert(len(packetNames) == int(PacketMax), "packetName length mismatches with PacketMax", "len(packetNames)", len(packetNames), "PacketMax", int(PacketMax))
	assert.Assert(PacketMax <= 64, "packet types exceeded allowed limit of 64 types")
}

func (e PacketType) IsSupported() bool {
	return e < PacketMax
}

func (e PacketType) String() string {
	if !e.IsSupported() {
		return fmt.Sprintf("UnsupportedPacket(%d)", e)
	}
	return packetNames[e]
}

const (
	VERSION          = byte(2)
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
	n := uint(len(payload))
	assert.Assert(n <= PAYLOAD_MAX_SIZE, "size of payload must be valid", "size", n)

	data := make([]byte, HEADER_SIZE+n)

	data[VERSION_OFFSET] = VERSION

	packetType, encoding := byte(encoder.Type()), byte(encoder.Encoding())
	assert.Assert(packetType <= 63, "packet type exceeded allowed size", "type", packetType)
	assert.Assert(encoding <= 3, "encoding exceeded allowed size", "encoding", encoding)
	data[TYPE_OFFSET] = packetType | encoding<<6

	binary.BigEndian.PutUint16(data[LENGTH_OFFSET:], uint16(n)) // #nosec G115

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
	assert.Never("oops, found someone who uses packet.String(), improve this")
	return fmt.Sprintf("Packet(v%v t%v %v [%v bytes...])", p.Version(), p.Encoding().String(), p.Type(), p.PayloadLength())
}

func (p Packet) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("version", int(p.Version())),
		slog.String("encoding", p.Encoding().String()),
		slog.String("type", p.Type().String()),
		slog.Int("payload_length", int(p.PayloadLength())),
		slog.Int("total_bytes", len(p.data)),
	)
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
		payload = &Error{}

	case PacketTosInfo:
		payload = &TosInfo{}
	case PacketAcceptTos:
		payload = &AcceptTos{}

	case PacketGetNonce:
		payload = &GetNonce{}
	case PacketNonceInfo:
		payload = &NonceInfo{}

	case PacketAuthenticate:
		payload = &Authenticate{}

	case PacketSetUserData:
		payload = &SetUserData{}
	case PacketGetUserData:
		payload = &GetUserData{}

	case PacketCreateNetwork:
		payload = &CreateNetwork{}
	case PacketUpdateNetwork:
		payload = &UpdateNetwork{}
	case PacketTransferNetwork:
		payload = &TransferNetwork{}
	case PacketDeleteNetwork:
		payload = &DeleteNetwork{}
	case PacketNetworksInfo:
		payload = &NetworksInfo{}

	case PacketCreateFrequency:
		payload = &CreateFrequency{}
	case PacketUpdateFrequency:
		payload = &UpdateFrequency{}
	case PacketDeleteFrequency:
		payload = &DeleteFrequency{}
	case PacketSwapFrequencies:
		payload = &SwapFrequencies{}
	case PacketFrequenciesInfo:
		payload = &FrequenciesInfo{}

	case PacketSendMessage:
		payload = &SendMessage{}
	case PacketEditMessage:
		payload = &EditMessage{}
	case PacketDeleteMessage:
		payload = &DeleteMessage{}
	case PacketRequestMessages:
		payload = &RequestMessages{}
	case PacketMessagesInfo:
		payload = &MessagesInfo{}

	case PacketGetBannedMembers:
		payload = &GetBannedMembers{}
	case PacketSetMember:
		payload = &SetMember{}
	case PacketMembersInfo:
		payload = &MembersInfo{}

	case PacketTrustUser:
		payload = &TrustUser{}
	case PacketTrustInfo:
		payload = &TrustInfo{}

	case PacketSetLastReadMessages:
		payload = &SetLastReadMessages{}
	case PacketNotificationsInfo:
		payload = &NotificationsInfo{}

	case PacketBlockUser:
		payload = &BlockUser{}
	case PacketBlockInfo:
		payload = &BlockInfo{}

	case PacketGetUsers:
		payload = &GetUsers{}
	case PacketUsersInfo:
		payload = &UsersInfo{}

	default:
		assert.Assert(!p.Type().IsSupported(), "supported PackeType wasn't handled", "type", p.Type())
		return nil, fmt.Errorf("unsupported PackeType: %v", p.Type().String())
	}
	err := p.DecodePayloadInto(payload)
	return payload, err
}

var (
	ErrUnsupportedVersion  error = errors.New("packet error: unsupported version")
	ErrUnsupportedEncoding error = errors.New("packet error: unsupported encoding")
	ErrUnsupportedType     error = errors.New("packet error: unsupported type")
)

type PacketFramer struct {
	Out    chan Packet
	buffer []byte
}

const ReadQueueSize = 10

func NewFramer() PacketFramer {
	return PacketFramer{
		Out: make(chan Packet, ReadQueueSize),
	}
}

func (f *PacketFramer) Push(ctx context.Context, data []byte) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	f.buffer = append(f.buffer, data...)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

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
		return nil, ErrUnsupportedVersion
	}

	encoding := Encoding(f.buffer[ENCODING_OFFSET] >> 6)
	if !encoding.IsSupported() {
		return nil, ErrUnsupportedEncoding
	}

	packetType := PacketType(f.buffer[TYPE_OFFSET] & 63)
	if !packetType.IsSupported() {
		return nil, ErrUnsupportedType
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

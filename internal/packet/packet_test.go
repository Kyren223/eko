package packet

import (
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

func TestMsgPackEncoding(t *testing.T) {
	request := EkoMessage{Message: "test"}
	data, err := msgpack.Marshal(&request)
	if err != nil {
		t.Errorf("encoding error: %v", err)
		return
	}
	var response EkoMessage
	err = msgpack.Unmarshal(data, &response)
	if err != nil {
		t.Errorf("decoding error: %v", err)
		return
	}
	if request.Message != response.Message {
		t.Errorf("%v != %v", request.Message, response.Message)
		return
	}
}

func TestPacketMsgPackEncoding(t *testing.T) {
	request := EkoMessage{Message: "test"}
	encoder1, err := NewMsgPackEncoder(&request)
	encoder2, _ := NewMsgPackEncoder(&request)
	if err != nil {
		t.Errorf("encoding error: %v", err)
		return
	}
	encodedBytes := make([]byte, PACKET_MAX_SIZE)
	n, _ := encoder2.Read(encodedBytes[HEADER_SIZE:])

	packet := NewPacket(encoder1)
	var response EkoMessage
	err = packet.DecodePayload(&response)
	if err != nil {
		t.Errorf("decoding error: %#v: packet: %v encoder: %v", err, packet.Payload(), encodedBytes[HEADER_SIZE:HEADER_SIZE+n])
		return
	}
	if request.Message != response.Message {
		t.Errorf("%v != %v", request.Message, response.Message)
		return
	}
}

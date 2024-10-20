package packet

import (
	"context"
	"io"
	"testing"
	"time"

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

type TestIoReader struct {
	data []byte
	len  int
}

func (r *TestIoReader) Read(data []byte) (int, error) {
	if len(r.data) <= r.len {
		// log.Println("RETURNING EOF:", len(data), r.len)
		return 0, io.EOF
	}
	n := copy(data, r.data[r.len:])
	r.len += n
	// log.Println("RETURNING N:", len(r.data), r.len)
	return n, nil
}

func (r *TestIoReader) start(msg TypedMessage, t *testing.T) {
	encoder, err := NewMsgPackEncoder(msg)
	if err != nil {
		t.Errorf("encoding error: %v", err)
		return
	}
	r.data = NewPacket(encoder).data
	// log.Println("LEN:", len(r.data))
}

func TestPacketFramer(t *testing.T) {
	reader := &TestIoReader{}

	// Long msg to test multiple
	msg := EkoMessage{"Testing FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting Framer"}
	reader.start(&msg, t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, outErr := RunFramer(ctx, reader)

	var message EkoMessage
	select {
	case packet := <-out:
		if err := packet.DecodePayload(&message); err != nil {
			t.Errorf("error decoding response: %v", err)
			return
		}
		if msg.Message != message.Message {
			t.Errorf("%v != %v", msg.Message, message.Message)
			return
		}

	case err := <-outErr:
		t.Errorf("error receiving packet: %v", err)
		return
	}
}

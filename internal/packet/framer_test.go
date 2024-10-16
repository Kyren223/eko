package packet

import (
	"context"
	"io"
	"testing"
	"time"
)

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

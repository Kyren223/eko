package packet

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/util"
)

const framerPacketCapacity = 10

var (
	PacketUnsupportedVersion  error = errors.New("packet error: unsupported version")
	PacketUnsupportedEncoding error = errors.New("packet error: unsupported encoding")
	PacketUnsupportedType     error = errors.New("packet error: unsupported type")
)

type packetFramer struct {
	buffer []byte
	len    uint16
	in     chan<- Packet
	inErr  chan<- error
}

func RunFramer(ctx context.Context, reader io.Reader) (out <-chan Packet, outErr <-chan error) {
	ch := make(chan Packet, framerPacketCapacity)
	errCh := make(chan error)

	framer := packetFramer{
		buffer: make([]byte, PACKET_MAX_SIZE),
		len:    0,
		in:     ch,
		inErr:  errCh,
	}

	go framer.run(ctx, util.NewChannelReader(ctx, reader))

	return ch, errCh
}

func (f *packetFramer) run(ctx context.Context, reader util.ChannelReader) {
	defer close(f.in)
	defer close(f.inErr)

outer:
	for {
		select {
		case data := <-reader.Out:
			dataRead := 0
			for dataRead < len(data) {
				n := copy(f.buffer[f.len:], data[dataRead:])
				assert.Assert(0 <= n && n <= int(PACKET_MAX_SIZE), "n must fit in a u16")
				f.len += uint16(n)
				dataRead += n
				if err := f.parse(); err != nil {
					f.inErr <- err
					break outer
				}
			}

		case err := <-reader.Err:
			f.inErr <- err
			break outer

		case <-ctx.Done():
			f.inErr <- ctx.Err()
			break outer
		}
	}
}

func (f *packetFramer) parse() error {
	for f.len > HEADER_SIZE {
		if f.buffer[VERSION_OFFSET] != VERSION {
			return fmt.Errorf("%w version=%v", PacketUnsupportedVersion, f.buffer[VERSION_OFFSET])
		}

		encoding := Encoding(f.buffer[ENCODING_OFFSET] >> 6)
		packetType := PacketType(f.buffer[TYPE_OFFSET] & 63)
		if !encoding.IsSupported() {
			return PacketUnsupportedEncoding
		}
		if !packetType.IsSupported() {
			return PacketUnsupportedType
		}

		length := binary.BigEndian.Uint16(f.buffer[LENGTH_OFFSET:])
		if f.len-HEADER_SIZE < length {
			// Wait for more data to arrive
			return nil
		}

		fullLength := HEADER_SIZE + length
		packetBuffer := make([]byte, fullLength)
		copy(packetBuffer, f.buffer[:fullLength])

		f.len = uint16(copy(f.buffer, f.buffer[fullLength:f.len]))

		f.in <- Packet{packetBuffer}
	}
	return nil
}

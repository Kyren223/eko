package packet

import (
	"context"
	"encoding/binary"
	"errors"
	"io"

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
	// Reads a bunch from ioReader
	// If bytes r more than header size
	// Try parsing 1 or more packets
	// Send those packets to a channel
	// Have a goroutine read from the channel

	ch := make(chan Packet, framerPacketCapacity)
	errCh := make(chan error)

	framer := packetFramer{
		buffer: make([]byte, PACKET_MAX_SIZE, PACKET_MAX_SIZE),
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
			return PacketUnsupportedVersion
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
		packetBuffer := make([]byte, fullLength, fullLength)
		copy(packetBuffer, f.buffer[HEADER_SIZE:fullLength])

		f.len = uint16(copy(f.buffer[:fullLength], f.buffer[fullLength:]))

		f.in <- Packet{packetBuffer}
	}
	return nil
}

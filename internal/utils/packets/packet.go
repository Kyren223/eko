package packets

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const magicBytes = "EKO"

const (
	FlagHandshake byte = 0b1000_0000
	FlagV1        byte = 0b0000_0000
)

type Packet struct {
	flag byte
	data []byte
}

func NewPacket(flag byte, data []byte) Packet {
	return Packet{flag, data}
}

func (p Packet) Write(w io.Writer) error {
	if err := binary.Write(w, binary.LittleEndian, magicBytes); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, p.flag); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(p.data))); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, p.data); err != nil {
		return err
	}
	return nil
}

func ReadPacket(r io.Reader) (Packet, error) {
	buffer := [len(magicBytes) + 1 + 4]byte{}
	bytesRead, err := r.Read(buffer[:])
	if bytesRead != len(buffer) {
		if err != nil && err != io.EOF {
			return Packet{}, err
		}
		return Packet{}, fmt.Errorf("invalid packet size: got %v, want %v", bytesRead, len(buffer))
	}

	magic := buffer[:len(magicBytes)]
	if !bytes.Equal(magic, []byte(magicBytes)) {
		return Packet{}, fmt.Errorf("invalid magic number: got %v, want %v", magic, magicBytes)
	}

	flag := buffer[len(magicBytes)]

	// TODO: Consider adding some data limit (maybe 65k?)
	var length uint32
	err = binary.Read(
		bytes.NewReader(buffer[len(magicBytes):len(magicBytes)+4]),
		binary.LittleEndian,
		&length,
	)
	if err != nil {
		panic(fmt.Errorf("Assertion Failed in packet.ReadPacket(io.Reader): %v", err))
	}

	data := make([]byte, length)
	bytesRead, err = r.Read(data)
	if uint32(bytesRead) != length {
		if err != nil {
			return Packet{}, err
		}
		return Packet{}, fmt.Errorf("invalid data size: got %v, want %v", bytesRead, length)
	}

	return Packet{flag, data}, nil
}

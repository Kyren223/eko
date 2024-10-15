package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
)

func handleConnection(conn net.Conn, wg *sync.WaitGroup) {
	log.Println("accepted client:", conn.RemoteAddr().String())
	defer log.Println("disconnected client:", conn.RemoteAddr().String())
	defer conn.Close()
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, outErr := packet.RunFramer(ctx, conn)
	log.Printf("client %v: running framer\n", conn.RemoteAddr().String())

outer:
	for {
		select {
		case packet := <-out:
			log.Printf("client %v: request packet: %v\n", conn.RemoteAddr().String(), packet)
			responsePacket, err := handlePacket(packet)
			log.Printf("client %v: response packet: %v\n", conn.RemoteAddr().String(), responsePacket)
			if err != nil {
				log.Printf("client %v: error processing request: %v\n", conn.RemoteAddr().String(), err)
				break outer
			}
			err = responsePacket.Into(conn)
			if err != nil {
				log.Printf("client %v: error writing packet: %v\n", conn.RemoteAddr().String(), err)
				break outer
			}

		case err := <-outErr:
			if err == packet.PacketUnsupportedEncoding {
				err := unsupportedEncodingErrorPacket.Into(conn)
				log.Printf("client %v: error writing unsupported encoding packet: %v\n", conn.RemoteAddr().String(), err)
			} else if err == packet.PacketUnsupportedType {
				err := unsupportedTypeErrorPacket.Into(conn)
				log.Printf("client %v: error writing unsupported type packet: %v\n", conn.RemoteAddr().String(), err)
			} else {
				log.Printf("client %v: internal error: %v\n", conn.RemoteAddr().String(), err)
			}
			break outer

		case <-ctx.Done():
			log.Printf("client %v: %v\n", conn.RemoteAddr().String(), ctx.Err())
			break outer
		}
	}
}

func handlePacket(pkt packet.Packet) (packet.Packet, error) {
	switch pkt.Type() {
	case packet.TypeEko:
		var request packet.EkoMessage
		if err := pkt.DecodePayload(&request); err != nil {
			return packet.Packet{}, fmt.Errorf("decode error: %v", err)
		}

		response := packet.EkoMessage{Message: "Eko \"" + request.Message + "\""}
		encoder, err := packet.NewMsgPackEncoder(&response)
		if err != nil {
			return packet.Packet{}, fmt.Errorf("encode error: %v", err)
		}
		return packet.NewPacket(encoder), nil

	case packet.TypeError:
		return packet.Packet{}, errors.New("TODO: not implemented yet")
	default:
		assert.Unreachable("type should be checked for validity before handler, packet = %v", pkt.String())
		return packet.Packet{}, nil
	}
}

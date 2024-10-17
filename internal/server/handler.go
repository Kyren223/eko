package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

func handleConnection(conn net.Conn, wg *sync.WaitGroup) {
	log.Println("accepted client:", conn.RemoteAddr().String())
	defer log.Println("disconnected client:", conn.RemoteAddr().String())
	defer conn.Close()
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, outErr := packet.RunFramer(ctx, conn)

outer:
	for {
		select {
		case packet, ok := <-out:
			if !ok {
				break outer
			}
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
			} else if err != nil {
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
	var response packet.TypedMessage
	switch pkt.Type() {
	case packet.TypeEko:
		var request packet.EkoMessage
		if err := pkt.DecodePayload(&request); err != nil {
			return packet.Packet{}, fmt.Errorf("decode error: %v", err)
		}

		response = &packet.EkoMessage{Message: "Eko \"" + request.Message + "\""}
	case packet.TypeSendMessage:
		var request packet.SendMessageMessage
		if err := pkt.DecodePayload(&request); err != nil {
			return packet.Packet{}, fmt.Errorf("decode error: %v", err)
		}

		content := strings.TrimSpace(request.Content)
		if content == "" {
			response = &packet.ErrorMessage{Error: "content must not be blank"}
			break
		}

		message := data.Message{
			Id:          node.Generate(),
			SenderId:    node.Generate(),
			FrequencyId: node.Generate(),
			NetworkId:   node.Generate(),
			Contents:    content,
		}
		messages = append(messages, message)

		response = &packet.EkoMessage{Message: "Eko OK"}
	case packet.TypeGetMessages:
		var request packet.GetMessagesMessage
		if err := pkt.DecodePayload(&request); err != nil {
			return packet.Packet{}, fmt.Errorf("decode error: %v", err)
		}

		response = &packet.MessagesMessage{Messages: messages}
	default:
		return packet.Packet{}, errors.New("TODO: not implemented yet")
	}

	assert.NotNil(response, "response must always be set")
	encoder, err := packet.NewMsgPackEncoder(response)
	if err != nil {
		return packet.Packet{}, fmt.Errorf("encode error: %v", err)
	}
	return packet.NewPacket(encoder), nil
}

var (
	node     = snowflake.NewNode(1)
	messages []data.Message
)

package api

import (
	"context"
	"strings"

	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/internal/server/session"
	"github.com/kyren223/eko/pkg/assert"
)

func SendMessage(ctx context.Context, request *packet.SendMessage) packet.Payload {
	sess, ok := session.FromContext(ctx)
	assert.Assert(ok, "context in process packet should always have a session")

	content := strings.TrimSpace(request.Content)
	if content == "" {
		return &packet.ErrorMessage{Error: "message content must not be blank"}
	}

	node := sess.Manager().Node()
	message := data.Message{
		Id:          node.Generate(),
		SenderId:    sess.ID(),
		FrequencyId: node.Generate(), // TODO: replace with actual ID
		NetworkId:   node.Generate(), // TODO: replace with actual ID
		Contents:    content,
	}

	messages = append(messages, message)

	// TODO: broadcast message
	payload := &packet.Messages{Messages: messages}
	pkt := packet.NewPacket(packet.NewMsgPackEncoder(payload))
	sess.WriteQueue <- pkt

	return packet.NewOkMessage()
}

var messages []data.Message

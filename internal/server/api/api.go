package api

import (
	"context"
	"log"
	"strings"

	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/internal/server/session"
	"github.com/kyren223/eko/pkg/assert"
)

func SendMessage(ctx context.Context, request *packet.SendMessage) packet.Payload {
	sess, ok := session.FromContext(ctx)
	assert.Assert(ok, "context in process packet should always have a session")

	if (request.ReceiverID != nil) != (request.FrequencyID != nil) {
		return &packet.ErrorMessage{Error: "either receiver id or frequency id must exist"}
	}

	content := strings.TrimSpace(request.Content)
	if content == "" {
		return &packet.ErrorMessage{Error: "message content must not be blank"}
	}

	node := sess.Manager().Node()

	queries := data.New(db)
	message, err := queries.CreateMessage(ctx, data.CreateMessageParams{
		ID:          node.Generate(),
		SenderID:    sess.ID(),
		Content:     content,
		FrequencyID: request.FrequencyID,
		ReceiverID:  request.ReceiverID,
	})
	if err != nil {
		log.Println(sess.Addr(), "SendMessage database error:", err)
		return &packet.ErrorMessage{Error: "internal server error"}
	}

	return &packet.Messages{Messages: []data.Message{message}}
}

func GetMessages(ctx context.Context, request *packet.GetMessagesRange) packet.Payload {
}

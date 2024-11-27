package api

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"log"
	"strconv"
	"strings"

	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/internal/server/session"
	"github.com/kyren223/eko/pkg/snowflake"
)

func SendMessage(ctx context.Context, sess *session.Session, request *packet.SendMessage) packet.Payload {
	if (request.ReceiverID != nil) == (request.FrequencyID != nil) {
		return &packet.Error{Error: "either receiver id or frequency id must exist"}
	}

	content := strings.TrimSpace(request.Content)
	if content == "" {
		return &packet.Error{Error: "message content must not be blank"}
	}

	queries := data.New(db)
	message, err := queries.CreateMessage(ctx, data.CreateMessageParams{
		ID:          sess.Manager().Node().Generate(),
		SenderID:    sess.ID(),
		Content:     content,
		FrequencyID: request.FrequencyID,
		ReceiverID:  request.ReceiverID,
	})
	if err != nil {
		log.Println(sess.Addr(), "database error:", err, "in SendMessage")
		return &packet.Error{Error: "internal server error"}
	}

	return &packet.MessagesInfo{Messages: []data.Message{message}}
}

func RequestMessages(ctx context.Context, sess *session.Session, request *packet.RequestMessages) packet.Payload {
	queries := data.New(db)
	var messages []data.Message
	var err error

	if request.FrequencyID != nil && request.ReceiverID == nil {
		messages, err = queries.GetFrequencyMessages(ctx, request.FrequencyID)
	} else if request.ReceiverID != nil && request.FrequencyID == nil {
		messages, err = queries.GetDirectMessages(ctx, data.GetDirectMessagesParams{
			User1: sess.ID(),
			User2: request.ReceiverID,
		})
	} else {
		return &packet.Error{Error: "either receiver id or frequency id must exist"}
	}

	if err != nil {
		log.Println("database error when retrieving messages:", err)
		return &packet.Error{Error: "internal server error"}
	}
	return &packet.MessagesInfo{Messages: messages}
}

func CreateOrGetUser(ctx context.Context, node *snowflake.Node, pubKey ed25519.PublicKey) (data.User, error) {
	queries := data.New(db)
	user, err := queries.GetUserByPublicKey(ctx, pubKey)
	if err == sql.ErrNoRows {
		id := node.Generate()
		user, err = queries.CreateUser(ctx, data.CreateUserParams{
			ID:        id,
			Name:      "User" + strconv.FormatInt(id.Time()%1000, 10),
			PublicKey: pubKey,
		})
	}
	if err != nil {
		return data.User{}, err
	}
	return user, nil
}

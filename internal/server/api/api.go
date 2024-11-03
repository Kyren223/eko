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
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

func SendMessage(ctx context.Context, request *packet.SendMessage) packet.Payload {
	sess, ok := session.FromContext(ctx)
	assert.Assert(ok, "context in process packet should always have a session")

	if (request.ReceiverID != nil) == (request.FrequencyID != nil) {
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
	queries := data.New(db)
	messages, err := queries.ListMessages(ctx)
	if err != nil {
		log.Println("database error when retrieving messages:", err)
		return &packet.ErrorMessage{Error: "internal server error"}
	}
	return &packet.Messages{Messages: messages}
}

func GetUserById(ctx context.Context, request *packet.GetUserByID) packet.Payload {
	queries := data.New(db)
	user, err := queries.GetUserById(ctx, request.UserID)
	if err == sql.ErrNoRows {
		return &packet.Users{Users: []data.User{}}
	}
	if err != nil {
		log.Println("database error when retrieving user by id:", err)
		return &packet.ErrorMessage{Error: "internal server error"}
	}
	return &packet.Users{Users: []data.User{user}}
}

func CreateOrGetUser(ctx context.Context, node *snowflake.Node, pubKey ed25519.PublicKey) (data.User, error) {
	queries := data.New(db)
	user, err := queries.GetUserByPublicKey(ctx, pubKey)
	if err == sql.ErrNoRows {
		id := node.Generate()
		user, err = queries.CreateUser(ctx, data.CreateUserParams{
			ID:        id,
			Name:      "User" + strconv.FormatInt(id.Time() % 1000, 10),
			PublicKey: pubKey,
		})
	}
	if err != nil {
		return data.User{}, err
	}
	return user, nil
}

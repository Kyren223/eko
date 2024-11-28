package api

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/internal/server/session"
	"github.com/kyren223/eko/pkg/snowflake"
)

var internalError = packet.Error{Error: "internal server error"}

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
		return &internalError
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
	return user, err
}

func CreateNetwork(ctx context.Context, sess *session.Session, request *packet.CreateNetwork) packet.Payload {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return &packet.Error{Error: "server name must not be blank"}
	}

	if len(request.Icon) > MaxIconSize {
		return &packet.Error{Error: fmt.Sprintf("icon is too large, must be smaller than %v bytes", MaxIconSize)}
	}

	if ok, err := isValidHexColor(request.BgHexColor); !ok {
		return &packet.Error{Error: err}
	}
	if ok, err := isValidHexColor(request.FgHexColor); !ok {
		return &packet.Error{Error: err}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Println("database error:", err)
		return &internalError
	}
	defer tx.Rollback() //nolint

	queries := data.New(db)
	qtx := queries.WithTx(tx)

	network, err := qtx.CreateNetwork(ctx, data.CreateNetworkParams{
		ID:         sess.Manager().Node().Generate(),
		OwnerID:    sess.ID(),
		Name:       name,
		IsPublic:   request.IsPublic,
		Icon:       request.Icon,
		BgHexColor: request.BgHexColor,
		FgHexColor: request.FgHexColor,
	})
	if err != nil {
		log.Println("database error 1:", err)
		return &internalError
	}

	frequency, err := qtx.CreateFrequency(ctx, data.CreateFrequencyParams{
		ID:        sess.Manager().Node().Generate(),
		NetworkID: network.ID,
		Name:      DefaultFrequencyName,
		HexColor:  nil,
		Perms:     PermReadWrite,
	})
	if err != nil {
		log.Println("database error 2:", err)
		return &internalError
	}

	networkUser, err := qtx.SetNetworkUser(ctx, data.SetNetworkUserParams{
		UserID:    network.OwnerID,
		NetworkID: network.ID,
		IsMember:  true,
		IsAdmin:   true,
		IsMuted:   false,
		IsBanned:  false,
		BanReason: nil,
	})
	if err != nil {
		log.Println("database error 3:", err)
		return &internalError
	}

	user, err := qtx.GetUserById(ctx, network.OwnerID)
	if err != nil {
		log.Println("database error 4:", err)
		return &internalError
	}

	err = tx.Commit()
	if err != nil {
		log.Println("database error 5:", err)
		return &internalError
	}

	fullNetwork := packet.FullNetwork{
		Network:     network,
		Frequencies: []data.Frequency{frequency},
		Members: []data.GetNetworkMembersRow{{
			JoinedAt: networkUser.JoinedAt,
			User:     user,
			IsAdmin:  networkUser.IsAdmin,
			IsMuted:  networkUser.IsMuted,
		}},
		Position: int(*networkUser.Position),
	}
	return &packet.NetworksInfo{
		Networks: []packet.FullNetwork{fullNetwork},
	}
}

func GetNetworksInfo(ctx context.Context, sess *session.Session) (packet.Payload, error) {
	networksInfo := &packet.NetworksInfo{}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint

	queries := data.New(db)
	qtx := queries.WithTx(tx)

	networks, err := qtx.GetUserNetworks(ctx, sess.ID())
	if err != nil {
		return nil, err
	}

	for _, userNetwork := range networks {
		network := userNetwork.Network
		position := int(*userNetwork.Position)
		frequencies, err := qtx.GetNetworkFrequencies(ctx, network.ID)
		if err != nil {
			return nil, err
		}

		members, err := qtx.GetNetworkMembers(ctx, network.ID)
		if err != nil {
			return nil, err
		}

		networksInfo.Networks = append(networksInfo.Networks, packet.FullNetwork{
			Network:     network,
			Frequencies: frequencies,
			Members:     members,
			Position:    position,
		})
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return networksInfo, nil
}

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

var (
	ErrInternalError    = packet.Error{Error: "internal server error"}
	ErrPermissionDenied = packet.Error{Error: "permission denied"}
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
		return &ErrInternalError
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

	if len(request.Icon) > packet.MaxIconBytes {
		return &packet.Error{Error: fmt.Sprintf(
			"exceeded allowed icon size in bytes: %v", packet.MaxIconBytes,
		)}
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
		return &ErrInternalError
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
		return &ErrInternalError
	}

	frequency, err := qtx.CreateFrequency(ctx, data.CreateFrequencyParams{
		ID:        sess.Manager().Node().Generate(),
		NetworkID: network.ID,
		Name:      packet.DefaultFrequencyName,
		HexColor:  packet.DefaultFrequencyColor,
		Perms:     packet.PermReadWrite,
	})
	if err != nil {
		log.Println("database error 2:", err)
		return &ErrInternalError
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
		return &ErrInternalError
	}

	user, err := qtx.GetUserById(ctx, network.OwnerID)
	if err != nil {
		log.Println("database error 4:", err)
		return &ErrInternalError
	}

	err = tx.Commit()
	if err != nil {
		log.Println("database error 5:", err)
		return &ErrInternalError
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
		Networks:       []packet.FullNetwork{fullNetwork},
		Set:            false,
		RemoveNetworks: nil,
	}
}

func GetNetworksInfo(ctx context.Context, sess *session.Session) (packet.Payload, error) {
	var fullNetworks []packet.FullNetwork

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

		fullNetworks = append(fullNetworks, packet.FullNetwork{
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

	return &packet.NetworksInfo{
		Networks:       fullNetworks,
		RemoveNetworks: nil,
		Set:            true,
	}, nil
}

func SwapUserNetworks(ctx context.Context, sess *session.Session, request *packet.SwapUserNetworks) packet.Payload {
	queries := data.New(db)
	pos1, pos2 := int64(request.Pos1), int64(request.Pos2)
	err := queries.SwapUserNetworks(ctx, data.SwapUserNetworksParams{
		Pos1:   &pos1,
		Pos2:   &pos2,
		UserID: sess.ID(),
	})
	if err != nil {
		log.Println("database error:", err)
		return &ErrInternalError
	}

	return request
}

func CreateFrequency(ctx context.Context, sess *session.Session, request *packet.CreateFrequency) packet.Payload {
	queries := data.New(db)

	isAdmin, err := IsNetworkAdmin(ctx, queries, sess.ID(), request.Network)
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "either user or network don't exist"}
	}
	if err != nil {
		log.Println("database error 1:", err)
		return &ErrInternalError
	}
	if !isAdmin {
		return &ErrPermissionDenied
	}

	if len(request.Name) > packet.MaxFrequencyName {
		return &packet.Error{Error: fmt.Sprintf(
			"exceeded allowed frequency name length in bytes: %v",
			packet.MaxFrequencyName,
		)}
	}

	if ok, err := isValidHexColor(request.HexColor); !ok {
		return &packet.Error{Error: err}
	}

	if request.Perms < 0 || request.Perms > packet.PermMax {
		return &packet.Error{Error: fmt.Sprintf(
			"exceeded allowed perms value: 0 <= perms < %v", packet.PermMax,
		)}
	}

	frequency, err := queries.CreateFrequency(ctx, data.CreateFrequencyParams{
		ID:        sess.Manager().Node().Generate(),
		NetworkID: request.Network,
		Name:      request.Name,
		HexColor:  request.HexColor,
		Perms:     int64(request.Perms),
	})
	if err != nil {
		log.Println("database error 2:", err)
		return &ErrInternalError
	}

	return &packet.FrequenciesInfo{
		RemoveFrequencies: nil,
		Frequencies:       []data.Frequency{frequency},
		Network:           request.Network,
		Set:               false,
	}
}

func SwapFrequencies(ctx context.Context, sess *session.Session, request *packet.SwapFrequencies) packet.Payload {
	queries := data.New(db)

	isAdmin, err := IsNetworkAdmin(ctx, queries, sess.ID(), request.Network)
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "either user or network don't exist"}
	}
	if err != nil {
		log.Println("database error 1:", err)
		return &ErrInternalError
	}
	if !isAdmin {
		return &ErrPermissionDenied
	}

	err = queries.SwapFrequencies(ctx, data.SwapFrequenciesParams{
		Pos1:      int64(request.Pos1),
		Pos2:      int64(request.Pos2),
		NetworkID: request.Network,
	})
	if err != nil {
		log.Println("database error 2:", err)
		return &ErrInternalError
	}

	return &packet.SwapFrequencies{
		Network: request.Network,
		Pos1:    request.Pos1,
		Pos2:    request.Pos2,
	}
}

func DeleteFrequency(ctx context.Context, sess *session.Session, request *packet.DeleteFrequency) packet.Payload {
	queries := data.New(db)

	// Existence
	frequency, err := queries.GetFrequencyById(ctx, request.Frequency)
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "frequency doesn't exist"}
	}
	if err != nil {
		log.Println("database error 1:", err)
		return &ErrInternalError
	}

	// Authentication
	isAdmin, err := IsNetworkAdmin(ctx, queries, sess.ID(), frequency.NetworkID)
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "either user or network don't exist"}
	}
	if err != nil {
		log.Println("database error 2:", err)
		return &ErrInternalError
	}
	if !isAdmin {
		return &ErrPermissionDenied
	}

	// At least one frequency exists
	frequencies, err := queries.GetNetworkFrequencies(ctx, frequency.NetworkID)
	if err != nil {
		log.Println("database error 3:", err)
		return &ErrInternalError
	}
	if len(frequencies) == 1 {
		return &packet.Error{Error: "at least 1 frequency must exist at all times"}
	}

	err = queries.DeleteFrequency(ctx, frequency.ID)
	if err != nil {
		log.Println("database error 4:", err)
		return &ErrInternalError
	}

	return &packet.FrequenciesInfo{
		RemoveFrequencies: []snowflake.ID{frequency.ID},
		Frequencies:       nil,
		Network:           frequency.NetworkID,
		Set:               false,
	}
}

func DeleteNetwork(ctx context.Context, sess *session.Session, request *packet.DeleteNetwork) packet.Payload {
	queries := data.New(db)

	network, err := queries.GetNetworkById(ctx, request.Network)
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "network doesn't exist"}
	}
	if err != nil {
		log.Println("database error 1:", err)
		return &ErrInternalError
	}

	// NOTE: important check, make sure they are the owner (authorized)
	if network.OwnerID != sess.ID() {
		return &ErrPermissionDenied
	}

	err = queries.DeleteNetwork(ctx, request.Network)
	if err != nil {
		log.Println("database error 2:", err)
		return &ErrInternalError
	}

	return &packet.NetworksInfo{
		Networks:       nil,
		RemoveNetworks: []snowflake.ID{request.Network},
		Set:            false,
	}
}

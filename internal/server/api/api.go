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
	"github.com/kyren223/eko/pkg/assert"
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

	if len(request.Content) > packet.MaxMessageBytes {
		return &packet.Error{Error: fmt.Sprintf(
			"message conent must not exceed %v bytes",
			packet.MaxMessageBytes,
		)}
	}

	content := strings.TrimSpace(request.Content)
	if content == "" {
		return &packet.Error{Error: "message content must not be blank"}
	}

	queries := data.New(db)

	if request.FrequencyID != nil {
		frequency, err := queries.GetFrequencyById(ctx, *request.FrequencyID)
		if err == sql.ErrNoRows {
			return &packet.Error{Error: "frequency doesn't exist"}
		}
		if err != nil {
			log.Println("database error 0:", err)
			return &ErrInternalError
		}

		member, err := queries.GetMemberById(ctx, data.GetMemberByIdParams{
			NetworkID: frequency.NetworkID,
			UserID:    sess.ID(),
		})
		if err == sql.ErrNoRows {
			return &ErrPermissionDenied // Not a member
		}
		if err != nil {
			log.Println("database error 1:", err)
			return &ErrInternalError
		}
		if !member.IsMember {
			return &ErrPermissionDenied
		}

		if frequency.Perms != packet.PermReadWrite && !member.IsAdmin {
			log.Println("No perms")
			return &ErrPermissionDenied
		}

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

		return NetworkPropagate(ctx, sess, frequency.NetworkID, &packet.MessagesInfo{
			Messages:        []data.Message{message},
			RemovedMessages: nil,
		})
	}

	if request.ReceiverID != nil {
		return &packet.Error{Error: "receiver messages not supported yet!"}
	}

	assert.Never("already checked in the first line for the case where both are nil")
	return nil
}

func RequestMessages(ctx context.Context, sess *session.Session, request *packet.RequestMessages) packet.Payload {
	queries := data.New(db)

	if request.FrequencyID != nil && request.ReceiverID == nil {
		frequency, err := queries.GetFrequencyById(ctx, *request.FrequencyID)
		if err == sql.ErrNoRows {
			return &packet.Error{Error: "frequency doesn't exist"}
		}
		if err != nil {
			log.Println("database error 0:", err)
			return &ErrInternalError
		}

		member, err := queries.GetMemberById(ctx, data.GetMemberByIdParams{
			NetworkID: frequency.NetworkID,
			UserID:    sess.ID(),
		})
		if err == sql.ErrNoRows {
			return &ErrPermissionDenied // Not a member
		}
		if err != nil {
			log.Println("database error 1:", err)
			return &ErrInternalError
		}
		if !member.IsMember {
			return &ErrPermissionDenied
		}

		if frequency.Perms == packet.PermNoAccess && !member.IsAdmin {
			return &ErrPermissionDenied
		}

		messages, err := queries.GetFrequencyMessages(ctx, request.FrequencyID)
		if err != nil {
			log.Println("database error 0:", err)
			return &ErrInternalError
		}

		return &packet.MessagesInfo{
			Messages:        messages,
			RemovedMessages: nil,
		}
	}

	if request.ReceiverID != nil && request.FrequencyID == nil {
		return &packet.Error{Error: "receiver message requests are not implemented yet!"}
	}

	return &packet.Error{Error: "either receiver id or frequency id must be specified"}
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
	if len(name) > packet.MaxNetworkNameBytes {
		return &packet.Error{Error: fmt.Sprintf(
			"network name may not exceed %v bytes", packet.MaxNetworkNameBytes,
		)}
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
	defer tx.Rollback()

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

	member, err := qtx.SetMember(ctx, data.SetMemberParams{
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
		Members:     []data.Member{member},
		Users:       []data.User{user},
	}
	return &packet.NetworksInfo{
		Networks:        []packet.FullNetwork{fullNetwork},
		RemovedNetworks: nil,
		Partial:         false,
	}
}

func GetNetworksInfo(ctx context.Context, sess *session.Session) (packet.Payload, error) {
	var fullNetworks []packet.FullNetwork

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	queries := data.New(db)
	qtx := queries.WithTx(tx)

	networks, err := qtx.GetUserNetworks(ctx, sess.ID())
	if err != nil {
		return nil, err
	}

	for _, network := range networks {
		frequencies, err := qtx.GetNetworkFrequencies(ctx, network.ID)
		if err != nil {
			return nil, err
		}

		membersAndUsers, err := qtx.GetNetworkMembers(ctx, network.ID)
		if err != nil {
			return nil, err
		}
		members, users := SplitMembersAndUsers(membersAndUsers)

		fullNetworks = append(fullNetworks, packet.FullNetwork{
			Network:     network,
			Frequencies: frequencies,
			Members:     members,
			Users:       users,
		})
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &packet.NetworksInfo{
		Networks:        fullNetworks,
		RemovedNetworks: nil,
		Partial:         false,
	}, nil
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
			"exceeded allowed frequency name length, max %v bytes",
			packet.MaxFrequencyName,
		)}
	}

	if ok, err := isValidHexColor(request.HexColor); !ok {
		return &packet.Error{Error: err}
	}

	if request.Perms < 0 || request.Perms >= packet.PermMax {
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

	return NetworkPropagate(ctx, sess, request.Network, &packet.FrequenciesInfo{
		RemovedFrequencies: nil,
		Frequencies:        []data.Frequency{frequency},
		Network:            request.Network,
	})
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

	return NetworkPropagate(ctx, sess, request.Network, &packet.SwapFrequencies{
		Network: request.Network,
		Pos1:    request.Pos1,
		Pos2:    request.Pos2,
	})
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
		return &ErrPermissionDenied // User not in network
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

	return NetworkPropagate(ctx, sess, frequency.NetworkID, &packet.FrequenciesInfo{
		RemovedFrequencies: []snowflake.ID{frequency.ID},
		Frequencies:        nil,
		Network:            frequency.NetworkID,
	})
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

	return NetworkPropagate(ctx, sess, network.ID, &packet.NetworksInfo{
		Networks:        nil,
		RemovedNetworks: []snowflake.ID{request.Network},
		Partial:         false,
	})
}

func SetMember(ctx context.Context, sess *session.Session, request *packet.SetMember) packet.Payload {
	queries := data.New(db)

	network, err := queries.GetNetworkById(ctx, request.Network)
	if err != nil {
		log.Println("database error 1:", err)
		return &ErrInternalError
	}

	member, err := queries.GetMemberById(ctx, data.GetMemberByIdParams{
		NetworkID: request.Network,
		UserID:    request.User,
	})

	wantsToJoin := request.Member != nil && *request.Member && request.User == sess.ID()
	if err == sql.ErrNoRows && network.IsPublic && wantsToJoin {
		newMember, err := queries.SetMember(ctx, data.SetMemberParams{
			UserID:    request.User,
			NetworkID: request.Network,
			IsMember:  true,
			IsAdmin:   false,
			IsMuted:   false,
			IsBanned:  false,
			BanReason: nil,
		})
		if err != nil {
			log.Println("database error 2:", err)
			return &ErrInternalError
		}

		user, err := queries.GetUserById(ctx, newMember.UserID)
		if err != nil {
			log.Println("database error 3:", err)
			return &ErrInternalError
		}

		NetworkPropagate(ctx, sess, request.Network, &packet.MembersInfo{
			RemovedMembers: nil,
			Members:        []data.Member{newMember},
			Users:          []data.User{user},
			Network:        request.Network,
		})

		frequencies, err := queries.GetNetworkFrequencies(ctx, network.ID)
		if err != nil {
			log.Println("database error 4:", err)
			return &ErrInternalError
		}

		membersAndUsers, err := queries.GetNetworkMembers(ctx, network.ID)
		if err != nil {
			log.Println("database error 5:", err)
			return &ErrInternalError
		}
		members, users := SplitMembersAndUsers(membersAndUsers)

		return &packet.NetworksInfo{
			Networks: []packet.FullNetwork{{
				Network:     network,
				Frequencies: frequencies,
				Members:     members,
				Users:       users,
			}},
			RemovedNetworks: nil,
			Partial:         false,
		}
	}

	if err != nil {
		log.Println("database error 6:", err)
		return &ErrInternalError
	}

	isSessAdmin, err := IsNetworkAdmin(ctx, queries, sess.ID(), request.Network)
	if err != nil {
		log.Println("database error 7:", err)
		return &ErrInternalError
	}

	isMember := member.IsMember
	isAdmin := member.IsAdmin
	isMuted := member.IsMuted
	IsBanned := member.IsBanned
	banReason := member.BanReason

	if request.Member != nil && !IsBanned {
		isLeave := !*request.Member && request.User == sess.ID()
		isKick := !*request.Member && isSessAdmin
		if request.User != network.OwnerID && (isLeave || isKick) {
			isMember = false
			isAdmin = false // Important for security
		}
		isJoin := *request.Member && request.User == sess.ID() && network.IsPublic
		if isJoin {
			isMember = true
		}
	} else if request.Admin != nil {
		if network.OwnerID == sess.ID() && request.User != sess.ID() {
			isAdmin = *request.Admin
		}
	} else if request.Muted != nil {
		notSelf := request.User != sess.ID()
		notOwner := request.User != network.OwnerID
		if isSessAdmin && notSelf && notOwner {
			isMuted = *request.Muted
		}
	} else if request.Banned != nil {
		notSelf := request.User != sess.ID()
		notOwner := request.User != network.OwnerID
		if isSessAdmin && notSelf && notOwner {
			IsBanned = *request.Banned
			banReason = request.BanReason
			isAdmin = false // Important for security
		}
	}

	newMember, err := queries.SetMember(ctx, data.SetMemberParams{
		UserID:    request.User,
		NetworkID: request.Network,
		IsMember:  isMember,
		IsAdmin:   isAdmin,
		IsMuted:   isMuted,
		IsBanned:  IsBanned,
		BanReason: banReason,
	})
	if err != nil {
		log.Println("database error 8:", err)
		return &ErrInternalError
	}

	if !newMember.IsMember {
		NetworkPropagate(ctx, sess, request.Network, &packet.MembersInfo{
			RemovedMembers: []snowflake.ID{newMember.UserID},
			Members:        nil,
			Users:          nil,
			Network:        request.Network,
		})

		return &packet.NetworksInfo{
			Networks:        nil,
			RemovedNetworks: []snowflake.ID{request.Network},
			Partial:         false,
		}
	}

	user, err := queries.GetUserById(ctx, newMember.UserID)
	if err != nil {
		log.Println("database error 9:", err)
		return &ErrInternalError
	}

	payload := NetworkPropagate(ctx, sess, request.Network, &packet.MembersInfo{
		RemovedMembers: nil,
		Members:        []data.Member{newMember},
		Users:          []data.User{user},
		Network:        request.Network,
	})

	// Joined
	if !member.IsMember && newMember.IsMember {
		frequencies, err := queries.GetNetworkFrequencies(ctx, network.ID)
		if err != nil {
			log.Println("database error 10:", err)
			return &ErrInternalError
		}

		membersAndUsers, err := queries.GetNetworkMembers(ctx, network.ID)
		if err != nil {
			log.Println("database error 11:", err)
			return &ErrInternalError
		}
		members, users := SplitMembersAndUsers(membersAndUsers)

		return &packet.NetworksInfo{
			Networks: []packet.FullNetwork{{
				Network:     network,
				Frequencies: frequencies,
				Members:     members,
				Users:       users,
			}},
			RemovedNetworks: nil,
			Partial:         false,
		}
	}

	// Normal case, was already in the server
	return payload
}

func SetUserData(ctx context.Context, sess *session.Session, request *packet.SetUserData) packet.Payload {
	queries := data.New(db)

	if len(request.Data) > packet.MaxUserDataBytes {
		return &packet.Error{
			Error: "data bytes may not exceed " +
				strconv.FormatInt(packet.MaxUserDataBytes, 10) + " bytes",
		}
	}

	_, err := queries.SetUserData(ctx, data.SetUserDataParams{
		UserID: sess.ID(),
		Data:   request.Data,
	})
	if err != nil {
		log.Println("database error:", err)
		return &ErrInternalError
	}

	return &packet.SetUserData{
		Data: request.Data,
	}
}

func GetUserData(ctx context.Context, sess *session.Session, request *packet.GetUserData) packet.Payload {
	queries := data.New(db)

	data, err := queries.GetUserData(ctx, sess.ID())
	if err != nil {
		log.Println("database error:", err)
		return &ErrInternalError
	}

	return &packet.SetUserData{
		Data: data,
	}
}

func UpdateNetwork(ctx context.Context, sess *session.Session, request *packet.UpdateNetwork) packet.Payload {
	queries := data.New(db)

	network, err := queries.GetNetworkById(ctx, request.Network)
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "network doesn't exist"}
	}
	if err != nil {
		log.Println("database error 0:", err)
		return &ErrInternalError
	}

	if network.OwnerID != sess.ID() {
		return &ErrPermissionDenied
	}

	name := strings.TrimSpace(request.Name)
	if name == "" {
		return &packet.Error{Error: "server name must not be blank"}
	}
	if len(name) > packet.MaxNetworkNameBytes {
		return &packet.Error{Error: fmt.Sprintf(
			"network name may not exceed %v bytes", packet.MaxNetworkNameBytes,
		)}
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

	network, err = queries.UpdateNetwork(ctx, data.UpdateNetworkParams{
		Name:       name,
		Icon:       request.Icon,
		BgHexColor: request.BgHexColor,
		FgHexColor: request.FgHexColor,
		IsPublic:   request.IsPublic,
		ID:         network.ID,
	})
	if err != nil {
		log.Println("database error 1:", err)
		return &ErrInternalError
	}

	return NetworkPropagate(ctx, sess, network.ID, &packet.NetworksInfo{
		Networks: []packet.FullNetwork{{
			Network:     network,
			Frequencies: nil,
			Members:     nil,
			Users:       nil,
		}},
		RemovedNetworks: nil,
		Partial:         true,
	})
}

func UpdateFrequency(ctx context.Context, sess *session.Session, request *packet.UpdateFrequency) packet.Payload {
	queries := data.New(db)

	frequency, err := queries.GetFrequencyById(ctx, request.Frequency)
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "frequency doesn't exist"}
	}
	if err != nil {
		log.Println("database error 0:", err)
		return &ErrInternalError
	}

	isAdmin, err := IsNetworkAdmin(ctx, queries, sess.ID(), frequency.NetworkID)
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "either network doesn't exist or user is not apart of this network"}
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
			"exceeded allowed frequency name length, max %v bytes",
			packet.MaxFrequencyName,
		)}
	}

	if ok, err := isValidHexColor(request.HexColor); !ok {
		return &packet.Error{Error: err}
	}

	if request.Perms < 0 || request.Perms >= packet.PermMax {
		return &packet.Error{Error: fmt.Sprintf(
			"exceeded allowed perms value: 0 <= perms < %v", packet.PermMax,
		)}
	}

	frequency, err = queries.UpdateFrequency(ctx, data.UpdateFrequencyParams{
		Name:     request.Name,
		HexColor: request.HexColor,
		Perms:    int64(request.Perms),
		ID:       frequency.ID,
	})
	if err != nil {
		log.Println("database error 2:", err)
		return &ErrInternalError
	}

	return NetworkPropagate(ctx, sess, frequency.NetworkID, &packet.FrequenciesInfo{
		RemovedFrequencies: nil,
		Frequencies:        []data.Frequency{frequency},
		Network:            frequency.NetworkID,
	})
}

package api

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/internal/server/session"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

var (
	ErrInternalError    = packet.Error{Error: "internal server error"}
	ErrPermissionDenied = packet.Error{Error: "permission denied"}
	ErrNotImplemented   = packet.Error{Error: "not implemented yet"}
	ErrRateLimited      = packet.Error{Error: "rate limited"}
	ErrSuccess          = packet.Error{Error: "success"}

	DefaultBanReason = ""
)

func SendMessage(ctx context.Context, sess *session.Session, request *packet.SendMessage) packet.Payload {
	if (request.ReceiverID != nil) == (request.FrequencyID != nil) {
		return &packet.Error{Error: "either receiver id or frequency id must exist"}
	}

	if len(request.Content) > packet.MaxMessageBytes {
		return &packet.Error{Error: fmt.Sprintf(
			"message content must not exceed %v bytes",
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
			slog.ErrorContext(ctx, "database error", "error", err)
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
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}
		if !member.IsMember {
			return &ErrPermissionDenied
		}

		if frequency.Perms != packet.PermReadWrite && !member.IsAdmin {
			return &ErrPermissionDenied
		}

		if request.Ping != nil {
			if *request.Ping == packet.PingEveryone {
				if !member.IsAdmin {
					return &ErrPermissionDenied
				}
			} else if *request.Ping != packet.PingAdmins {
				_, err := queries.GetUserById(ctx, *request.Ping)
				if err == sql.ErrNoRows {
					return &packet.Error{Error: "pinged user doesn't exist"}
				}
				if err != nil {
					slog.ErrorContext(ctx, "database error", "error", err)
					return &ErrInternalError
				}
			}
		}

		message, err := queries.CreateMessage(ctx, data.CreateMessageParams{
			ID:          sess.Manager().Node().Generate(),
			SenderID:    sess.ID(),
			Content:     content,
			FrequencyID: request.FrequencyID,
			ReceiverID:  nil,
			Ping:        request.Ping,
		})
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		return NetworkPropagateWithFilter(ctx, sess, frequency.NetworkID, &packet.MessagesInfo{
			Messages:        []data.Message{message},
			RemovedMessages: nil,
		}, func(userId snowflake.ID) (pass bool) {
			if frequency.Perms != packet.PermNoAccess {
				return true
			}
			isAdmin, _ := IsNetworkAdmin(ctx, queries, userId, frequency.NetworkID)
			return isAdmin
		})
	}

	if request.ReceiverID != nil {
		user, err := queries.GetUserById(ctx, *request.ReceiverID)
		if err == sql.ErrNoRows {
			return &packet.Error{Error: "user doesn't exist"}
		}
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		// Session user blocked the user he tried to message
		_, err = queries.IsUserBlocked(ctx, data.IsUserBlockedParams{
			BlockingUserID: sess.ID(),
			BlockedUserID:  user.ID,
		})
		if err != nil && err != sql.ErrNoRows {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}
		if err != sql.ErrNoRows {
			// Can't message a user if you blocked them
			return &ErrPermissionDenied
		}

		// Session user was blocked by the user they tried to message
		_, err = queries.IsUserBlocked(ctx, data.IsUserBlockedParams{
			BlockingUserID: user.ID,
			BlockedUserID:  sess.ID(),
		})
		if err != nil && err != sql.ErrNoRows {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}
		if err != sql.ErrNoRows {
			// Can't message a user if they blocked you
			return &ErrPermissionDenied
		}

		if !user.IsPublicDM {
			pubKey, err := queries.GetTrustedPublicKey(ctx, data.GetTrustedPublicKeyParams{
				TrustingUserID: user.ID,
				TrustedUserID:  sess.ID(),
			})
			if err == sql.ErrNoRows {
				return &ErrPermissionDenied
			}
			if err != nil {
				slog.ErrorContext(ctx, "database error", "error", err)
				return &ErrInternalError
			}
			if !bytes.Equal(sess.PubKey(), pubKey) {
				return &ErrPermissionDenied
			}
		}

		message, err := queries.CreateMessage(ctx, data.CreateMessageParams{
			ID:          sess.Manager().Node().Generate(),
			Content:     content,
			SenderID:    sess.ID(),
			FrequencyID: nil,
			ReceiverID:  request.ReceiverID,
			Ping:        nil,
		})
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		err = queries.InsertLastReadMessage(ctx, data.InsertLastReadMessageParams{
			UserID:   *request.ReceiverID,
			SourceID: sess.ID(),
			LastRead: 0,
		})
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		return UserPropagate(ctx, sess, user.ID, &packet.MessagesInfo{
			Messages:        []data.Message{message},
			RemovedMessages: nil,
		})
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
			slog.ErrorContext(ctx, "database error", "error", err)
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
			slog.ErrorContext(ctx, "database error", "error", err)
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
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		return &packet.MessagesInfo{
			Messages:        messages,
			RemovedMessages: nil,
		}
	}

	if request.ReceiverID != nil && request.FrequencyID == nil {
		messages, err := queries.GetDirectMessages(ctx, data.GetDirectMessagesParams{
			User1: sess.ID(),
			User2: request.ReceiverID,
		})
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		return &packet.MessagesInfo{
			Messages:        messages,
			RemovedMessages: nil,
		}
	}

	return &packet.Error{Error: "either receiver id or frequency id must be specified"}
}

func CreateNetwork(ctx context.Context, sess *session.Session, request *packet.CreateNetwork) packet.Payload {
	// TODO: implement private servers
	if !request.IsPublic {
		return &ErrNotImplemented
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

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}
	defer func() { _ = tx.Rollback() }()

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
		slog.ErrorContext(ctx, "database error", "error", err)
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
		slog.ErrorContext(ctx, "database error", "error", err)
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
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	user, err := qtx.GetUserById(ctx, network.OwnerID)
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	err = tx.Commit()
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
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

func GetNetworksInfo(ctx context.Context, sess *session.Session) packet.Payload {
	var fullNetworks []packet.FullNetwork

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}
	defer func() { _ = tx.Rollback() }()

	queries := data.New(db)
	qtx := queries.WithTx(tx)

	networks, err := qtx.GetUserNetworks(ctx, sess.ID())
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	for _, network := range networks {
		frequencies, err := qtx.GetNetworkFrequencies(ctx, network.ID)
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		membersAndUsers, err := qtx.GetNetworkMembers(ctx, network.ID)
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
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
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	return &packet.NetworksInfo{
		Networks:        fullNetworks,
		RemovedNetworks: nil,
		Partial:         false,
	}
}

func CreateFrequency(ctx context.Context, sess *session.Session, request *packet.CreateFrequency) packet.Payload {
	queries := data.New(db)

	isAdmin, err := IsNetworkAdmin(ctx, queries, sess.ID(), request.Network)
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "either user or network don't exist"}
	}
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
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
		slog.ErrorContext(ctx, "database error", "error", err)
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
		slog.ErrorContext(ctx, "database error", "error", err)
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
		slog.ErrorContext(ctx, "database error", "error", err)
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
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	// Authentication
	isAdmin, err := IsNetworkAdmin(ctx, queries, sess.ID(), frequency.NetworkID)
	if err == sql.ErrNoRows {
		return &ErrPermissionDenied // User not in network
	}
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}
	if !isAdmin {
		return &ErrPermissionDenied
	}

	// At least one frequency exists
	frequencies, err := queries.GetNetworkFrequencies(ctx, frequency.NetworkID)
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}
	if len(frequencies) == 1 {
		return &packet.Error{Error: "at least 1 frequency must exist at all times"}
	}

	err = queries.DeleteFrequency(ctx, frequency.ID)
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
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
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	// NOTE: important check, make sure they are the owner (authorized)
	if network.OwnerID != sess.ID() {
		return &ErrPermissionDenied
	}

	err = queries.DeleteNetwork(ctx, request.Network)
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	return NetworkPropagate(ctx, sess, network.ID, &packet.NetworksInfo{
		Networks:        nil,
		RemovedNetworks: []snowflake.ID{request.Network},
		Partial:         false,
	})
}

func SetMember(ctx context.Context, sess *session.Session, request *packet.SetMember) packet.Payload {
	if request.BanReason != nil && len(*request.BanReason) > packet.MaxBanReasonBytes {
		return &packet.Error{Error: fmt.Sprintf(
			"Ban reason may not exceed %v bytes", packet.MaxBanReasonBytes,
		)}
	}

	queries := data.New(db)

	network, err := queries.GetNetworkById(ctx, request.Network)
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
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
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		user, err := queries.GetUserById(ctx, newMember.UserID)
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
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
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		membersAndUsers, err := queries.GetNetworkMembers(ctx, network.ID)
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
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
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	isSessOwner := sess.ID() == network.OwnerID
	isSessAdmin, err := IsNetworkAdmin(ctx, queries, sess.ID(), request.Network)
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	isMember := member.IsMember
	isAdmin := member.IsAdmin
	isMuted := member.IsMuted
	isBanned := member.IsBanned
	banReason := member.BanReason

	if request.Member != nil && !isBanned {
		isLeave := !*request.Member && request.User == sess.ID()
		isKick := !*request.Member && isSessAdmin && (!isAdmin || isSessOwner)
		if request.User != network.OwnerID && (isLeave || isKick) {
			isMember = false
			isAdmin = false // Important for security
		}
		isJoin := *request.Member && request.User == sess.ID() && network.IsPublic
		if isJoin {
			isMember = true
		}
	} else if request.Admin != nil {
		if isSessOwner && request.User != sess.ID() && isMember {
			isAdmin = *request.Admin
		}
	} else if request.Muted != nil {
		notSelf := request.User != sess.ID()
		if isSessAdmin && notSelf && (!isAdmin || isSessOwner) {
			isMuted = *request.Muted
		}
	} else if request.Banned != nil {
		notSelf := request.User != sess.ID()
		if isSessAdmin && notSelf && (!isAdmin || isSessOwner) {
			isBanned = *request.Banned
			if isBanned {
				isMember = false // kick if user got banned
				banReason = request.BanReason
				isAdmin = false // Important for security
				if banReason == nil {
					banReason = &DefaultBanReason
				}
			} else {
				banReason = nil
			}
		}
	}

	newMember, err := queries.SetMember(ctx, data.SetMemberParams{
		UserID:    request.User,
		NetworkID: request.Network,
		IsMember:  isMember,
		IsAdmin:   isAdmin,
		IsMuted:   isMuted,
		IsBanned:  isBanned,
		BanReason: banReason,
	})
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	if !newMember.IsMember && member.IsMember {
		membersInfoPayload := NetworkPropagateWithFilter(ctx, sess, request.Network, &packet.MembersInfo{
			RemovedMembers: []snowflake.ID{newMember.UserID},
			Members:        nil,
			Users:          nil,
			Network:        request.Network,
		}, func(userId snowflake.ID) (pass bool) {
			return userId != newMember.UserID
		})

		networksInfoPayload := &packet.NetworksInfo{
			Networks:        nil,
			RemovedNetworks: []snowflake.ID{request.Network},
			Partial:         false,
		}

		if newMember.UserID == sess.ID() {
			return networksInfoPayload
		} else {
			UserPropagate(ctx, sess, newMember.UserID, networksInfoPayload)
			return membersInfoPayload
		}
	}

	user, err := queries.GetUserById(ctx, newMember.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
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
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		membersAndUsers, err := queries.GetNetworkMembers(ctx, network.ID)
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
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

	// Normal case, isMember wasn't changed
	return payload
}

func SetUserData(ctx context.Context, sess *session.Session, request *packet.SetUserData) packet.Payload {
	queries := data.New(db)

	if request.Data != nil {
		if len(*request.Data) > packet.MaxUserDataBytes {
			return &packet.Error{Error: fmt.Sprintf(
				"data bytes may not exceed %v bytes",
				packet.MaxUserDataBytes,
			)}
		}

		_, err := queries.SetUserData(ctx, data.SetUserDataParams{
			UserID: sess.ID(),
			Data:   *request.Data,
		})
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}
	}

	var userPtr *data.User = nil
	if request.User != nil {

		name := request.User.Name
		if len(name) > packet.MaxUsernameBytes {
			return &packet.Error{Error: fmt.Sprintf(
				"username bytes may not exceed %v bytes",
				packet.MaxUsernameBytes,
			)}
		}

		description := request.User.Description
		if len(name) > packet.MaxUserDescriptionBytes {
			return &packet.Error{Error: fmt.Sprintf(
				"user description bytes may not exceed %v bytes",
				packet.MaxUserDescriptionBytes,
			)}
		}

		user, err := queries.UpdateUser(ctx, data.UpdateUserParams{
			Name:        name,
			Description: description,
			IsPublicDM:  request.User.IsPublicDM,
			ID:          sess.ID(),
		})
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		userPtr = &user
	}

	return &packet.SetUserData{
		Data: request.Data,
		User: userPtr,
	}
}

func GetUserData(ctx context.Context, sess *session.Session, request *packet.GetUserData) packet.Payload {
	queries := data.New(db)

	user, err := queries.GetUserById(ctx, sess.ID())
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	data, err := queries.GetUserData(ctx, sess.ID())
	if err == sql.ErrNoRows {
		data := ""
		return &packet.SetUserData{
			Data: &data,
			User: &user,
		}
	}
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	return &packet.SetUserData{
		Data: &data,
		User: &user,
	}
}

func UpdateNetwork(ctx context.Context, sess *session.Session, request *packet.UpdateNetwork) packet.Payload {
	queries := data.New(db)

	network, err := queries.GetNetworkById(ctx, request.Network)
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "network doesn't exist"}
	}
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
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
		slog.ErrorContext(ctx, "database error", "error", err)
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
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	isAdmin, err := IsNetworkAdmin(ctx, queries, sess.ID(), frequency.NetworkID)
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "either network doesn't exist or user is not apart of this network"}
	}
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
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
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	return NetworkPropagate(ctx, sess, frequency.NetworkID, &packet.FrequenciesInfo{
		RemovedFrequencies: nil,
		Frequencies:        []data.Frequency{frequency},
		Network:            frequency.NetworkID,
	})
}

func DeleteMessage(ctx context.Context, sess *session.Session, request *packet.DeleteMessage) packet.Payload {
	queries := data.New(db)

	message, err := queries.GetMessageById(ctx, request.Message)
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "message doesn't exist"}
	}
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	if message.FrequencyID != nil {
		frequency, err := queries.GetFrequencyById(ctx, *message.FrequencyID)
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		network, err := queries.GetNetworkById(ctx, frequency.NetworkID)
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		isSessAdmin, err := IsNetworkAdmin(ctx, queries, sess.ID(), frequency.NetworkID)
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		isSelf := message.SenderID == sess.ID()
		isAdmin := isSessAdmin && message.SenderID != network.OwnerID
		isOwner := network.OwnerID == sess.ID()
		if !isSelf && !isOwner && !isAdmin {
			return &ErrPermissionDenied
		}

		err = queries.DeleteMessage(ctx, message.ID)
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		return NetworkPropagateWithFilter(ctx, sess, frequency.NetworkID, &packet.MessagesInfo{
			Messages:        nil,
			RemovedMessages: []snowflake.ID{message.ID},
		}, func(userId snowflake.ID) (pass bool) {
			if frequency.Perms != packet.PermNoAccess {
				return true
			}
			isAdmin, _ := IsNetworkAdmin(ctx, queries, userId, frequency.NetworkID)
			return isAdmin
		})
	}

	if message.ReceiverID != nil {
		if message.SenderID != sess.ID() {
			return &ErrPermissionDenied
		}

		err = queries.DeleteMessage(ctx, message.ID)
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		return UserPropagate(ctx, sess, *message.ReceiverID, &packet.MessagesInfo{
			Messages:        nil,
			RemovedMessages: []snowflake.ID{message.ID},
		})
	}

	assert.Never("unreachable")
	return nil
}

func EditMessage(ctx context.Context, sess *session.Session, request *packet.EditMessage) packet.Payload {
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

	message, err := queries.GetMessageById(ctx, request.Message)
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "message doesn't exist"}
	}
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	// Note: it is possible to edit your messages in any context
	// regardless if you are in the network or if you have access to
	// the frequency (or a user signal), as long as you know the message ID
	// This should be fine but may be changed later to be more strict
	if message.SenderID != sess.ID() {
		return &ErrPermissionDenied
	}

	if message.FrequencyID != nil {
		frequency, err := queries.GetFrequencyById(ctx, *message.FrequencyID)
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		editedMessage, err := queries.EditMessage(ctx, data.EditMessageParams{
			Content: content,
			ID:      message.ID,
		})
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		return NetworkPropagateWithFilter(ctx, sess, frequency.NetworkID, &packet.MessagesInfo{
			Messages:        []data.Message{editedMessage},
			RemovedMessages: nil,
		}, func(userId snowflake.ID) (pass bool) {
			if frequency.Perms != packet.PermNoAccess {
				return true
			}
			isAdmin, _ := IsNetworkAdmin(ctx, queries, userId, frequency.NetworkID)
			return isAdmin
		})
	}

	if message.ReceiverID != nil {
		editedMessage, err := queries.EditMessage(ctx, data.EditMessageParams{
			Content: content,
			ID:      message.ID,
		})
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		return UserPropagate(ctx, sess, *message.ReceiverID, &packet.MessagesInfo{
			Messages:        []data.Message{editedMessage},
			RemovedMessages: nil,
		})
	}

	assert.Never("unreachable")
	return nil
}

func TrustUser(ctx context.Context, sess *session.Session, request *packet.TrustUser) packet.Payload {
	if sess.ID() == request.User {
		return &packet.Error{Error: "you cannot trust yourself"}
	}

	queries := data.New(db)

	user, err := queries.GetUserById(ctx, request.User)
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "requested user doesn't exist"}
	}
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	if request.Trust {
		publicKey, err := queries.GetTrustedPublicKey(ctx, data.GetTrustedPublicKeyParams{
			TrustingUserID: sess.ID(),
			TrustedUserID:  user.ID,
		})
		if err == nil {
			return &packet.TrustInfo{
				TrustedUsers:        []snowflake.ID{user.ID},
				TrustedPublicKeys:   []ed25519.PublicKey{publicKey},
				RemovedTrustedUsers: nil,
			}
		}
		if err != sql.ErrNoRows {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		_, err = queries.IsUserBlocked(ctx, data.IsUserBlockedParams{
			BlockingUserID: sess.ID(),
			BlockedUserID:  user.ID,
		})
		if err == nil {
			return &packet.Error{Error: "cannot trust blocked user, unblock them first"}
		}
		if err != sql.ErrNoRows {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		err = queries.TrustUser(ctx, data.TrustUserParams{
			TrustingUserID:   sess.ID(),
			TrustedUserID:    user.ID,
			TrustedPublicKey: user.PublicKey,
		})
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		return &packet.TrustInfo{
			TrustedUsers:        []snowflake.ID{user.ID},
			TrustedPublicKeys:   []ed25519.PublicKey{user.PublicKey},
			RemovedTrustedUsers: nil,
		}
	} else {
		err = queries.UntrustUser(ctx, data.UntrustUserParams{
			TrustingUserID: sess.ID(),
			TrustedUserID:  user.ID,
		})
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		return &packet.TrustInfo{
			TrustedUsers:        nil,
			TrustedPublicKeys:   nil,
			RemovedTrustedUsers: []snowflake.ID{user.ID},
		}
	}
}

func GetTrustedUsers(ctx context.Context, sess *session.Session) packet.Payload {
	queries := data.New(db)

	trustedRows, err := queries.GetTrustedUsers(ctx, sess.ID())
	if err != nil && err != sql.ErrNoRows {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	trusteds := make([]snowflake.ID, 0, len(trustedRows))
	trustedPublicKeys := make([]ed25519.PublicKey, 0, len(trustedRows))

	for _, row := range trustedRows {
		trusteds = append(trusteds, row.TrustedUserID)
		trustedPublicKeys = append(trustedPublicKeys, row.TrustedPublicKey)
	}

	return &packet.TrustInfo{
		TrustedUsers:        trusteds,
		TrustedPublicKeys:   trustedPublicKeys,
		RemovedTrustedUsers: nil,
	}
}

func GetBannedMembers(ctx context.Context, sess *session.Session, request *packet.GetBannedMembers) packet.Payload {
	queries := data.New(db)

	member, err := queries.GetMemberById(ctx, data.GetMemberByIdParams{
		NetworkID: request.Network,
		UserID:    sess.ID(),
	})
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "network doesn't exist"}
	}
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	if !member.IsAdmin {
		return &ErrPermissionDenied
	}

	bannedMembersRow, err := queries.GetBannedMembers(ctx, request.Network)
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}
	members := make([]data.Member, 0, len(bannedMembersRow))
	users := make([]data.User, 0, len(bannedMembersRow))
	for _, memberAndUser := range bannedMembersRow {
		members = append(members, memberAndUser.Member)
		users = append(users, memberAndUser.User)
	}

	return &packet.MembersInfo{
		RemovedMembers: nil,
		Members:        members,
		Users:          users,
		Network:        request.Network,
	}
}

func GetNotifications(ctx context.Context, sess *session.Session) packet.Payload {
	info, err := getNotifications(ctx, sess.ID())
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	return &info
}

func SetLastReadMessages(ctx context.Context, sess *session.Session, request *packet.SetLastReadMessages) packet.Payload {
	if len(request.Source) != len(request.LastRead) {
		return &packet.Error{Error: fmt.Sprintf(
			"%v sources doesn't match %v last reads",
			len(request.Source), len(request.LastRead),
		)}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}
	defer func() { _ = tx.Rollback() }()

	queries := data.New(db)
	qtx := queries.WithTx(tx)

	// OPTIMIZE: Convert this loop into a SQL query
	for i := 0; i < len(request.Source); i++ {
		_, err := qtx.GetUserById(ctx, request.Source[i])
		if err == nil {
			// ID is signal
			err = qtx.SetLastReadMessage(ctx, data.SetLastReadMessageParams{
				UserID:   sess.ID(),
				SourceID: request.Source[i],
				LastRead: request.LastRead[i],
			})
			if err != nil {
				slog.ErrorContext(ctx, "database error", "error", err)
				return &ErrInternalError
			}
			continue
		}
		if err != sql.ErrNoRows {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		// ID is frequency
		frequency, err := qtx.GetFrequencyById(ctx, request.Source[i])
		if err == sql.ErrNoRows {
			return &packet.Error{Error: fmt.Sprintf(
				"source at %v is not a valid frequency or user id", i,
			)}
		}
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}
		if frequency.Perms == packet.PermNoAccess {
			isAdmin, err := IsNetworkAdmin(ctx, qtx, sess.ID(), frequency.NetworkID)
			if err != nil {
				slog.ErrorContext(ctx, "database error", "error", err)
				return &ErrInternalError
			}
			if !isAdmin {
				return &ErrPermissionDenied
			}
		}

		err = qtx.SetLastReadMessage(ctx, data.SetLastReadMessageParams{
			UserID:   sess.ID(),
			SourceID: request.Source[i],
			LastRead: request.LastRead[i],
		})
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}
		continue
	}

	err = tx.Commit()
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	return &ErrSuccess
}

func BlockUser(ctx context.Context, sess *session.Session, request *packet.BlockUser) packet.Payload {
	if sess.ID() == request.User {
		return &packet.Error{Error: "you cannot block yourself"}
	}

	queries := data.New(db)

	user, err := queries.GetUserById(ctx, request.User)
	if err == sql.ErrNoRows {
		return &packet.Error{Error: "requested user doesn't exist"}
	}
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	if request.Block {
		_, err := queries.IsUserBlocked(ctx, data.IsUserBlockedParams{
			BlockingUserID: sess.ID(),
			BlockedUserID:  user.ID,
		})
		if err == nil {
			return &packet.BlockInfo{
				BlockedUsers:         []snowflake.ID{user.ID},
				RemovedBlockedUsers:  nil,
				BlockingUsers:        nil,
				RemovedBlockingUsers: nil,
			}
		}
		if err != sql.ErrNoRows {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		err = queries.UntrustUser(ctx, data.UntrustUserParams{
			TrustingUserID: sess.ID(),
			TrustedUserID:  user.ID,
		})
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		err = queries.BlockUser(ctx, data.BlockUserParams{
			BlockingUserID: sess.ID(),
			BlockedUserID:  user.ID,
		})
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		UserPropagate(ctx, sess, user.ID, &packet.BlockInfo{
			BlockedUsers:         nil,
			RemovedBlockedUsers:  nil,
			BlockingUsers:        []snowflake.ID{sess.ID()},
			RemovedBlockingUsers: nil,
		})

		return &packet.BlockInfo{
			BlockedUsers:         []snowflake.ID{user.ID},
			RemovedBlockedUsers:  nil,
			BlockingUsers:        nil,
			RemovedBlockingUsers: nil,
		}
	} else {
		_, err := queries.IsUserBlocked(ctx, data.IsUserBlockedParams{
			BlockingUserID: sess.ID(),
			BlockedUserID:  user.ID,
		})
		if err == sql.ErrNoRows {
			return &packet.BlockInfo{
				BlockedUsers:         nil,
				RemovedBlockedUsers:  []snowflake.ID{user.ID},
				BlockingUsers:        nil,
				RemovedBlockingUsers: nil,
			}
		}
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		err = queries.UnblockUser(ctx, data.UnblockUserParams{
			BlockingUserID: sess.ID(),
			BlockedUserID:  user.ID,
		})
		if err != nil {
			slog.ErrorContext(ctx, "database error", "error", err)
			return &ErrInternalError
		}

		UserPropagate(ctx, sess, user.ID, &packet.BlockInfo{
			BlockedUsers:         nil,
			RemovedBlockedUsers:  nil,
			BlockingUsers:        nil,
			RemovedBlockingUsers: []snowflake.ID{sess.ID()},
		})

		return &packet.BlockInfo{
			BlockedUsers:         nil,
			RemovedBlockedUsers:  []snowflake.ID{user.ID},
			BlockingUsers:        nil,
			RemovedBlockingUsers: nil,
		}
	}
}

func GetBlockedUsers(ctx context.Context, sess *session.Session) packet.Payload {
	queries := data.New(db)

	blockedUsers, err := queries.GetBlockedUsers(ctx, sess.ID())
	if err != nil && err != sql.ErrNoRows {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	blockingUsers, err := queries.GetBlockingUsers(ctx, sess.ID())
	if err != nil && err != sql.ErrNoRows {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	return &packet.BlockInfo{
		BlockedUsers:         blockedUsers,
		RemovedBlockedUsers:  nil,
		BlockingUsers:        blockingUsers,
		RemovedBlockingUsers: nil,
	}
}

func GetUsers(ctx context.Context, sess *session.Session, request *packet.GetUsers) packet.Payload {
	if len(request.Users) > packet.MaxUsersInGetUsers {
		return &packet.Error{Error: fmt.Sprintf(
			"Max users per request may not exceed %v", packet.MaxUsersInGetUsers,
		)}
	}

	queries := data.New(db)
	users, err := queries.GetUsersByIds(ctx, request.Users)
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	return &packet.UsersInfo{
		Users: users,
	}
}

func GetNonce(ctx context.Context, sess *session.Session, request *packet.GetNonce) packet.Payload {
	return &packet.NonceInfo{
		Nonce: sess.Challenge(),
	}
}

func Authenticate(ctx context.Context, sess *session.Session, request *packet.Authenticate) packet.Payload {
	if sess.IsAuthenticated() {
		return &packet.Error{Error: "already authenticated"}
	}

	if len(request.PubKey) != ed25519.PublicKeySize {
		return &packet.Error{Error: fmt.Sprintf(
			"public key must be exactly %v bytes", ed25519.PublicKeySize,
		)}
	}

	if len(request.Signature) != ed25519.SignatureSize {
		return &packet.Error{Error: fmt.Sprintf(
			"signature must be exactly %v bytes", ed25519.SignatureSize,
		)}
	}

	// IMPORTANT
	if ok := ed25519.Verify(request.PubKey, sess.Challenge(), request.Signature); !ok {
		return &packet.Error{Error: "signature verification failed"}
	}

	// Authenticated from here on out

	queries := data.New(db)

	user, err := queries.GetUserByPublicKey(ctx, request.PubKey)
	if err == sql.ErrNoRows {
		id := sess.Manager().Node().Generate()
		user, err = queries.CreateUser(ctx, data.CreateUserParams{
			ID:        id,
			Name:      "User" + strconv.FormatInt(id.Time()%1000, 10),
			PublicKey: request.PubKey,
		})
	}
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
		return &ErrInternalError
	}

	if user.IsDeleted {
		slog.WarnContext(ctx, "login with deleted user public key", "deleted_user", user.PublicKey)
		return &packet.Error{Error: "public key is already taken by a deleted user"}
	}

	sess.Manager().AddSession(sess, user.ID, request.PubKey)

	// NOTE: as per the protocol, this must be the first message after auth
	payload := &packet.UsersInfo{Users: []data.User{user}}
	ok := sess.Write(ctx, payload)
	if !ok {
		// Timeout, send at least this payload, client can request the rest
		return payload
	}

	ok = sendInitialAuthPackets(ctx, sess) // Send rest of packets
	if !ok {
		// Timeout, notify the client at least
		return &packet.Error{Error: "timeout: not all initial auth packets were sent"}
	}

	return nil // manually writing requests to control order
}

func sendInitialAuthPackets(ctx context.Context, sess *session.Session) bool {
	payloads := []packet.Payload{}

	payloads = append(payloads, GetUserData(ctx, sess, &packet.GetUserData{}))
	payloads = append(payloads, GetTrustedUsers(ctx, sess))
	payloads = append(payloads, GetBlockedUsers(ctx, sess))
	payloads = append(payloads, GetBlockedUsers(ctx, sess))
	payloads = append(payloads, GetNetworksInfo(ctx, sess))
	payloads = append(payloads, GetNotifications(ctx, sess))

	success := true
	for _, payload := range payloads {
		if payload == &ErrInternalError {
			success = false
			continue
		}
		slog.InfoContext(ctx, "sending initial auth payload", "payload", payload, "payload_type", payload.Type())
		ok := sess.Write(ctx, payload)
		if !ok {
			success = false
			continue
		}
	}

	return success
}

var (
	ipDeviceID map[uint32]string = map[uint32]string{}
	deviceIdMu sync.Mutex
)

func DeviceAnalytics(ctx context.Context, sess *session.Session, request *packet.DeviceAnalytics) packet.Payload {
	const DeviceIdLength = 64

	if len(request.DeviceID) != DeviceIdLength {
		return &packet.Error{Error: fmt.Sprintf(
			"DeviceID must be exactly %v bytes", DeviceIdLength,
		)}
	}

	for _, c := range request.DeviceID {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return &packet.Error{
				Error: "DeviceID must be all lowercase hexadecimal",
			}
		}
	}

	ip := binary.BigEndian.Uint32(sess.Addr().IP.To4())
	deviceIdMu.Lock()
	if deviceId, ok := ipDeviceID[ip]; ok {
		if request.DeviceID != deviceId {
			slog.WarnContext(ctx, "device ID mismatch", "existing_device_id", deviceId, "request_device_id", request.DeviceID)
			request.DeviceID = deviceId
			// Override the request to use the known ID
			// This avoids abuse
		}
	} else {
		ipDeviceID[ip] = request.DeviceID
	}
	deviceIdMu.Unlock()

	if !IsValidAnalytics(ctx, request) {
		// This is either malicious or we should actually add new variations
		// In either case the client shouldn't need a response
		return nil
	}

	sess.SetAnalytics(request)
	queries := data.New(db)
	queries.SetDeviceAnalytics(ctx, data.SetDeviceAnalyticsParams{
		DeviceID:  request.DeviceID,
		Os:        &request.OS,
		Arch:      &request.Arch,
		Term:      &request.Term,
		Colorterm: &request.Colorterm,
	})

	return nil
}

func SetLastUserActivity(ctx context.Context, sess *session.Session) {
	queries := data.New(db)
	now := time.Now().UnixMilli()
	err := queries.UpdateUserLastActivity(ctx, data.UpdateUserLastActivityParams{
		LastActivity: &now,
		ID:           sess.ID(),
	})
	if err != nil {
		slog.ErrorContext(ctx, "database error", "error", err)
	} else {
		slog.DebugContext(ctx, "set user activity", "now", now)
	}
}

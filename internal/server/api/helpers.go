package api

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/internal/server/session"
	"github.com/kyren223/eko/pkg/snowflake"
)

const hex = "0123456789abcdefABCDEF"

func isValidHexColor(color string) (bool, string) {
	if len(color) != 7 {
		return false, "color must be hex with length of 7"
	}

	if color[0] != '#' {
		return false, "color must start with '#'"
	}

	for _, c := range color[1:] {
		if !strings.ContainsRune(hex, c) {
			return false, "color must start with '#' and contain exactly 6 digits 0-9, a-f, A-F"
		}
	}

	return true, ""
}

func IsNetworkAdmin(ctx context.Context, queries *data.Queries, userId, networkId snowflake.ID) (bool, error) {
	userNetwork, err := queries.GetUserNetwork(ctx, data.GetUserNetworkParams{
		UserID:    userId,
		NetworkID: networkId,
	})
	if err != nil {
		return false, err
	}

	isAdmin := userNetwork.IsAdmin && userNetwork.IsMember && !userNetwork.IsBanned
	return isAdmin, nil
}

func NetworkPropagate(
	ctx context.Context, sess *session.Session,
	network snowflake.ID, payload packet.Payload,
) packet.Payload {
	var sessions []snowflake.ID
	sess.Manager().UseSessions(func(s map[snowflake.ID]*session.Session) {
		sessions = make([]snowflake.ID, 0, len(s)-1)
		for key := range s {
			if key != sess.ID() {
				sessions = append(sessions, key)
			}
		}
	})

	queries := data.New(db)
	sessions, err := queries.FilterUsersInNetwork(ctx, data.FilterUsersInNetworkParams{
		NetworkID: network,
		Users:     sessions,
	})
	if err != nil {
		log.Println("database error in propagate:", err)
		return &ErrInternalError
	}

	for _, sessionId := range sessions {
		session := sess.Manager().Session(sessionId)
		if session == nil {
			continue
		}
		timeout := 1 * time.Second
		context, cancel := context.WithTimeout(context.Background(), timeout)
		go func() {
			defer cancel()
			pkt := packet.NewPacket(packet.NewMsgPackEncoder(payload))
			if ok := session.Write(context, pkt); !ok {
				log.Println(sess.Addr(), "propagation to", session.Addr(), "failed")
			}
		}()
	}

	return payload
}

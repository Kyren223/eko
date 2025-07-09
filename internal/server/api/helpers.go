package api

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/internal/server/ctxkeys"
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
	userNetwork, err := queries.GetMemberById(ctx, data.GetMemberByIdParams{
		NetworkID: networkId,
		UserID:    userId,
	})
	if err != nil {
		return false, err
	}

	isAdmin := userNetwork.IsAdmin && userNetwork.IsMember && !userNetwork.IsBanned
	return isAdmin, nil
}

func NetworkPropagateWithFilter(
	ctx context.Context, sess *session.Session,
	network snowflake.ID, payload packet.Payload,
	filter func(userId snowflake.ID) (pass bool),
) packet.Payload {
	var sessions []snowflake.ID
	sess.Manager().UseSessions(func(s map[snowflake.ID]*session.Session) {
		sessions = make([]snowflake.ID, 0, len(s)-1)
		for key := range s {
			if key != sess.ID() && filter(key) {
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
		slog.ErrorContext(ctx, "database error", "error", err)
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
			if ok := session.Write(context, payload); !ok {
				slog.ErrorContext(ctx, "propagation failed", "session", session.LogValue(), "reason", "write failed")
			}
		}()
	}

	return payload
}

func NetworkPropagate(
	ctx context.Context, sess *session.Session,
	network snowflake.ID, payload packet.Payload,
) packet.Payload {
	return NetworkPropagateWithFilter(ctx, sess, network, payload, func(userId snowflake.ID) bool {
		return true
	})
}

func SplitMembersAndUsers(membersAndUsers []data.GetNetworkMembersRow) ([]data.Member, []data.User) {
	members := make([]data.Member, 0, len(membersAndUsers))
	users := make([]data.User, 0, len(membersAndUsers))
	for _, memberAndUser := range membersAndUsers {
		members = append(members, memberAndUser.Member)
		users = append(users, memberAndUser.User)
	}

	return members, users
}

func UserPropagate(
	ctx context.Context, sess *session.Session,
	userId snowflake.ID, payload packet.Payload,
) packet.Payload {
	session := sess.Manager().Session(userId)
	if session == nil {
		slog.ErrorContext(ctx, "propagation failed", ctxkeys.UserID.String(), userId, "reason", "session is nil")
		return payload
	}
	timeout := 1 * time.Second
	context, cancel := context.WithTimeout(context.Background(), timeout)
	go func() {
		defer cancel()
		if ok := session.Write(context, payload); !ok {
			slog.ErrorContext(ctx, "propagation failed", "session", session.LogValue(), "reason", "write failed")
		}
	}()

	return payload
}

const getNotificationsQuery = `-- name: GetNotifications :many
WITH
entries AS (
  SELECT source_id, last_read
  FROM last_read_messages
  WHERE user_id = ?
),
permitted_frequencies AS (
  SELECT f.id, m.is_admin
  FROM frequencies f
  JOIN entries e ON f.id = e.source_id
  LEFT JOIN members m
    ON m.user_id = ?
    AND m.network_id = f.network_id
  WHERE m.is_member = true AND (f.perms != 0 OR m.is_admin = true)
)
SELECT
  e.source_id, e.last_read,
  CASE
    WHEN COUNT(m.id) = 0 THEN NULL
    ELSE SUM(CASE WHEN (m.frequency_id IS NULL OR
	m.ping = 0 OR (m.ping = 1 AND pf.is_admin = true) OR m.ping = ?) THEN 1 ELSE 0 END)
	-- 0 is @everyone, 1 is @admins, otherwise it's user_id
  END AS pings
FROM entries e
LEFT JOIN permitted_frequencies pf ON e.source_id = pf.id
LEFT JOIN messages m ON m.id > e.last_read
  AND ((m.frequency_id = e.source_id AND pf.id IS NOT NULL) OR
    (m.receiver_id = e.source_id AND m.sender_id = ?) OR
    (m.sender_id = e.source_id AND m.receiver_id = ?))
GROUP BY e.source_id, e.last_read;
`

func getNotifications(ctx context.Context, userId snowflake.ID) (packet.NotificationsInfo, error) {
	query := getNotificationsQuery
	rows, err := db.QueryContext(ctx, query, userId, userId, userId, userId, userId)
	if err != nil {
		return packet.NotificationsInfo{}, err
	}
	defer rows.Close()
	var items packet.NotificationsInfo
	for rows.Next() {
		var source *snowflake.ID
		var lastRead *int64
		var pings *int64
		if err := rows.Scan(&source, &lastRead, &pings); err != nil {
			return packet.NotificationsInfo{}, err
		}
		items.Source = append(items.Source, *source)
		items.LastRead = append(items.LastRead, *lastRead)
		items.Pings = append(items.Pings, pings)
	}
	if err := rows.Close(); err != nil {
		return packet.NotificationsInfo{}, err
	}
	if err := rows.Err(); err != nil {
		return packet.NotificationsInfo{}, err
	}
	return items, nil
}

-- name: GetNetworkMembers :many
SELECT
  sqlc.embed(users),
  sqlc.embed(members)
FROM members
JOIN users ON users.id = members.user_id
WHERE network_id = ? AND is_member = true;

-- name: GetMemberById :one
SELECT * FROM members
WHERE network_id = ? AND user_id = ?;

-- name: GetUserNetworks :many
SELECT networks.* FROM networks
JOIN members ON networks.id = members.network_id
WHERE members.user_id = ? AND members.is_member = true;

-- name: SetMember :one
INSERT INTO members (
  user_id, network_id,
  is_member, is_admin, is_muted,
  is_banned, ban_reason
) VALUES (
  @user_id, @network_id,
  @is_member, @is_admin, @is_muted,
  @is_banned, @ban_reason
)
ON CONFLICT DO
UPDATE SET
  is_member = EXCLUDED.is_member, is_admin = EXCLUDED.is_admin, is_muted = EXCLUDED.is_muted,
  is_banned = EXCLUDED.is_banned, ban_reason = EXCLUDED.ban_reason
WHERE user_id = EXCLUDED.user_id AND network_id = EXCLUDED.network_id
RETURNING *;

-- name: FilterUsersInNetwork :many
SELECT user_id FROM members
WHERE network_id = ? AND user_id IN (sqlc.slice('users'));

-- name: GetNetworkBannedUsers :many
SELECT
  sqlc.embed(users),
  users_networks.ban_reason
FROM users_networks
JOIN users ON users.id = users_networks.user_id
WHERE users_networks.network_id = ?;

-- name: GetNetworkMembers :many
SELECT
  sqlc.embed(users),
  users_networks.joined_at,
  users_networks.is_admin,
  users_networks.is_muted
FROM users_networks
JOIN users ON users.id = users_networks.user_id
WHERE users_networks.network_id = ? AND is_member = true;

-- name: GetNetworkMemberById :one
SELECT *
FROM users_networks
WHERE users_networks.network_id = ? AND users_networks.user_id = ?;

-- name: GetUserNetworks :many
SELECT sqlc.embed(networks), users_networks.position FROM networks
JOIN users_networks ON networks.id = users_networks.network_id
WHERE users_networks.user_id = ?
ORDER BY users_networks.position;

-- name: GetUserNetwork :one
SELECT * FROM users_networks
WHERE user_id = ? AND network_id = ?;

-- name: SetMember :one
INSERT INTO users_networks (
  user_id, network_id,
  is_member, is_admin, is_muted,
  is_banned, ban_reason, position
) VALUES (
  @user_id, @network_id,
  @is_member, @is_admin, @is_muted,
  @is_banned, @ban_reason,
  CASE
    WHEN @is_member = false THEN NULL
    ELSE (SELECT COUNT(*) FROM users_networks WHERE user_id = @user_id)
  END
)
ON CONFLICT DO 
UPDATE SET
  is_member = EXCLUDED.is_member, is_admin = EXCLUDED.is_admin, is_muted = EXCLUDED.is_muted,
  is_banned = EXCLUDED.is_banned, ban_reason = EXCLUDED.ban_reason, position = EXCLUDED.position
WHERE user_id = EXCLUDED.user_id AND network_id = EXCLUDED.network_id
RETURNING *;

-- name: SwapUserNetworks :exec
UPDATE users_networks SET
  position = CASE
    WHEN position = @pos1 THEN @pos2
    WHEN position = @pos2 THEN @pos1
  END
WHERE user_id = @user_id AND position IN (@pos1, @pos2);

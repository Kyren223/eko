-- name: GetPublicNetworks :many
SELECT * FROM networks
WHERE is_public = true;

-- name: GetNetworkById :one
SELECT * FROM networks
WHERE id = ?;

-- name: CreateNetwork :one
INSERT INTO networks (
  id, owner_id, name, is_public,
  icon, bg_hex_color, fg_hex_color
) VALUES (
  ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetBannedUsersInNetwork :many
SELECT
  sqlc.embed(users),
  network_banned_users.banned_at,
  network_banned_users.reason
FROM network_banned_users
JOIN users ON users.id = network_banned_users.banned_user_id
WHERE network_banned_users.network_id = ?;

-- name: GetUsersInNetwork :many
SELECT
  sqlc.embed(users),
  users_networks.joined_at,
  users_networks.is_admin,
  users_networks.is_muted
FROM users_networks
JOIN users ON users.id = users_networks.user_id
WHERE users_networks.network_id = ?;

-- name: DeleteNetwork :exec
DELETE FROM networks WHERE id = ?;

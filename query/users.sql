-- name: GetUserById :one
SELECT * FROM users
WHERE id = ? AND is_deleted = false;

-- name: GetUserByPublicKey :one
SELECT * FROM users
WHERE public_key = ? AND is_deleted = false;

-- name: GetDeletedUserById :one
SELECT * FROM users
WHERE id = ? AND is_deleted = true;

-- name: CreateUser :one
INSERT INTO users (
  id, name, public_key
) VALUES (
  ?, ?, ?
)
RETURNING *;

-- name: SetUserName :one
UPDATE users SET
  name = ?
WHERE id = ? AND is_deleted = false
RETURNING *;

-- name: SetUserPublicKey :one
UPDATE users SET
  public_key = ?
WHERE id = ? AND is_deleted = false
RETURNING *;

-- name: SetUserDescription :one
UPDATE users SET
  description = ?
WHERE id = ? AND is_deleted = false
RETURNING *;

-- name: SetUserPublicDMs :one
UPDATE users SET
  is_public_dm = ?
WHERE id = ? AND is_deleted = false
RETURNING *;

-- name: DeleteUser :exec
UPDATE users SET
  is_deleted = true
WHERE id = ? AND is_deleted = false;

-- name: GetUserNetworks :many
SELECT networks.* FROM networks
JOIN users_networks ON networks.id = users_networks.network_id
WHERE users_networks.user_id = ?;

-- name: GetUserById :one
SELECT * FROM users
WHERE id = ? AND is_deleted = false;

-- name: GetUserByPublicKey :one
SELECT * FROM users
WHERE public_key = ?;

-- name: CreateUser :one
INSERT INTO users (
  id, name, public_key
) VALUES (
  ?, ?, ?
)
RETURNING *;

-- name: DeleteUser :exec
UPDATE users SET
  is_deleted = true
WHERE id = ? AND is_deleted = false;

-- name: SetUserData :one
INSERT INTO user_data (
  user_id, data
) VALUES (
  @user_id, @data
)
ON CONFLICT DO
UPDATE SET
  user_id = EXCLUDED.user_id, data = EXCLUDED.data
WHERE user_id = EXCLUDED.user_id
RETURNING *;

-- name: UpdateUser :one
UPDATE users SET
  name = ?, description = ?, is_public_dm = ?
WHERE id = ?
RETURNING *;

-- name: GetUserData :one
SELECT data FROM user_data
WHERE user_id = ?;

-- name: GetUsersByIds :many
SELECT * FROM users
WHERE id IN (sqlc.slice('ids'));

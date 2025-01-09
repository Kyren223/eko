-- name: GetUserById :one
SELECT * FROM users
WHERE id = ? AND is_deleted = false;

-- name: GetUserByPublicKey :one
SELECT * FROM users
WHERE public_key = ? AND is_deleted = false;

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

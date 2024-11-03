-- name: GetUserById :one
SELECT * FROM users
WHERE id = ?;

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

-- name: SetUserName :one
UPDATE users SET
  name = ?
WHERE id = ?
RETURNING *;

-- name: SetUserPublicKey :one
UPDATE users SET
  public_key = ?
WHERE id = ?
RETURNING *;

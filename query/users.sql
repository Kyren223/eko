-- name: GetUser :one
SELECT * FROM users
WHERE id = ?;

-- name: CreateUser :one
INSERT INTO users (
  id, name, public_key
) VALUES (
  ?, 'User' || abs(random()) % 1000000, ?
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

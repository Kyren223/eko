-- name: ListMessages :many
SELECT * FROM messages
ORDER BY id;

-- name: CreateDirectMessage :one
INSERT INTO messages (
  id, content, sender_id, receiver_id
) VALUES (
  ?, ?, ?, ?
)
RETURNING *;

-- name: CreateMessage :one
INSERT INTO messages (
  id, content, sender_id, frequency_id
) VALUES (
  ?, ?, ?, ?
)
RETURNING *;

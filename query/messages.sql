-- name: ListMessages :many
SELECT * FROM messages
ORDER BY id;

-- name: CreateMessage :one
INSERT INTO messages (
  id, content, sender_id, frequency_id, receiver_id
) VALUES (
  ?, ?, ?, ?, ?
)
RETURNING *;

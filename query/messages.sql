-- name: GetMessageById :one
SELECT * FROM messages
WHERE id = ?;

-- name: GetFrequencyMessages :many
SELECT * FROM messages
WHERE frequency_id = ?
ORDER BY id;

-- name: GetDirectMessages :many
SELECT * FROM messages
WHERE
  (sender_id = @user1 AND receiver_id = @user2) OR
  (sender_id = @user2 AND receiver_id = @user1)
ORDER BY id;

-- name: CreateMessage :one
INSERT INTO messages (
  id, content, sender_id, frequency_id, receiver_id
) VALUES (
  ?, ?, ?, ?, ?
)
RETURNING *;

-- name: EditMessage :one
UPDATE messages SET
  edited = true,
  content = ?
WHERE id = ?
RETURNING *;

-- name: DeleteMessage :exec
DELETE FROM messages
WHERE id = ?;

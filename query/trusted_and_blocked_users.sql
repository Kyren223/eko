-- name: GetTrustedUsers :many
SELECT trusted_user_id, trusted_public_key FROM trusted_users
WHERE trusting_user_id = ?;

-- name: GetTrustedPublicKey :one
SELECT trusted_public_key FROM trusted_users
WHERE trusting_user_id = ? AND trusted_user_id = ?;

-- name: TrustUser :exec
INSERT OR IGNORE INTO trusted_users (
  trusting_user_id, trusted_user_id, trusted_public_key
) VALUES (?, ?, ?);

-- name: UntrustUser :exec
DELETE FROM trusted_users
WHERE trusting_user_id = ? AND trusted_user_id = ?;

-- name: GetBlockedUsers :many
SELECT blocked_user_id FROM blocked_users
WHERE blocking_user_id = ?;

-- name: BlockUser :exec
INSERT OR IGNORE INTO blocked_users (
  blocking_user_id, blocking_user_id
) VALUES (?, ?);

-- name: UnblockUser :exec
DELETE FROM blocked_users
WHERE blocking_user_id = ? AND blocked_user_id = ?;

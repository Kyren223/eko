-- name: GetUserTrusteds :many
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

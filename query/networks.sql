-- name: GetNetworkById :one
SELECT * FROM networks
WHERE id = ?;

-- name: CreateNetwork :one
INSERT INTO networks (
  id, owner_id, name, is_public,
  icon, bg_hex_color, fg_hex_color
) VALUES (
  ?, ?, ?, ?,
  ?, ?, ?
)
RETURNING *;

-- name: TransferNetwork :one
UPDATE networks SET
  owner_id = ?
WHERE id = ?
RETURNING *;

-- name: DeleteNetwork :exec
DELETE FROM networks WHERE id = ?;

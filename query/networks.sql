-- name: GetPublicNetworks :many
SELECT * FROM networks
WHERE is_public = true;

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

-- name: SetNetworkName :one
UPDATE networks SET
  name = ?
WHERE id = ?
RETURNING *;

-- name: SetNetworkIcon :one
UPDATE networks SET
  icon = ?,
  bg_hex_color = ?,
  fg_hex_color = ?
WHERE id = ?
RETURNING *;

-- name: SetNetworkIsPublic :one
UPDATE networks SET
  is_public = ?
WHERE id = ?
RETURNING *;

-- name: TransferNetwork :one
UPDATE networks SET
  owner_id = ?
WHERE id = ?
RETURNING *;

-- name: DeleteNetwork :exec
DELETE FROM networks WHERE id = ?;

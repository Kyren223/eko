-- name: ListNetworks :many
SELECT * FROM networks
ORDER BY id;

-- name: GetNetwork :one
SELECT * FROM networks
WHERE id = ?;

-- name: CreateNetwork :one
INSERT INTO networks (
  id, name, owner_id
) VALUES (
  ?, ?, ?
)
RETURNING *;

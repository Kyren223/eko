-- name: GetNetworkFrequencies :many
SELECT * FROM frequencies
WHERE network_id = ?
ORDER BY id;

-- name: CreateFrequency :one
INSERT INTO frequencies (
  id, network_id, name
) VALUES (
  ?, ?, ?
)
RETURNING *;

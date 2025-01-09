-- name: GetNetworkFrequencies :many
SELECT * FROM frequencies
WHERE network_id = ?
ORDER BY position;

-- name: GetFrequencyById :one
SELECT * FROM frequencies
WHERE id = ?;

-- name: CreateFrequency :one
INSERT INTO frequencies (
  id, network_id,
  name, hex_color,
  perms, position
) VALUES (
  @id, @network_id, @name, @hex_color, @perms,
  (SELECT COUNT(*) FROM frequencies WHERE network_id = @network_id)
)
RETURNING *;

-- name: SwapFrequencies :exec
UPDATE frequencies SET
  position = CASE
    WHEN position = @pos1 THEN @pos2
    WHEN position = @pos2 THEN @pos1
  END
WHERE network_id = @network_id AND position IN (@pos1, @pos2);

-- name: DeleteFrequency :exec
DELETE FROM frequencies WHERE id = ?;

-- name: SetDeviceAnalytics :one
INSERT INTO device_analytics (
  device_id, os, arch, term, colorterm
) VALUES (
  @device_id, @os, @arch, @term, @colorterm
)
ON CONFLICT DO
UPDATE SET
  os = EXCLUDED.os, arch = EXCLUDED.arch, term = EXCLUDED.term, colorterm = EXCLUDED.colorterm
WHERE device_id = EXCLUDED.device_id
RETURNING *;

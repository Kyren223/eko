-- name: SetLastReadMessage :exec
INSERT INTO last_read_messages (
  user_id, source_id, last_read
) VALUES (?, ?, ?)
ON CONFLICT DO
UPDATE SET last_read = EXCLUDED.last_read
WHERE user_id = EXCLUDED.user_id AND source_id = EXCLUDED.source_id;


-- name: InsertLastReadMessage :exec
INSERT OR IGNORE INTO last_read_messages (
  user_id, source_id, last_read
) VALUES (?, ?, ?);

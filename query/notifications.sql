-- name: GetNotifications :many
WITH entries(source, lastId) AS (
  VALUES
    ('source1', 12345),
    ('source2', 54321)
)
SELECT
  e.source,
  CASE WHEN COUNT(m.id) > 0 THEN 1 ELSE 0 END AS hasNotif,
  COALESCE(SUM(CASE WHEN m.ping = ? THEN 1 ELSE 0 END), 0) AS pings
FROM entries e
LEFT JOIN messages m ON m.id > e.lastId
  AND (m.s1 = e.source OR m.s2 = e.source OR m.s3 = e.source)
GROUP BY e.source, e.lastId;

-- +goose Up
CREATE TABLE IF NOT EXISTS device_analytics (
    device_id TEXT PRIMARY KEY,
    os TEXT,
    arch TEXT,
    term TEXT,
    colorterm TEXT
);

-- +goose Down
DROP TABLE IF EXISTS device_analytics;

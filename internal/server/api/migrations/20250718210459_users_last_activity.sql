-- +goose Up
ALTER TABLE users ADD COLUMN last_activity INTEGER;

-- +goose Down
ALTER TABLE users DROP COLUMN last_activity;

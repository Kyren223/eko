-- +goose Up
CREATE TABLE IF NOT EXISTS frequencies (
  id INTEGER PRIMARY KEY,
  network_id INT NOT NULL REFERENCES networks (id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  hex_color TEXT NOT NULL,
  perms INT NOT NULL CHECK (perms IN (0, 1, 2)),
  -- 0 no access | 1 read | 2 read & write
  position INT NOT NULL
);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS on_frequency_delete
AFTER DELETE ON frequencies
BEGIN
  UPDATE frequencies SET
    position = position - 1
  WHERE network_id = OLD.network_id AND position > OLD.position;
END
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS frequencies;
DROP TRIGGER IF EXISTS on_frequency_delete;

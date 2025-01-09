-- +goose Up
CREATE TABLE IF NOT EXISTS networks (
  id INTEGER PRIMARY KEY,
  owner_id INT NOT NULL REFERENCES users (id),
  name TEXT NOT NULL,
  icon TEXT NOT NULL,
  bg_hex_color TEXT NOT NULL,
  fg_hex_color TEXT NOT NULL,
  is_public BOOLEAN NOT NULL CHECK (is_public IN (false, true))
);

-- +goose Down
DROP TABLE IF EXISTS networks;

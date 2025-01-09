-- +goose Up
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  public_key BLOB NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  is_public_dm BOOLEAN NOT NULL CHECK (is_public_dm IN (false, true)) DEFAULT true,
  is_deleted BOOLEAN NOT NULL CHECK (is_deleted IN (false, true)) DEFAULT false
);

CREATE TABLE IF NOT EXISTS user_data (
  user_id INTEGER PRIMARY KEY REFERENCES users(id),
  data TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS user_data;

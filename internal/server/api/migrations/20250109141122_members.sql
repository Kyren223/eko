-- +goose Up
CREATE TABLE IF NOT EXISTS members (
  user_id INT NOT NULL REFERENCES users (id),
  network_id INT NOT NULL REFERENCES networks (id) ON DELETE CASCADE,
  joined_at TEXT NOT NULL DEFAULT current_timestamp,
  is_member BOOLEAN NOT NULL CHECK (is_member IN (false, true)) DEFAULT true,
  is_admin BOOLEAN NOT NULL CHECK (is_admin IN (false, true)) DEFAULT false,
  is_muted BOOLEAN NOT NULL CHECK (is_muted IN (false, true)) DEFAULT false,
  is_banned BOOLEAN NOT NULL CHECK (is_banned IN (false, true)) DEFAULT false,
  ban_reason TEXT, -- null ONLY if is_banned is false
  PRIMARY KEY (user_id, network_id)
);

-- +goose Down
DROP TABLE IF EXISTS members;

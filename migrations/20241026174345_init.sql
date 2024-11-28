-- +goose Up
CREATE TABLE messages (
  id INTEGER PRIMARY KEY,
  sender_id INT NOT NULL REFERENCES users (id),
  content TEXT NOT NULL,
  edited BOOLEAN NOT NULL CHECK (edited IN (false, true)) DEFAULT false,

  frequency_id INT REFERENCES frequencies (id) ON DELETE CASCADE,
  receiver_id INT REFERENCES users (id),

  CHECK (
    (frequency_id IS NOT NULL AND receiver_id IS NULL) OR 
    (frequency_id IS NULL AND receiver_id IS NOT NULL)
  )
);

CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  public_key BLOB NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  is_public_dm BOOLEAN NOT NULL CHECK (is_public_dm IN (false, true)) DEFAULT true,
  is_deleted BOOLEAN NOT NULL CHECK (is_deleted IN (false, true)) DEFAULT false
);

CREATE TABLE networks (
  id INTEGER PRIMARY KEY,
  owner_id INT NOT NULL REFERENCES users (id),
  name TEXT NOT NULL,
  icon TEXT NOT NULL,
  bg_hex_color TEXT NOT NULL,
  fg_hex_color TEXT NOT NULL,
  is_public BOOLEAN NOT NULL CHECK (is_public IN (false, true))
);

CREATE TABLE frequencies (
  id INTEGER PRIMARY KEY,
  network_id INT NOT NULL REFERENCES networks (id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  hex_color TEXT,
  perms INT NOT NULL CHECK (perms IN (0, 1, 2)),
  -- 0 no access | 1 read | 2 read & write
  position INT NOT NULL
);

CREATE TABLE users_networks (
  user_id INT NOT NULL REFERENCES users (id),
  network_id INT NOT NULL REFERENCES networks (id) ON DELETE CASCADE,
  joined_at TEXT NOT NULL DEFAULT current_timestamp,
  is_member BOOLEAN NOT NULL CHECK (is_member IN (false, true)) DEFAULT true,
  is_admin BOOLEAN NOT NULL CHECK (is_admin IN (false, true)) DEFAULT false,
  is_muted BOOLEAN NOT NULL CHECK (is_muted IN (false, true)) DEFAULT false,
  is_banned BOOLEAN NOT NULL CHECK (is_banned IN (false, true)) DEFAULT false,
  ban_reason TEXT,
  position INT, -- null only if is_member = false
  PRIMARY KEY (user_id, network_id)
);

CREATE TABLE user_trusted_users (
  truster_user_id INT NOT NULL REFERENCES users (id),
  trusted_user_id INT NOT NULL REFERENCES users (id),
  trusted_public_key BLOB NOT NULL,
  PRIMARY KEY (truster_user_id, trusted_user_id)
);

CREATE TABLE user_blocked_users (
  blocker_user_id INT NOT NULL REFERENCES users (id),
  blocked_user_id INT NOT NULL REFERENCES users (id),
  PRIMARY KEY (blocker_user_id, blocked_user_id)
);

CREATE INDEX idx_network_name ON networks (name) WHERE is_public = true;
CREATE INDEX idx_frequency_network ON frequencies (network_id);
CREATE INDEX idx_blocked_by_user ON user_blocked_users (blocker_user_id);
CREATE INDEX idx_banned_by_network ON users_networks (network_id) WHERE is_banned = true;
CREATE INDEX idx_direct_messages ON messages (sender_id, receiver_id);
CREATE INDEX idx_frequency_messages ON messages (frequency_id);

-- +goose StatementBegin
CREATE TRIGGER on_user_delete
AFTER UPDATE OF is_deleted ON users
WHEN NEW.is_deleted = true
BEGIN
  DELETE FROM networks WHERE owner_id = NEW.id;
  DELETE FROM users_networks WHERE user_id = NEW.id;
  DELETE FROM user_blocked_users WHERE blocker_user_id = NEW.id OR blocked_user_id = NEW.id;
  DELETE FROM user_trusted_users WHERE truster_user_id = NEW.id OR trusted_user_id = NEW.id;
  DELETE FROM network_banned_users WHERE banned_user_id = NEW.id;
END
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER on_frequency_delete
AFTER DELETE ON frequencies
BEGIN
  UPDATE frequencies SET
    position = position - 1
  WHERE network_id = OLD.network_id AND position > OLD.position;
END
-- +goose StatementEnd

-- +goose Down
DROP TABLE messages;
DROP TABLE users;
DROP TABLE networks;
DROP TABLE frequencies;
DROP TABLE users_networks;
DROP TABLE user_trusted_users;
DROP TABLE user_blocked_users;
DROP TABLE network_banned_users;

DROP INDEX idx_network_name;
DROP INDEX idx_frequency_network;
DROP INDEX idx_blocked_by_user;
DROP INDEX idx_banned_by_network;
DROP INDEX idx_direct_messages;
DROP INDEX idx_frequency_messages;

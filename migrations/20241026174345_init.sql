-- +goose Up
CREATE TABLE messages (
  id INTEGER PRIMARY KEY,
  sender_id INT NOT NULL REFERENCES users (id),
  content TEXT NOT NULL,
  edited BOOLEAN NOT NULL CHECK (edited IN (0, 1)),

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
  description TEXT,
  is_public_dm BOOLEAN NOT NULL CHECK (is_public_dm IN (0, 1))
);

CREATE TABLE networks (
  id INTEGER PRIMARY KEY,
  owner_id INT NOT NULL REFERENCES users (id),
  name TEXT NOT NULL,
  icon TEXT NOT NULL,
  bg_hex_color TEXT,
  fg_hex_color TEXT NOT NULL,
  is_public BOOLEAN NOT NULL CHECK (is_public IN (0, 1))
);

CREATE TABLE frequencies (
  id INTEGER PRIMARY KEY,
  network_id INT NOT NULL REFERENCES networks (id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  hex_color TEXT,
  perms INT NOT NULL CHECK (perms IN (0, 1, 2)),
  -- 0 rw admin | 1 r all w admin | 2 rw all
  position INT NOT NULL
);

CREATE TABLE users_networks (
  user_id INT NOT NULL REFERENCES users (id),
  network_id INT NOT NULL REFERENCES networks (id) ON DELETE CASCADE,
  joined_at TEXT NOT NULL DEFAULT current_timestamp,
  is_admin BOOLEAN NOT NULL CHECK (is_admin IN (0, 1)),
  PRIMARY KEY (user_id, network_id)
);

CREATE TABLE user_trusted_users (
  truster_user_id INT NOT NULL REFERENCES users (id),
  trusted_user_id INT NOT NULL REFERENCES users (id),
  trusted_public_key BLOB NOT NULL UNIQUE,
  PRIMARY KEY (truster_user_id, trusted_user_id)
);

CREATE TABLE user_blocked_users (
  blocker_user_id INT NOT NULL REFERENCES users (id),
  blocked_user_id INT NOT NULL REFERENCES users (id),
  PRIMARY KEY (blocker_user_id, blocked_user_id)
);

CREATE TABLE network_banned_users (
  network_id INT NOT NULL REFERENCES networks (id) ON DELETE CASCADE,
  banned_user_id INT NOT NULL REFERENCES users (id),
  banned_at TEXT NOT NULL DEFAULT current_timestamp,
  reason TEXT,
  PRIMARY KEY (network_id, banned_user_id)
);

-- +goose Down
DROP TABLE messages;
DROP TABLE users;
DROP TABLE networks;
DROP TABLE frequencies;
DROP TABLE users_networks;
DROP TABLE user_trusted_users;
DROP TABLE user_blocked_users;
DROP TABLE network_banned_users;

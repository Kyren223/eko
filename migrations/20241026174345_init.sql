-- +goose Up
CREATE TABLE messages (
  id INTEGER PRIMARY KEY,
  sender_id INT NOT NULL REFERENCES users (id),
  content TEXT NOT NULL,
  edited BOOLEAN NOT NULL CHECK (edited IN (0, 1)),

  frequency_id INT REFERENCES frequencies (id),
  receiver_id INT REFERENCES users (id),

  CHECK (
    (frequency_id IS NOT NULL AND receiver_id IS NULL) OR 
    (frequency_id IS NULL AND receiver_id IS NOT NULL)
  )
);

CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  public_key BLOB NOT NULL,
  bio TEXT
);

CREATE TABLE networks (
  id INTEGER PRIMARY KEY,
  owner_id INT NOT NULL REFERENCES users (id),
  name TEXT NOT NULL,
  icon TEXT NOT NULL,
  bg TEXT NULL,
  fg TEXT NOT NULL
);

CREATE TABLE frequencies (
  id INTEGER PRIMARY KEY,
  network_id INT NOT NULL REFERENCES networks (id),
  name TEXT NOT NULL,
  color TEXT,
  perms INT NOT NULL CHECK (perms IN (0, 1, 2)),
  position INT NOT NULL
);

CREATE TABLE users_networks (
  user_id INT NOT NULL REFERENCES users (id),
  network_id INT NOT NULL REFERENCES networks (id),
  is_admin BOOLEAN NOT NULL CHECK (is_admin IN (0, 1)),
  PRIMARY KEY (user_id, network_id)
);

-- +goose Down
DROP TABLE messages;
DROP TABLE users;
DROP TABLE networks;
DROP TABLE frequencies;
DROP TABLE users_networks;

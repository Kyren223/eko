-- +goose Up
CREATE TABLE messages (
  id INTEGER PRIMARY KEY,
  sender_id INT NOT NULL REFERENCES users (id),
  content TEXT NOT NULL,

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
  public_key BLOB NOT NULL
);

CREATE TABLE networks (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  owner_id INT NOT NULL REFERENCES users (id)
);

CREATE TABLE frequencies (
  id INTEGER PRIMARY KEY,
  network_id INT NOT NULL REFERENCES networks (id),
  name TEXT NOT NULL
);

CREATE TABLE users_networks (
  user_id INT NOT NULL REFERENCES users (id),
  network_id INT NOT NULL REFERENCES networks (id),
  PRIMARY KEY (user_id, network_id)
);

-- +goose Down
DROP TABLE messages;
DROP TABLE users;
DROP TABLE networks;
DROP TABLE frequencies;
DROP TABLE users_networks;

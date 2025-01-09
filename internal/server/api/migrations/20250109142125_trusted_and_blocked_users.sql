-- +goose Up
CREATE TABLE IF NOT EXISTS trusted_users (
  trusting_user_id INT NOT NULL REFERENCES users (id),
  trusted_user_id INT NOT NULL REFERENCES users (id),
  trusted_public_key BLOB NOT NULL,
  PRIMARY KEY (trusting_user_id, trusted_user_id)
);

CREATE TABLE IF NOT EXISTS blocked_users (
  blocking_user_id INT NOT NULL REFERENCES users (id),
  blocked_user_id INT NOT NULL REFERENCES users (id),
  PRIMARY KEY (blocking_user_id, blocked_user_id)
);

-- +goose Down
DROP TABLE IF EXISTS trusted_users;
DROP TABLE IF EXISTS blocked_users;

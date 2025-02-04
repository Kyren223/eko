-- +goose Up
CREATE TABLE IF NOT EXISTS last_read_messages (
  user_id INT NOT NULL REFERENCES users (id),
  source_id INT NOT NULL, -- frequency_id or receiver_id or sender_id
  last_read INT NOT NULL,
  -- can be 0 for "no messages ever read" (null)
  -- can be between msgs in cases like the message getting deleted
  PRIMARY KEY (user_id, source_id)
);

DROP TRIGGER IF EXISTS on_frequency_delete;
-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS on_frequency_delete
AFTER DELETE ON frequencies
BEGIN
  UPDATE frequencies SET
    position = position - 1
  WHERE network_id = OLD.network_id AND position > OLD.position;

  DELETE FROM last_read_messages WHERE source_id = OLD.id;
END
-- +goose StatementEnd

DROP TRIGGER IF EXISTS on_user_delete;
-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS on_user_delete
AFTER UPDATE OF is_deleted ON users
WHEN NEW.is_deleted = true
BEGIN
  DELETE FROM networks WHERE owner_id = NEW.id;
  DELETE FROM members WHERE user_id = NEW.id;
  DELETE FROM trusted_users WHERE trusting_user_id = NEW.id OR trusted_user_id = NEW.id;
  DELETE FROM blocked_users WHERE blocking_user_id = NEW.id OR blocked_user_id = NEW.id;
  DELETE FROM last_read_messages WHERE user_id = NEW.id;
END
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS last_read_messages;

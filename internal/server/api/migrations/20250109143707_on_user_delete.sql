-- +goose Up
-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS on_user_delete
AFTER UPDATE OF is_deleted ON users
WHEN NEW.is_deleted = true
BEGIN
  DELETE FROM networks WHERE owner_id = NEW.id;
  DELETE FROM members WHERE user_id = NEW.id;
  DELETE FROM trusted_users WHERE trusting_user_id = NEW.id OR trusted_user_id = NEW.id;
  DELETE FROM blocked_users WHERE blocking_user_id = NEW.id OR blocked_user_id = NEW.id;
END
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS on_user_delete;

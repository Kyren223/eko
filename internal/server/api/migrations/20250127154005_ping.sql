-- +goose Up
ALTER TABLE messages ADD ping INTEGER DEFAULT NULL;
-- Must be null if freuqencyId is null
-- 0 - @ping:everyone, 1 - @ping:admins, otherwise references userId


-- +goose Down
ALTER TABLE messages DROP ping;

-- +goose Up
ALTER TABLE messages ADD ping INTEGER DEFAULT NULL;
-- Must be null if freuqencyId is null
-- 0 - @everyone, 1 - @admins, otherwise references userId


-- +goose Down
ALTER TABLE messages DROP ping;

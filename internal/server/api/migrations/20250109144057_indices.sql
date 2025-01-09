-- +goose Up
CREATE INDEX IF NOT EXISTS idx_frequency_network ON frequencies (network_id);
CREATE INDEX IF NOT EXISTS idx_frequency_messages ON messages (frequency_id);
CREATE INDEX IF NOT EXISTS idx_direct_messages ON messages (sender_id, receiver_id);

-- +goose Down
DROP INDEX IF EXISTS idx_frequency_network;
DROP INDEX IF EXISTS idx_frequency_messages;
DROP INDEX IF EXISTS idx_direct_messages;

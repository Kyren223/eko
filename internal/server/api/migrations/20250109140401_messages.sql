-- +goose Up
CREATE TABLE IF NOT EXISTS messages (
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

-- +goose Down
DROP TABLE IF EXISTS messages;

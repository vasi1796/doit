-- +goose Up
ALTER TABLE events ADD COLUMN counter INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE events DROP COLUMN counter;

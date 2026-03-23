-- +goose Up
CREATE TABLE ical_tokens (
    id         BIGSERIAL PRIMARY KEY,
    user_id    UUID NOT NULL UNIQUE REFERENCES users(id),
    token      VARCHAR(64) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ical_tokens_token ON ical_tokens (token);

-- +goose Down
DROP TABLE IF EXISTS ical_tokens;

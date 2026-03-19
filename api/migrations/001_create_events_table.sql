-- +goose Up
CREATE TABLE events (
    id              UUID PRIMARY KEY,
    aggregate_id    UUID NOT NULL,
    aggregate_type  VARCHAR(50) NOT NULL,
    event_type      VARCHAR(100) NOT NULL,
    user_id         UUID NOT NULL,
    data            JSONB NOT NULL,
    timestamp       TIMESTAMPTZ NOT NULL,
    version         INTEGER NOT NULL,
    UNIQUE (aggregate_id, version)
);

CREATE INDEX idx_events_aggregate ON events (aggregate_id, version);
CREATE INDEX idx_events_user_timestamp ON events (user_id, timestamp);
CREATE INDEX idx_events_type ON events (event_type);

-- +goose Down
DROP TABLE IF EXISTS events;

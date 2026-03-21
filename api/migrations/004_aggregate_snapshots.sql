-- +goose Up
CREATE TABLE aggregate_snapshots (
    aggregate_id UUID NOT NULL PRIMARY KEY,
    aggregate_type TEXT NOT NULL,
    user_id UUID NOT NULL,
    version INTEGER NOT NULL,
    data JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_snapshots_user ON aggregate_snapshots (user_id, aggregate_type);

-- +goose Down
DROP TABLE IF EXISTS aggregate_snapshots;

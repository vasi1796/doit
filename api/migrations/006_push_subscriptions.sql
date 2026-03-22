-- +goose Up

CREATE TABLE push_subscriptions (
    id         BIGSERIAL PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id),
    endpoint   TEXT NOT NULL UNIQUE,
    key_p256dh TEXT NOT NULL,
    key_auth   TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_push_subscriptions_user ON push_subscriptions (user_id);

CREATE TABLE reminder_log (
    user_id    UUID NOT NULL REFERENCES users(id),
    sent_date  DATE NOT NULL,
    task_count INTEGER NOT NULL DEFAULT 0,
    sent_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, sent_date)
);

-- +goose Down

DROP TABLE IF EXISTS reminder_log;
DROP TABLE IF EXISTS push_subscriptions;

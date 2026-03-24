-- +goose Up
CREATE TABLE task_reminder_log (
    task_id   UUID NOT NULL,
    due_date  DATE NOT NULL,
    sent_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, due_date)
);

-- +goose Down
DROP TABLE IF EXISTS task_reminder_log;

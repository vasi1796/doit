-- +goose Up
CREATE TABLE users (
    id          UUID PRIMARY KEY,
    google_id   VARCHAR(255) UNIQUE NOT NULL,
    email       VARCHAR(255) UNIQUE NOT NULL,
    name        VARCHAR(255) NOT NULL,
    avatar_url  TEXT,
    allowed     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE lists (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id),
    name        VARCHAR(255) NOT NULL,
    colour      VARCHAR(7),
    icon        VARCHAR(50),
    position    VARCHAR(255) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_lists_user ON lists (user_id);

CREATE TABLE tasks (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    list_id         UUID REFERENCES lists(id),
    title           TEXT NOT NULL,
    description     TEXT,
    priority        INTEGER NOT NULL DEFAULT 0 CHECK (priority >= 0 AND priority <= 3),
    due_date        DATE,
    due_time        TIME,
    start_date      DATE,
    recurrence_rule TEXT,
    position        VARCHAR(255) NOT NULL,
    is_completed    BOOLEAN NOT NULL DEFAULT false,
    completed_at    TIMESTAMPTZ,
    is_deleted      BOOLEAN NOT NULL DEFAULT false,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_tasks_user ON tasks (user_id);
CREATE INDEX idx_tasks_list ON tasks (list_id);
CREATE INDEX idx_tasks_due ON tasks (user_id, due_date) WHERE NOT is_deleted;
CREATE INDEX idx_tasks_deleted ON tasks (user_id, deleted_at) WHERE is_deleted;

CREATE TABLE labels (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id),
    name        VARCHAR(255) NOT NULL,
    colour      VARCHAR(7),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_labels_user ON labels (user_id);

CREATE TABLE task_labels (
    task_id     UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    label_id    UUID NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, label_id)
);

CREATE TABLE subtasks (
    id              UUID PRIMARY KEY,
    task_id         UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    is_completed    BOOLEAN NOT NULL DEFAULT false,
    position        VARCHAR(255) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_subtasks_task ON subtasks (task_id);

CREATE TABLE user_config (
    id                  UUID PRIMARY KEY,
    user_id             UUID UNIQUE NOT NULL REFERENCES users(id),
    theme               VARCHAR(20) NOT NULL DEFAULT 'system',
    sidebar_collapsed   BOOLEAN NOT NULL DEFAULT false,
    default_list_id     UUID REFERENCES lists(id),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE aggregate_snapshots (
    aggregate_id    UUID NOT NULL,
    aggregate_type  VARCHAR(50) NOT NULL,
    user_id         UUID NOT NULL REFERENCES users(id),
    data            JSONB NOT NULL,
    version         INTEGER NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (aggregate_id, aggregate_type)
);
CREATE INDEX idx_snapshots_user ON aggregate_snapshots (user_id);

-- +goose Down
DROP TABLE IF EXISTS aggregate_snapshots;
DROP TABLE IF EXISTS user_config;
DROP TABLE IF EXISTS subtasks;
DROP TABLE IF EXISTS task_labels;
DROP TABLE IF EXISTS labels;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS lists;
DROP TABLE IF EXISTS users;

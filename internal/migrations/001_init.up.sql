CREATE TABLE tasks (
    uuid        UUID PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT,
    status      VARCHAR(20) NOT NULL,
    due_time    TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ,
    deleted_at  TIMESTAMPTZ,
    version     INTEGER NOT NULL DEFAULT 1,
    flag        VARCHAR(20) NOT NULL DEFAULT 'active'
);

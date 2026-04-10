-- +goose Up
CREATE TABLE app.chat_threads (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    brand_id INTEGER NOT NULL REFERENCES app.brands(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    title TEXT,
    status TEXT NOT NULL DEFAULT 'regular' CHECK (status IN ('regular', 'archived')),
    head_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE app.chat_messages (
    id TEXT NOT NULL,
    thread_id TEXT NOT NULL REFERENCES app.chat_threads(id) ON DELETE CASCADE,
    parent_id TEXT,
    content JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (thread_id, id)
);

CREATE INDEX chat_messages_thread_id_created_at_idx ON app.chat_messages (thread_id, created_at);

GRANT SELECT, INSERT, UPDATE, DELETE ON app.chat_threads TO app;
GRANT SELECT, INSERT, UPDATE, DELETE ON app.chat_messages TO app;

-- +goose Down
DROP TABLE app.chat_messages;
DROP TABLE app.chat_threads;

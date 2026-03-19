CREATE TABLE IF NOT EXISTS messages (
    id              TEXT PRIMARY KEY DEFAULT 'msg_' || replace(uuid_generate_v4()::text, '-', ''),
    chat_id         TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    account_id      TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL,
    provider_id     TEXT NOT NULL,
    text            TEXT,
    sender_id       TEXT NOT NULL,
    is_sender       BOOLEAN NOT NULL DEFAULT FALSE,
    timestamp       TIMESTAMPTZ NOT NULL,
    attachments     JSONB NOT NULL DEFAULT '[]',
    reactions       JSONB NOT NULL DEFAULT '[]',
    quoted          JSONB,
    seen            BOOLEAN NOT NULL DEFAULT FALSE,
    delivered       BOOLEAN NOT NULL DEFAULT FALSE,
    edited          BOOLEAN NOT NULL DEFAULT FALSE,
    deleted         BOOLEAN NOT NULL DEFAULT FALSE,
    hidden          BOOLEAN NOT NULL DEFAULT FALSE,
    is_event        BOOLEAN NOT NULL DEFAULT FALSE,
    event_type      INTEGER,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(account_id, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_messages_chat ON messages(chat_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_messages_account ON messages(account_id);
CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_id);

CREATE TABLE IF NOT EXISTS chats (
    id              TEXT PRIMARY KEY DEFAULT 'chat_' || replace(uuid_generate_v4()::text, '-', ''),
    account_id      TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL,
    provider_id     TEXT NOT NULL,
    type            TEXT NOT NULL DEFAULT 'ONE_TO_ONE',
    name            TEXT,
    is_group        BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived     BOOLEAN NOT NULL DEFAULT FALSE,
    unread_count    INTEGER NOT NULL DEFAULT 0,
    last_message_at TIMESTAMPTZ,
    last_message_preview TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(account_id, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_chats_account ON chats(account_id);
CREATE INDEX IF NOT EXISTS idx_chats_updated ON chats(updated_at DESC);

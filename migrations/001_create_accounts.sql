CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS accounts (
    id              TEXT PRIMARY KEY DEFAULT 'acc_' || replace(uuid_generate_v4()::text, '-', ''),
    provider        TEXT NOT NULL,
    name            TEXT NOT NULL,
    identifier      TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'CONNECTING',
    status_detail   TEXT,
    capabilities    JSONB NOT NULL DEFAULT '[]',
    credentials_enc BYTEA,
    proxy_config    JSONB,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_synced_at  TIMESTAMPTZ,
    UNIQUE(provider, identifier)
);

CREATE INDEX IF NOT EXISTS idx_accounts_provider ON accounts(provider);
CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);

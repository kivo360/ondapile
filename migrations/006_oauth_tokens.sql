CREATE TABLE oauth_tokens (
    id              TEXT PRIMARY KEY DEFAULT 'otk_' || replace(uuid_generate_v4()::text, '-', ''),
    account_id      TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL,
    access_token_enc  BYTEA NOT NULL,
    refresh_token_enc BYTEA,
    token_type      TEXT NOT NULL DEFAULT 'Bearer',
    expiry          TIMESTAMPTZ,
    scopes          JSONB NOT NULL DEFAULT '[]',
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(account_id, provider)
);

CREATE INDEX idx_oauth_tokens_account ON oauth_tokens(account_id);
CREATE INDEX idx_oauth_tokens_provider ON oauth_tokens(provider);
CREATE INDEX idx_oauth_tokens_expiry ON oauth_tokens(expiry);

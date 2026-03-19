CREATE TABLE IF NOT EXISTS webhooks (
    id          TEXT PRIMARY KEY DEFAULT 'whk_' || replace(uuid_generate_v4()::text, '-', ''),
    url         TEXT NOT NULL,
    events      JSONB NOT NULL DEFAULT '[]',
    secret      TEXT NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id          BIGSERIAL PRIMARY KEY,
    webhook_id  TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event       TEXT NOT NULL,
    payload     JSONB NOT NULL,
    status_code INTEGER,
    attempts    INTEGER NOT NULL DEFAULT 0,
    next_retry  TIMESTAMPTZ,
    delivered   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_pending ON webhook_deliveries(next_retry) WHERE delivered = FALSE;

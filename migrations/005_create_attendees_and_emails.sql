CREATE TABLE IF NOT EXISTS attendees (
    id              TEXT PRIMARY KEY DEFAULT 'att_' || replace(uuid_generate_v4()::text, '-', ''),
    account_id      TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL,
    provider_id     TEXT NOT NULL,
    name            TEXT,
    identifier      TEXT NOT NULL,
    identifier_type TEXT NOT NULL,
    avatar_url      TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(account_id, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_attendees_account ON attendees(account_id);

-- Add emails table for IMAP/SMTP provider
CREATE TABLE IF NOT EXISTS emails (
    id                  TEXT PRIMARY KEY DEFAULT 'eml_' || replace(uuid_generate_v4()::text, '-', ''),
    account_id          TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider            TEXT NOT NULL DEFAULT 'IMAP',
    provider_id         JSONB,
    subject             TEXT,
    body                TEXT,
    body_plain          TEXT,
    from_attendee       JSONB,
    to_attendees        JSONB DEFAULT '[]',
    cc_attendees        JSONB DEFAULT '[]',
    bcc_attendees       JSONB DEFAULT '[]',
    reply_to_attendees  JSONB DEFAULT '[]',
    date_sent           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    has_attachments     BOOLEAN NOT NULL DEFAULT FALSE,
    attachments         JSONB DEFAULT '[]',
    folders             JSONB DEFAULT '["INBOX"]',
    role                TEXT NOT NULL DEFAULT 'inbox',
    is_read             BOOLEAN NOT NULL DEFAULT FALSE,
    read_date           TIMESTAMPTZ,
    is_complete         BOOLEAN NOT NULL DEFAULT FALSE,
    headers             JSONB NOT NULL DEFAULT '[]',
    tracking            JSONB DEFAULT '{}',
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_emails_account ON emails(account_id);
CREATE INDEX IF NOT EXISTS idx_emails_date ON emails(date_sent DESC);
CREATE INDEX IF NOT EXISTS idx_emails_folder ON emails(account_id, role);
CREATE INDEX IF NOT EXISTS idx_emails_unread ON emails(account_id, is_read) WHERE is_read = FALSE;

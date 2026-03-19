CREATE TABLE calendars (
    id              TEXT PRIMARY KEY DEFAULT 'cal_' || replace(uuid_generate_v4()::text, '-', ''),
    account_id      TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL,
    provider_id     TEXT NOT NULL,
    name            TEXT NOT NULL,
    color           TEXT,
    is_primary      BOOLEAN NOT NULL DEFAULT FALSE,
    is_read_only    BOOLEAN NOT NULL DEFAULT FALSE,
    timezone        TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(account_id, provider_id)
);

CREATE INDEX idx_calendars_account ON calendars(account_id);

CREATE TABLE calendar_events (
    id              TEXT PRIMARY KEY DEFAULT 'evt_' || replace(uuid_generate_v4()::text, '-', ''),
    calendar_id     TEXT NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    account_id      TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL,
    provider_id     TEXT NOT NULL,
    title           TEXT NOT NULL,
    description     TEXT,
    location        TEXT,
    start_at        TIMESTAMPTZ NOT NULL,
    end_at          TIMESTAMPTZ NOT NULL,
    all_day         BOOLEAN NOT NULL DEFAULT FALSE,
    status          TEXT NOT NULL DEFAULT 'CONFIRMED',
    attendees       JSONB NOT NULL DEFAULT '[]',
    reminders       JSONB NOT NULL DEFAULT '[]',
    conference_url  TEXT,
    recurrence      TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(account_id, provider_id)
);

CREATE INDEX idx_calendar_events_calendar ON calendar_events(calendar_id);
CREATE INDEX idx_calendar_events_account ON calendar_events(account_id);
CREATE INDEX idx_calendar_events_start ON calendar_events(start_at);
CREATE INDEX idx_calendar_events_end ON calendar_events(end_at);

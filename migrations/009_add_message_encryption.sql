-- Migration: Add message encryption support
-- Adds columns to track encrypted messages and their encryption keys

ALTER TABLE messages ADD COLUMN IF NOT EXISTS encrypted BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS encryption_key_id TEXT;

-- Add organization_id to accounts and webhooks for multi-tenant org scoping
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS organization_id TEXT;
ALTER TABLE webhooks ADD COLUMN IF NOT EXISTS organization_id TEXT;

CREATE INDEX IF NOT EXISTS idx_accounts_organization_id ON accounts(organization_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_organization_id ON webhooks(organization_id);

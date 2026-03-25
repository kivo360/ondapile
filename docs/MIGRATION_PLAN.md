# Ondapile Migration Plan: Go Backend → Hono + Drizzle

> Rewrite the Go backend (82 files, 49 endpoints) to TypeScript using Hono, Drizzle ORM, and Better Auth's native Drizzle adapter. Unified language, unified schema, unified tests.
>
> Created: 2026-03-25

---

## End State Architecture

```
Browser → TanStack Start (port 3000)
              ├── /api/auth/* → Better Auth handler (Drizzle adapter)
              ├── /api/v1/*   → Hono API routes (same process)
              ├── /t/:id      → Tracking pixel handler
              ├── /l/:id      → Link redirect handler
              └── SSR pages   → Dashboard, auth, connect

Single TypeScript process. Single PostgreSQL. Single Drizzle schema.
No Go. No port 8080. No dual auth hack.
```

## Tech Stack (Post-Migration)

| Layer | Current (Go) | Target (TypeScript) |
|-------|-------------|---------------------|
| API framework | Gin (Go) | Hono |
| ORM | pgx raw SQL | Drizzle ORM |
| Auth | Better Auth + Go middleware | Better Auth (native, no middleware) |
| IMAP client | custom Go imap_client.go | imapflow |
| SMTP client | custom Go smtp_client.go | nodemailer |
| Email tracking | custom Go tracking/ | Hono handlers + Drizzle |
| Webhook dispatch | custom Go dispatcher.go | Hono + pg-boss or custom |
| HTTP provider clients | Go net/http | fetch API |
| Tests | go test + vitest + playwright | vitest + playwright |
| Package manager | go mod + bun | bun |

---

## Migration Phases (Strangle Pattern)

### Phase 0: Setup (Foundation)
Create the new project structure alongside existing Go code.

**Tasks:**
- 0.1: Create `server/` directory with Hono + Drizzle skeleton
- 0.2: Define Drizzle schema matching ALL existing PostgreSQL tables
- 0.3: Configure Better Auth with Drizzle adapter (organization, apiKey, admin plugins)
- 0.4: Mount Better Auth in Hono at `/api/auth/*`
- 0.5: Add Zod validators for all API request/response types
- 0.6: Create shared types (export from schema for frontend + API)
- 0.7: Set up vitest with `app.request()` pattern
- 0.8: Verify Better Auth signup → org → API key flow works

**End state:** New TS server boots, Better Auth works, schema matches DB.

### Phase 1: Core API (Accounts + Auth)
Port account management — the foundation everything else depends on.

**Tasks:**
- 1.1: Port `DualAuthMiddleware` → Hono middleware using Better Auth's `apiKey.verify()`
- 1.2: Port `AccountHandler.List` → `GET /api/v1/accounts`
- 1.3: Port `AccountHandler.Create` → `POST /api/v1/accounts`
- 1.4: Port `AccountHandler.Get` → `GET /api/v1/accounts/:id`
- 1.5: Port `AccountHandler.Delete` → `DELETE /api/v1/accounts/:id`
- 1.6: Port `AccountHandler.Reconnect` → `POST /api/v1/accounts/:id/reconnect`
- 1.7: Port `AccountHandler.GetAuthChallenge` → `GET /api/v1/accounts/:id/auth-challenge`
- 1.8: Port `AccountHandler.GetQRCode` → `GET /api/v1/accounts/:id/qr`
- 1.9: Port `AccountHandler.SolveCheckpoint` → `POST /api/v1/accounts/:id/checkpoint`
- 1.10: Port `HostedAuthHandler.Create` → `POST /api/v1/accounts/hosted-auth`
- 1.11: Port `OAuthCallbackHandler.Callback` → `GET /api/v1/oauth/callback/:provider`
- 1.12: Port credential encryption (AES-256-GCM) to Node crypto
- 1.13: Port `HealthHandler` → `GET /health`
- 1.14: Port `MetricsHandler` → `GET /metrics`
- 1.15: Port CORS middleware
- 1.16: Port rate limiting middleware

**End state:** Account CRUD works through new TS server.

### Phase 2: Email Operations
Port all 9 email endpoints + IMAP/SMTP adapters.

**Tasks:**
- 2.1: Create provider adapter interface in TypeScript
- 2.2: Port IMAP adapter using `imapflow` library
- 2.3: Port SMTP sending using `nodemailer`
- 2.4: Port `EmailHandler.List` → `GET /api/v1/emails`
- 2.5: Port `EmailHandler.Send` → `POST /api/v1/emails`
- 2.6: Port `EmailHandler.Get` → `GET /api/v1/emails/:id`
- 2.7: Port `EmailHandler.Update` → `PUT /api/v1/emails/:id`
- 2.8: Port `EmailHandler.Delete` → `DELETE /api/v1/emails/:id`
- 2.9: Port `EmailHandler.Reply` → `POST /api/v1/emails/:id/reply`
- 2.10: Port `EmailHandler.Forward` → `POST /api/v1/emails/:id/forward`
- 2.11: Port `EmailHandler.ListFolders` → `GET /api/v1/emails/folders`
- 2.12: Port `EmailHandler.DownloadAttachment` → `GET /api/v1/emails/:id/attachments/:att_id`
- 2.13: Port email store (StoreEmail, GetEmail, ListEmails, etc.)
- 2.14: Port Gmail adapter (OAuth + Gmail API HTTP calls)
- 2.15: Port Outlook adapter (OAuth + Microsoft Graph HTTP calls)
- 2.16: Port IMAP polling loop (30s interval)

**End state:** All 14 email operations work through new TS server.

### Phase 3: Webhooks + Tracking
Port the event system.

**Tasks:**
- 3.1: Port webhook dispatcher (async delivery, HMAC signing, retry logic)
- 3.2: Port webhook CRUD endpoints (list, create, delete)
- 3.3: Port tracking pixel handler (`GET /t/:id`)
- 3.4: Port tracking link handler (`GET /l/:id`)
- 3.5: Port tracking store (RecordOpen, RecordClick)
- 3.6: Port audit log handler + store
- 3.7: Wire webhook dispatch into email send/reply/forward handlers
- 3.8: Wire webhook dispatch into tracking handlers (email.opened, email.clicked)

**End state:** Webhooks fire on all events, tracking works.

### Phase 4: Messaging + Calendar (Post-v1 but port the routes)
Port the remaining endpoints so the API surface is complete.

**Tasks:**
- 4.1: Port chat endpoints (7 routes)
- 4.2: Port message endpoints (5 routes)
- 4.3: Port attendee endpoints (5 routes)
- 4.4: Port calendar endpoints (7 routes)
- 4.5: Port search endpoint (1 route)
- 4.6: Port WhatsApp adapter skeleton
- 4.7: Port LinkedIn adapter skeleton
- 4.8: Port Telegram adapter skeleton

**End state:** Full API parity with Go backend.

### Phase 5: Frontend Integration
Switch the frontend from dual-server to single-server.

**Tasks:**
- 5.1: Remove `window.location.origin.replace(":3000", ":8080")` from `api-client.ts`
- 5.2: Change `fetchApi` to call same-origin `/api/v1/*` (no port switch)
- 5.3: Remove Go backend startup from dev workflow
- 5.4: Update `.env` to remove Go-specific vars
- 5.5: Update Playwright evals to test against single server
- 5.6: Verify all dashboard pages load correctly

**End state:** Frontend talks to single TS server. Go backend deleted.

### Phase 6: Cleanup
Remove Go code and finalize.

**Tasks:**
- 6.1: Delete `internal/`, `cmd/`, `go.mod`, `go.sum`
- 6.2: Delete Go migration files (replaced by Drizzle migrations)
- 6.3: Update README, deployment docs
- 6.4: Update `.env.example`
- 6.5: Final eval run — all 300+ evals must pass

---

## Drizzle Schema (Target)

```typescript
// server/src/db/schema.ts
import {
  pgTable, text, boolean, timestamp, integer, bigserial,
  jsonb, index, uniqueIndex,
} from 'drizzle-orm/pg-core';
import { relations } from 'drizzle-orm';

// ═══════════════════════════════════════════
// App Tables (custom — NOT managed by Better Auth)
// ═══════════════════════════════════════════

export const accounts = pgTable('accounts', {
  id: text('id').primaryKey().$defaultFn(() => `acc_${crypto.randomUUID().replaceAll('-', '')}`),
  organizationId: text('organization_id'),
  provider: text('provider').notNull(),
  name: text('name').notNull(),
  identifier: text('identifier').notNull(),
  status: text('status').notNull().default('CONNECTING'),
  statusDetail: text('status_detail'),
  capabilities: jsonb('capabilities').notNull().default([]),
  credentialsEnc: text('credentials_enc'), // base64-encoded encrypted blob
  proxyConfig: jsonb('proxy_config'),
  metadata: jsonb('metadata').notNull().default({}),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).notNull().defaultNow(),
  lastSyncedAt: timestamp('last_synced_at', { withTimezone: true }),
}, (t) => [
  index('idx_accounts_provider').on(t.provider),
  index('idx_accounts_status').on(t.status),
  index('idx_accounts_org').on(t.organizationId),
  uniqueIndex('idx_accounts_provider_identifier').on(t.provider, t.identifier),
]);

export const emails = pgTable('emails', {
  id: text('id').primaryKey().$defaultFn(() => `eml_${crypto.randomUUID().replaceAll('-', '')}`),
  accountId: text('account_id').notNull().references(() => accounts.id, { onDelete: 'cascade' }),
  provider: text('provider').notNull().default('IMAP'),
  providerId: jsonb('provider_id'), // { messageId, threadId }
  subject: text('subject'),
  body: text('body'),
  bodyPlain: text('body_plain'),
  fromAttendee: jsonb('from_attendee'),
  toAttendees: jsonb('to_attendees').default([]),
  ccAttendees: jsonb('cc_attendees').default([]),
  bccAttendees: jsonb('bcc_attendees').default([]),
  replyToAttendees: jsonb('reply_to_attendees').default([]),
  dateSent: timestamp('date_sent', { withTimezone: true }).notNull().defaultNow(),
  hasAttachments: boolean('has_attachments').notNull().default(false),
  attachments: jsonb('attachments').default([]),
  folders: jsonb('folders').default(['INBOX']),
  role: text('role').notNull().default('inbox'),
  isRead: boolean('is_read').notNull().default(false),
  readDate: timestamp('read_date', { withTimezone: true }),
  isComplete: boolean('is_complete').notNull().default(false),
  headers: jsonb('headers').notNull().default([]),
  tracking: jsonb('tracking').default({}),
  metadata: jsonb('metadata').notNull().default({}),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).notNull().defaultNow(),
}, (t) => [
  index('idx_emails_account').on(t.accountId),
  index('idx_emails_date').on(t.dateSent),
  index('idx_emails_folder').on(t.accountId, t.role),
]);

export const chats = pgTable('chats', {
  id: text('id').primaryKey().$defaultFn(() => `chat_${crypto.randomUUID().replaceAll('-', '')}`),
  accountId: text('account_id').notNull().references(() => accounts.id, { onDelete: 'cascade' }),
  provider: text('provider').notNull(),
  providerId: text('provider_id').notNull(),
  type: text('type').notNull().default('ONE_TO_ONE'),
  name: text('name'),
  isGroup: boolean('is_group').notNull().default(false),
  isArchived: boolean('is_archived').notNull().default(false),
  unreadCount: integer('unread_count').notNull().default(0),
  lastMessageAt: timestamp('last_message_at', { withTimezone: true }),
  lastMessagePreview: text('last_message_preview'),
  metadata: jsonb('metadata').notNull().default({}),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).notNull().defaultNow(),
}, (t) => [
  index('idx_chats_account').on(t.accountId),
  uniqueIndex('idx_chats_provider').on(t.accountId, t.providerId),
]);

export const messages = pgTable('messages', {
  id: text('id').primaryKey().$defaultFn(() => `msg_${crypto.randomUUID().replaceAll('-', '')}`),
  chatId: text('chat_id').notNull().references(() => chats.id, { onDelete: 'cascade' }),
  accountId: text('account_id').notNull().references(() => accounts.id, { onDelete: 'cascade' }),
  provider: text('provider').notNull(),
  providerId: text('provider_id').notNull(),
  text: text('text'),
  senderId: text('sender_id').notNull(),
  isSender: boolean('is_sender').notNull().default(false),
  timestamp: timestamp('timestamp', { withTimezone: true }).notNull(),
  attachments: jsonb('attachments').notNull().default([]),
  reactions: jsonb('reactions').notNull().default([]),
  quoted: jsonb('quoted'),
  seen: boolean('seen').notNull().default(false),
  delivered: boolean('delivered').notNull().default(false),
  edited: boolean('edited').notNull().default(false),
  deleted: boolean('deleted').notNull().default(false),
  hidden: boolean('hidden').notNull().default(false),
  isEvent: boolean('is_event').notNull().default(false),
  eventType: integer('event_type'),
  metadata: jsonb('metadata').notNull().default({}),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
}, (t) => [
  index('idx_messages_chat').on(t.chatId),
  index('idx_messages_account').on(t.accountId),
  uniqueIndex('idx_messages_provider').on(t.accountId, t.providerId),
]);

export const webhooks = pgTable('webhooks', {
  id: text('id').primaryKey().$defaultFn(() => `whk_${crypto.randomUUID().replaceAll('-', '')}`),
  organizationId: text('organization_id'),
  url: text('url').notNull(),
  events: jsonb('events').notNull().default([]),
  secret: text('secret').notNull(),
  active: boolean('active').notNull().default(true),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
}, (t) => [
  index('idx_webhooks_org').on(t.organizationId),
]);

export const webhookDeliveries = pgTable('webhook_deliveries', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  webhookId: text('webhook_id').notNull().references(() => webhooks.id, { onDelete: 'cascade' }),
  event: text('event').notNull(),
  payload: jsonb('payload').notNull(),
  statusCode: integer('status_code'),
  attempts: integer('attempts').notNull().default(0),
  nextRetry: timestamp('next_retry', { withTimezone: true }),
  delivered: boolean('delivered').notNull().default(false),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
});

export const attendees = pgTable('attendees', {
  id: text('id').primaryKey().$defaultFn(() => `att_${crypto.randomUUID().replaceAll('-', '')}`),
  accountId: text('account_id').notNull().references(() => accounts.id, { onDelete: 'cascade' }),
  provider: text('provider').notNull(),
  providerId: text('provider_id').notNull(),
  name: text('name'),
  identifier: text('identifier').notNull(),
  identifierType: text('identifier_type').notNull(),
  avatarUrl: text('avatar_url'),
  metadata: jsonb('metadata').notNull().default({}),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).notNull().defaultNow(),
}, (t) => [
  index('idx_attendees_account').on(t.accountId),
  uniqueIndex('idx_attendees_provider').on(t.accountId, t.providerId),
]);

export const oauthTokens = pgTable('oauth_tokens', {
  id: text('id').primaryKey().$defaultFn(() => `otk_${crypto.randomUUID().replaceAll('-', '')}`),
  accountId: text('account_id').notNull().references(() => accounts.id, { onDelete: 'cascade' }),
  provider: text('provider').notNull(),
  accessTokenEnc: text('access_token_enc').notNull(),
  refreshTokenEnc: text('refresh_token_enc'),
  tokenType: text('token_type').notNull().default('Bearer'),
  expiry: timestamp('expiry', { withTimezone: true }),
  scopes: jsonb('scopes').notNull().default([]),
  metadata: jsonb('metadata').notNull().default({}),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).notNull().defaultNow(),
}, (t) => [
  uniqueIndex('idx_oauth_provider').on(t.accountId, t.provider),
]);

export const calendars = pgTable('calendars', {
  id: text('id').primaryKey().$defaultFn(() => `cal_${crypto.randomUUID().replaceAll('-', '')}`),
  accountId: text('account_id').notNull().references(() => accounts.id, { onDelete: 'cascade' }),
  provider: text('provider').notNull(),
  providerId: text('provider_id').notNull(),
  name: text('name').notNull(),
  color: text('color'),
  isPrimary: boolean('is_primary').notNull().default(false),
  isReadOnly: boolean('is_read_only').notNull().default(false),
  timezone: text('timezone'),
  metadata: jsonb('metadata').notNull().default({}),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).notNull().defaultNow(),
});

export const calendarEvents = pgTable('calendar_events', {
  id: text('id').primaryKey().$defaultFn(() => `evt_${crypto.randomUUID().replaceAll('-', '')}`),
  calendarId: text('calendar_id').notNull().references(() => calendars.id, { onDelete: 'cascade' }),
  accountId: text('account_id').notNull().references(() => accounts.id, { onDelete: 'cascade' }),
  provider: text('provider').notNull(),
  providerId: text('provider_id').notNull(),
  title: text('title').notNull(),
  description: text('description'),
  location: text('location'),
  startAt: timestamp('start_at', { withTimezone: true }).notNull(),
  endAt: timestamp('end_at', { withTimezone: true }).notNull(),
  allDay: boolean('all_day').notNull().default(false),
  status: text('status').notNull().default('CONFIRMED'),
  attendees: jsonb('attendees').notNull().default([]),
  reminders: jsonb('reminders').notNull().default([]),
  conferenceUrl: text('conference_url'),
  recurrence: text('recurrence'),
  metadata: jsonb('metadata').notNull().default({}),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).notNull().defaultNow(),
});

export const auditLog = pgTable('audit_log', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  organizationId: text('organization_id').notNull(),
  actorId: text('actor_id').notNull(),
  actorName: text('actor_name'),
  action: text('action').notNull(),
  resourceType: text('resource_type'),
  resourceId: text('resource_id'),
  detail: jsonb('detail').default({}),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
}, (t) => [
  index('idx_audit_org').on(t.organizationId),
  index('idx_audit_created').on(t.createdAt),
]);

// ═══════════════════════════════════════════
// Better Auth Tables (generated via `npx auth@latest generate`)
// Added here for reference — actual generation via CLI
// ═══════════════════════════════════════════
// user, session, account (BA), verification,
// organization, member, invitation, apikey
```

---

## Environment Variables (Target)

```bash
# Database
DATABASE_URL="postgresql://kevinhill@localhost:5432/ondapile"

# Server
PORT=3000
BASE_URL="http://localhost:3000"

# Auth
BETTER_AUTH_SECRET="random-32-byte-secret"
GITHUB_CLIENT_ID=""
GITHUB_CLIENT_SECRET=""

# Encryption
ENCRYPTION_KEY="32-byte-hex-key"

# Google OAuth (for Gmail/GCal provider connections)
GOOGLE_CLIENT_ID=""
GOOGLE_CLIENT_SECRET=""

# Microsoft OAuth (for Outlook provider connections)
MICROSOFT_CLIENT_ID=""
MICROSOFT_CLIENT_SECRET=""
MICROSOFT_TENANT_ID="common"

# SMTP (for auth emails — verification, password reset)
SMTP_HOST="localhost"
SMTP_PORT=1025

# WhatsApp
WA_DEVICE_STORE_PATH="./devices"

# Logging
LOG_LEVEL="info"
```

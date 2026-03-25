# Phase 2: Backend Architecture

> **Status:** Draft
> **Date:** 2026-03-25
> **Source:** Codebase audit of `internal/`, `cmd/ondapile/main.go`, PRD.md

---

## Overview

Ondapile is a **single Go binary monolith** with clearly separated internal packages. There is no microservice boundary — all capabilities run in one process. The architecture follows a layered pattern:

```
HTTP Request → Gin Router → Middleware (auth, rate limit, CORS)
  → Handler (request parsing, validation)
    → Store (PostgreSQL queries) or Adapter (provider API calls)
      → Webhook Dispatcher (async event delivery)
```

The monolith is appropriate for v1: single deployment target, shared database, no inter-service communication overhead. The internal package boundaries are clean enough to extract later if needed.

---

## Capability Requirements

### Capability: HTTP API Server
- **Purpose:** Serve REST API endpoints for all platform operations
- **Characteristics:** Synchronous request/response, JSON payloads, stateless (session lives in frontend)
- **Consumers:** Frontend dashboard (via browser), Developer SDK/API (via HTTP client)
- **Data dependencies:** Reads/writes all PostgreSQL tables
- **Scale expectations:** 10s of concurrent users (self-hosted), 10 req/s sustained with 100 burst
- **Status:** ✅ Done — Gin framework with 49+ routes

### Capability: Provider Adapter System
- **Purpose:** Abstract provider-specific APIs (Gmail, Outlook, IMAP, WhatsApp, etc.) behind a unified interface
- **Characteristics:** Sync per-request (API calls to external providers), provider-specific error handling, credential-based auth
- **Consumers:** API handlers (emails, chats, messages, calendars, accounts)
- **Data dependencies:** Reads encrypted credentials from `accounts` table, reads/writes OAuth tokens
- **Scale expectations:** Bounded by external API rate limits (Gmail: 250 quota units/s, Microsoft Graph: 10K/10min)
- **Status:** ⚠️ Partial — IMAP mostly done, Gmail/Outlook stubbed, WhatsApp/LinkedIn/Instagram/Telegram exist but not v1 scope

### Capability: Authentication & Authorization
- **Purpose:** Verify API requests come from authorized users/API keys, scope data to organizations
- **Characteristics:** Sync middleware, runs on every `/api/v1/*` request
- **Consumers:** All API endpoints
- **Data dependencies:** `api_key` table (Better Auth plugin, SHA-256 hashed), static env var fallback
- **Scale expectations:** Every request — must be fast (DB lookup + hash compare)
- **Status:** ✅ Done — DualAuthMiddleware (Better Auth DB key → static key fallback)

### Capability: Webhook Event Delivery
- **Purpose:** Push real-time notifications to publisher's servers when events occur (email received, account connected, etc.)
- **Characteristics:** Async (goroutine-per-delivery), fire-and-forget from handler perspective, HMAC-SHA256 signed payloads
- **Consumers:** Webhook Consumer (Actor 6) — publisher's backend server
- **Data dependencies:** `webhooks` table (endpoint URLs, event subscriptions, secrets), `webhook_deliveries` table (delivery tracking, retry state)
- **Scale expectations:** 24 event types, delivery within 5s, 3 retries with exponential backoff (10s/60s/5min)
- **Status:** ✅ Done — PostgreSQL-backed queue, async goroutine delivery, retry loop

### Capability: Credential Encryption
- **Purpose:** Encrypt provider credentials (IMAP passwords, OAuth tokens) at rest
- **Characteristics:** Sync encryption/decryption on account create/connect/reconnect
- **Consumers:** Account handler, provider adapters on reconnect
- **Data dependencies:** `accounts.credentials_enc` column, `ONDAPILE_ENCRYPTION_KEY` env var
- **Scale expectations:** Per-account operation, not high frequency
- **Status:** ✅ Done — AES-256-GCM in `internal/config/crypto.go`

### Capability: Email Tracking
- **Purpose:** Track email opens (pixel) and link clicks (redirect) for sent emails
- **Characteristics:** Public HTTP endpoints (no auth), sub-100ms response time, fires webhook events
- **Consumers:** Email recipients (unknowingly), publishers via `email_opened`/`link_clicked` webhooks
- **Data dependencies:** Email tracking records, webhook dispatcher
- **Scale expectations:** Could be high volume if many emails tracked — public endpoints, no rate limiting currently
- **Status:** ✅ Routes exist — `GET /t/:id` (pixel), `GET /l/:id` (link redirect)

### Capability: Account Auto-Reconnect
- **Purpose:** On server startup, reconnect all previously OPERATIONAL accounts to their providers
- **Characteristics:** Async (goroutine per account), runs once at startup
- **Consumers:** Internal — ensures accounts don't go stale on restart
- **Data dependencies:** `accounts` table (status, encrypted credentials)
- **Scale expectations:** Bounded by number of connected accounts (10s-100s)
- **Status:** ✅ Done — `reconnectAccounts()` in `main.go`

### Capability: Database Migrations
- **Purpose:** Apply schema changes to PostgreSQL on startup
- **Characteristics:** Sync, runs before server starts listening
- **Consumers:** Internal
- **Data dependencies:** `migrations/*.sql` files
- **Scale expectations:** One-time per deployment
- **Status:** ✅ Done — sequential file-based migrations in `main.go`

---

## Service Boundaries

Ondapile is a monolith, but internally it has clear package boundaries that function as logical modules:

### Module: API Layer (`internal/api/`)
- **Responsibility:** HTTP request handling, input validation, response formatting, error mapping
- **Files:** 18 Go files (accounts, chats, messages, emails, webhooks, calendars, search, audit_log, hosted_auth, oauth_callback, health, metrics, middleware, errors, router)
- **Capabilities provided:** HTTP API Server, Authentication
- **Communication style:** Direct function calls to Store and Adapter modules
- **Failure mode:** Returns HTTP error responses (400/401/403/404/500)

### Module: Adapter Layer (`internal/adapter/` + provider packages)
- **Responsibility:** Provider abstraction — translates unified API calls to provider-specific API calls
- **Files:** 2 core files (adapter.go, registry.go) + 30 provider-specific files across 8 packages
- **Sub-modules:**
  | Package | Provider | v1 Scope | Status |
  |---------|----------|----------|--------|
  | `internal/email/` | IMAP/SMTP | ✅ Yes | ⚠️ Mostly done |
  | `internal/gmail/` | Gmail API | ✅ Yes | ⚠️ Stubbed |
  | `internal/outlook/` | Microsoft Graph | ✅ Yes | ❌ Missing |
  | `internal/whatsapp/` | WhatsApp Web | ❌ Post-v1 | ✅ Working |
  | `internal/linkedin/` | LinkedIn API | ❌ Post-v1 | ⚠️ Partial |
  | `internal/instagram/` | Instagram API | ❌ Post-v1 | ⚠️ Partial |
  | `internal/telegram/` | Telegram Bot API | ❌ Post-v1 | ⚠️ Partial |
  | `internal/gcal/` | Google Calendar | ❌ Post-v1 | ⚠️ Partial |
- **Capabilities provided:** Provider Adapter System
- **Communication style:** Called by API handlers, calls external provider APIs
- **Failure mode:** Returns `ErrNotSupported` or provider-specific errors, bubbled up as HTTP 500/502

### Module: Store Layer (`internal/store/`)
- **Responsibility:** All PostgreSQL interactions — CRUD operations, queries, pagination
- **Files:** 9 Go files (postgres.go, accounts.go, apikeys.go, audit_log.go, calendar_events.go, calendars.go, chats.go, messages.go, webhooks.go)
- **Capabilities provided:** Data persistence for all entities
- **Communication style:** Called by API handlers and adapters, executes SQL via `pgxpool`
- **Failure mode:** Returns Go errors, bubbled up as HTTP 500

### Module: Email Subsystem (`internal/email/`)
- **Responsibility:** IMAP client, SMTP client, email-specific storage, provider adapter for generic email
- **Files:** 4 Go files (adapter.go, imap_client.go, smtp_client.go, store.go)
- **Capabilities provided:** Email-specific CRUD, IMAP IDLE/polling, SMTP sending
- **Communication style:** Implements `adapter.Provider` interface, uses Store for persistence
- **Failure mode:** IMAP connection errors, SMTP delivery failures

### Module: Webhook Subsystem (`internal/webhook/`)
- **Responsibility:** Event dispatch, delivery tracking, retry with backoff, HMAC signing
- **Files:** 1 Go file (dispatcher.go)
- **Capabilities provided:** Webhook Event Delivery
- **Communication style:** Called by API handlers after state changes, async goroutine delivery
- **Failure mode:** HTTP delivery failures → retried 3x, logged, delivery marked as failed

### Module: OAuth Subsystem (`internal/oauth/`)
- **Responsibility:** OAuth token storage, token refresh, hosted auth flow management
- **Files:** 3 Go files (handler.go, source.go, store.go)
- **Capabilities provided:** OAuth flow for Gmail, Outlook, LinkedIn, Instagram
- **Communication style:** Called during account connection and token refresh
- **Failure mode:** Token expiry → auto-refresh or account status → CREDENTIALS_NEEDED

### Module: Tracking Subsystem (`internal/tracking/`)
- **Responsibility:** Email open/click tracking via pixel and link redirect
- **Files:** 4 Go files (tracker.go, pixel.go, links.go, store.go)
- **Capabilities provided:** Email Tracking
- **Communication style:** Public HTTP handlers, fires webhook events via dispatcher
- **Failure mode:** Silent failure — tracking pixel returns 1x1 GIF regardless

### Module: Config & Crypto (`internal/config/`, `internal/crypto/`)
- **Responsibility:** Environment variable loading, credential encryption/decryption
- **Files:** 3 Go files (config.go, crypto.go, message_encryption.go)
- **Capabilities provided:** Credential Encryption, configuration management
- **Communication style:** Called at startup and during credential operations
- **Failure mode:** Missing env vars → startup failure, bad encryption key → decrypt failure

### Module: Domain Models (`internal/model/`)
- **Responsibility:** Shared data structures used across all modules
- **Files:** 8 Go files (account.go, attendee.go, calendar.go, calendar_event.go, chat.go, email.go, message.go, webhook.go, pagination.go)
- **Capabilities provided:** Type definitions, no business logic
- **Communication style:** Imported by all other modules
- **Failure mode:** N/A — pure data structures

---

## API Surface

### Accounts (6 endpoints)
- **Actors:** SaaS Publisher, Developer (via SDK)
- **Operations:** List, Create (connect), Get, Delete (disconnect), Reconnect, Solve Checkpoint
- **Auth requirements:** API key (DualAuthMiddleware)
- **Rate sensitivity:** Low volume — account management is infrequent

### Emails (9 endpoints)
- **Actors:** Developer (via SDK)
- **Operations:** List (with search `?q=`), Get, Send, Update (read/starred/folder), Delete, Download Attachment, Reply, Forward, List Folders
- **Auth requirements:** API key (DualAuthMiddleware)
- **Rate sensitivity:** Medium — bounded by provider API limits
- **v1 gaps:** Reply/Forward exist as routes but adapter implementations incomplete for Gmail/Outlook

### Webhooks (3 endpoints)
- **Actors:** SaaS Publisher, Developer
- **Operations:** List, Create, Delete
- **Auth requirements:** API key (DualAuthMiddleware)
- **Rate sensitivity:** Low — webhook CRUD is infrequent

### Chats (7 endpoints)
- **Actors:** Developer
- **Operations:** List, Create, Get, Update, Delete, List Messages, Send Message
- **Auth requirements:** API key (DualAuthMiddleware)
- **Rate sensitivity:** Medium for messaging providers
- **Note:** Not in v1 scope (messaging providers are post-v1)

### Messages (5 endpoints)
- **Actors:** Developer
- **Operations:** List, Get, Delete, Download Attachment, Add Reaction
- **Auth requirements:** API key (DualAuthMiddleware)
- **Note:** Not in v1 scope

### Attendees (5 endpoints)
- **Actors:** Developer
- **Operations:** List, Get, Get Avatar, List Chats, List Messages
- **Auth requirements:** API key (DualAuthMiddleware)
- **Note:** Not in v1 scope

### Calendars (7 endpoints)
- **Actors:** Developer
- **Operations:** List Calendars, Get Calendar, List Events, Create Event, Get Event, Update Event, Delete Event
- **Auth requirements:** API key (DualAuthMiddleware)
- **Note:** Not in v1 scope

### Hosted Auth (1 endpoint + OAuth callback)
- **Actors:** SaaS Publisher (generates link), End User (uses link)
- **Operations:** Create hosted auth URL
- **Auth requirements:** API key for creation; callback is public (browser redirect)
- **Rate sensitivity:** Low

### Search (1 endpoint)
- **Actors:** Developer
- **Operations:** Cross-provider search
- **Auth requirements:** API key (DualAuthMiddleware)
- **Note:** Embedding provider not yet injected

### Audit Log (1 endpoint)
- **Actors:** Org Admin
- **Operations:** List (paginated, org-scoped)
- **Auth requirements:** API key (DualAuthMiddleware)
- **Rate sensitivity:** Low

### Public Routes (no auth)
- **Health:** `GET /health` — liveness check
- **Metrics:** `GET /metrics` — system metrics (DB pool stats)
- **OAuth Success:** `GET /oauth/success` — static HTML page
- **WhatsApp QR:** `GET /wa/qr/:id` — QR code page
- **Tracking Pixel:** `GET /t/:id` — 1x1 transparent GIF
- **Link Redirect:** `GET /l/:id` — click tracking redirect
- **OAuth Callback:** `GET /api/v1/oauth/callback/:provider` — outside auth middleware

---

## System Interactions

### Request Flow: API Call (e.g., Send Email)

```
Developer SDK                 Ondapile Server                    External Provider
     │                              │                                  │
     │  POST /api/v1/emails         │                                  │
     │  X-API-KEY: xxx              │                                  │
     │─────────────────────────────>│                                  │
     │                              │                                  │
     │                 DualAuthMiddleware                               │
     │                 ├─ SHA-256 hash lookup in api_key table         │
     │                 └─ Falls back to static ONDAPILE_API_KEY        │
     │                              │                                  │
     │                    emailH.Send()                                │
     │                    ├─ Parse request body                        │
     │                    ├─ Lookup account by account_id              │
     │                    ├─ Get provider from adapter registry        │
     │                    ├─ Call provider.SendEmail()─────────────────>│
     │                    │                          │  SMTP/Gmail API  │
     │                    │                          │<─────────────────│
     │                    ├─ Store email in PostgreSQL                  │
     │                    ├─ dispatcher.Dispatch("email_sent", data)    │
     │                    │  └─ async goroutine → POST to webhook URL   │
     │                    └─ Return 200 + email JSON                   │
     │<─────────────────────────────│                                  │
```

### Request Flow: Hosted Auth (Account Connection)

```
Publisher App          Ondapile API          Frontend         Google OAuth        Ondapile Callback
     │                      │                   │                  │                    │
     │ POST /hosted-auth    │                   │                  │                    │
     │─────────────────────>│                   │                  │                    │
     │ { providers, ... }   │                   │                  │                    │
     │<─────────────────────│                   │                  │                    │
     │ { url: /connect/tok }│                   │                  │                    │
     │                      │                   │                  │                    │
     │  (embed URL in app)  │                   │                  │                    │
     │                      │                   │                  │                    │
End User clicks link ──────────────────────────>│                  │                    │
                            │     GET /connect/$token              │                    │
                            │                   │ Select Gmail     │                    │
                            │                   │─────────────────>│                    │
                            │                   │    OAuth consent │                    │
                            │                   │<─────────────────│                    │
                            │                   │  User grants     │                    │
                            │                   │─────────────────>│                    │
                            │                   │                  │  callback?code=xxx │
                            │                   │                  │───────────────────>│
                            │                   │                  │                    │
                            │          oauthH.Callback()           │                    │
                            │          ├─ Exchange code for tokens  │                    │
                            │          ├─ Encrypt & store tokens    │                    │
                            │          ├─ Create account (OPERATIONAL)                  │
                            │          ├─ Dispatch("account_connected")                 │
                            │          └─ Redirect → /oauth/success │                    │
```

### Startup Flow: Provider Registration & Reconnect

```
main.go starts
  ├─ Load config from env vars
  ├─ Connect to PostgreSQL
  ├─ Run migrations (5 SQL files, sequential)
  ├─ Initialize webhook dispatcher + start retry loop (goroutine)
  ├─ Register providers (conditional on config):
  │   ├─ WhatsApp (always)
  │   ├─ IMAP/Email (always)
  │   ├─ Gmail + GCal (if GOOGLE_CLIENT_ID set)
  │   ├─ Outlook (if MICROSOFT_CLIENT_ID set)
  │   ├─ LinkedIn (if LINKEDIN_CLIENT_ID set)
  │   ├─ Instagram (if INSTAGRAM_CLIENT_ID set)
  │   └─ Telegram (always)
  ├─ Auto-reconnect OPERATIONAL accounts (goroutine per account)
  │   ├─ Try encrypted credentials first
  │   └─ Fall back to adapter.Reconnect() (WhatsApp SQLite session)
  ├─ Start Gin HTTP server (goroutine)
  └─ Wait for SIGINT/SIGTERM → graceful shutdown
```

---

## Tensions & Observations

### What This Phase Implies for Data (Phase 3)

1. **Single PostgreSQL for everything** — Queue (webhook deliveries), storage (emails, accounts), auth (Better Auth tables), tracking (opens/clicks). No Redis despite it being listed in PRD §3 tech stack table. This simplifies deployment but may become a bottleneck if webhook delivery volume is high.

2. **No connection pooling tuning** — `pgxpool` is used with default config. For a self-hosted platform with multiple provider connections doing concurrent IMAP/API calls, pool sizing matters.

3. **Webhook delivery table grows unboundedly** — No retention policy on `webhook_deliveries`. At scale, this table needs pruning or partitioning.

4. **Email store is separate from main store** — `internal/email/store.go` has its own `EmailStore` type wrapping the same `*store.Store`. This creates two data access patterns for the same database.

### Architectural Decisions That Constrain

1. **Provider interface is maximalist** — Every provider must implement ALL methods (messaging, email, calendar, attendees, media). Providers that don't support a capability return `ErrNotSupported`. This means the interface has 25+ methods. Consider splitting into `EmailProvider`, `MessagingProvider`, `CalendarProvider` sub-interfaces.

2. **No request-scoped organization context** — The DualAuthMiddleware resolves the API key but the org_id propagation to handlers is implicit. Handlers must extract it from the middleware context. This works but makes multi-tenant scoping easy to forget.

3. **Adapter registry is global singleton** — `adapter.Register()` writes to a package-level map. This makes testing harder (shared global state) and prevents per-org provider configuration (e.g., different OAuth apps per org).

4. **Goroutine-per-delivery for webhooks** — Simple and works for low volume, but no backpressure mechanism. If 1000 webhooks fire simultaneously, that's 1000 goroutines + 1000 HTTP connections.

### v1 Backend Priorities (from PRD)

| Priority | What | Status | Phase |
|----------|------|--------|-------|
| 1 | Complete IMAP adapter (all 14 email ops) | ⚠️ Partial | A |
| 2 | Complete Gmail adapter | ⚠️ Stubbed | B |
| 3 | Complete Outlook adapter | ❌ Missing | C |
| 4 | Email tracking system | ✅ Routes exist | D |
| 5 | Webhook coverage for all email events | ⚠️ Partial | E |
| 6 | Integration tests | ❌ Missing | F |
| 7 | SDK updates (reply, forward, drafts, search) | ❌ Missing | G |

---

## Pages NOT Served by Backend (Confirmed)

The Go backend does NOT serve the frontend. The frontend is a separate TanStack Start app on port 3000. The only HTML the Go server renders is:
- `/oauth/success` — inline static HTML
- `/wa/qr/:id` — QR code page (HTML generated by Go handler)

# Ondapile v1 — Product Requirements Document

> **Version:** 1.0
> **Date:** 2026-03-25
> **Status:** Draft
> **Source:** Deep interview spec (19.3% ambiguity), architecture audit, codebase analysis
> **Scope:** Email API platform only — no messaging, calendar, billing, or inbox UI

---

## 1. Product Overview

### 1.1 What is Ondapile?

Ondapile is an open-source, self-hosted unified communication API platform — a clone of [Unipile](https://unipile.com/). It enables SaaS publishers to integrate email capabilities into their products via a REST API and Node.js SDK, without building provider-specific adapters themselves.

### 1.2 v1 Goal

Ship a working email API platform where a developer can:
1. Sign up and get API keys
2. Connect end-user email accounts (Gmail, Outlook, IMAP) via hosted auth
3. Manage the full email lifecycle (14 operations) through a REST API
4. Receive real-time webhook notifications for email events
5. Use the Node.js SDK for convenience

### 1.3 Who It's For

The platform operator is also the first customer (dogfooding). Ondapile will be consumed by a separate AI gateway product (Chatwoot-style) that uses the API to receive emails, process them through an AI agent pipeline, and respond.

### 1.4 What v1 is NOT

- No unified inbox UI (API-only)
- No billing/subscription (free, no Stripe)
- No messaging providers (WhatsApp, LinkedIn, Instagram, Telegram)
- No calendar integration
- No AI agent features
- No admin panel, developer docs site, or Python SDK
- No outreach sequences or white-label branding
- The AI gateway / Chatwoot-style response pipeline is a **separate product** that consumes ondapile API

---

## 2. Actors

> Full actor definitions: [`docs/actors/ACTORS.md`](./actors/ACTORS.md)

| # | Actor | v1 Relevance | Summary |
|---|-------|-------------|---------|
| 1 | **Platform Operator** | High | Deploys ondapile, configures OAuth credentials, monitors health |
| 2 | **SaaS Publisher** | High | Signs up, gets API keys, integrates email into their product |
| 3 | **Developer** | High | Uses SDK/API to build the integration |
| 4 | **End User** | High | Connects their email account via hosted auth flow |
| 5 | **Org Admin** | Low | Team management exists but not critical path for v1 |
| 6 | **Webhook Consumer** | High | Publisher's server receives real-time email events |

---

## 3. Architecture (Existing)

| Layer | Tech | Port |
|-------|------|------|
| Frontend | TanStack Start (React 19) + Vite + shadcn/ui | 3000 |
| Auth | Better Auth (org, admin, apiKey plugins) | via frontend SSR |
| Backend API | Go + Gin | 8080 |
| Database | PostgreSQL (pgx/v5, raw SQL) + pgvector | 5432 |
| Cache | Redis | 6380 |
| SDK | Node.js (`sdk/node/`) | npm package |

### 3.1 Key Architectural Patterns

- **Provider Adapter Interface:** All providers implement `adapter.Provider` in `internal/adapter/adapter.go`. Email methods: `SendEmail`, `ListEmails`, `GetEmail`.
- **DualAuthMiddleware:** Tries Better Auth DB-backed API key lookup (SHA-256 hashed), falls back to static `ONDAPILE_API_KEY`. Better Auth keys carry `organization_id` for multi-tenant scoping.
- **Webhook Dispatcher:** PostgreSQL-backed queue (no Redis). Async goroutine-per-webhook delivery with HMAC-SHA256 signing. 3-level exponential backoff (10s/60s/5min). Polling retry loop every 30s.
- **Credential Encryption:** AES-256-GCM with key derived from `ONDAPILE_ENCRYPTION_KEY` env var.
- **Frontend as Admin Dashboard:** The frontend is a platform management dashboard, NOT an email client. No inbox/chat/message UI exists.

---

## 4. v1 Email Operations (14 Required)

All 14 operations must work across all 3 providers (Gmail, Outlook, IMAP).

| # | Operation | Endpoint | Webhook Events |
|---|-----------|----------|----------------|
| 1 | List inbox emails | `GET /api/v1/emails` | — |
| 2 | Read full email | `GET /api/v1/emails/:id` | — |
| 3 | Send email | `POST /api/v1/emails` | `email_sent` |
| 4 | Reply / forward | `POST /api/v1/emails/:id/reply`, `POST /api/v1/emails/:id/forward` | `email_sent` |
| 5 | Send attachments | `POST /api/v1/emails` (multipart) | `email_sent` |
| 6 | Download attachments | `GET /api/v1/emails/:id/attachments/:att_id` | — |
| 7 | Mark read / unread | `PUT /api/v1/emails/:id` | — |
| 8 | Star / flag | `PUT /api/v1/emails/:id` | — |
| 9 | Move to folder | `PUT /api/v1/emails/:id` | `email_moved` |
| 10 | List folders | `GET /api/v1/emails/folders` | — |
| 11 | Create / delete drafts | `POST /api/v1/drafts`, `DELETE /api/v1/drafts/:id` | — |
| 12 | Delete email | `DELETE /api/v1/emails/:id` | — |
| 13 | Search emails | `GET /api/v1/emails?q=search` | — |
| 14 | Email tracking | Pixel tracking + link wrapping | `email_opened`, `link_clicked` |

---

## 5. Current Implementation Status

### 5.1 Platform Operations (Done)

| Operation | Status | Location |
|-----------|--------|----------|
| Sign up (email/password + GitHub OAuth) | ✅ Done | Better Auth routes |
| Auto-create organization on signup | ✅ Done | `databaseHooks.user.create.after` |
| Create/list/revoke API keys | ✅ Done | Better Auth apiKey plugin |
| Generate hosted auth link | ✅ Done | `POST /api/v1/accounts/hosted-auth` |
| OAuth callback handling | ✅ Done | `GET /api/v1/oauth/callback/:provider` |
| List connected accounts | ✅ Done | `GET /api/v1/accounts` |
| Reconnect account | ✅ Done | `POST /api/v1/accounts/:id/reconnect` |
| Create/list/delete webhooks | ✅ Done | `GET/POST/DELETE /api/v1/webhooks` |
| Team management (invite/roles) | ✅ Done | Better Auth organization plugin |
| Audit log | ✅ Done | `GET /api/v1/audit-log` |
| Health + metrics | ✅ Done | `GET /health`, `GET /metrics` |
| Rate limiting | ✅ Done | 10 req/s sustained, 100 burst |
| HMAC-SHA256 webhook signatures | ✅ Done | `whsec_` prefixed secrets |
| AES-256-GCM credential encryption | ✅ Done | `internal/config/crypto.go` |

### 5.2 Email Handler Endpoints (Existing)

| Method | Path | Handler | Status |
|--------|------|---------|--------|
| `GET` | `/emails` | `emailH.List` | ✅ Implemented |
| `POST` | `/emails` | `emailH.Send` | ✅ Implemented |
| `GET` | `/emails/:id` | `emailH.Get` | ✅ Implemented |
| `PUT` | `/emails/:id` | `emailH.Update` | ✅ Implemented |
| `DELETE` | `/emails/:id` | `emailH.Delete` | ✅ Implemented |
| `GET` | `/emails/:id/attachments/:att_id` | `emailH.DownloadAttachment` | ⚠️ Returns 501 |
| `GET` | `/emails/folders` | `emailH.ListFolders` | ✅ Implemented |

### 5.3 Email Adapter Status (Per Provider)

| Capability | IMAP | Gmail | Outlook |
|------------|------|-------|---------|
| Connect/Auth | ✅ Direct creds | ✅ OAuth | ✅ OAuth |
| Disconnect/Reconnect | ✅ | ✅ | ✅ |
| Status | ✅ | ✅ | ✅ |
| **SendEmail** | ✅ | ⚠️ Stubbed | ❌ Missing |
| **ListEmails** | ✅ | ⚠️ Stubbed | ❌ Missing |
| **GetEmail** | ✅ | ⚠️ Stubbed | ❌ Missing |
| **ListFolders** | ✅ | ❌ Missing | ❌ Missing |
| **Update (read/folder)** | ❌ Missing | ❌ Missing | ❌ Missing |
| **Delete** | ❌ Missing | ❌ Missing | ❌ Missing |
| **DownloadAttachment** | ⚠️ Stubbed | ✅ | ✅ (helper only) |
| IDLE/polling (new email) | ✅ | ❌ Missing | ❌ Missing |

### 5.4 Email Persistence (Done)

`internal/email/store.go` — EmailStore with full CRUD:
- `StoreEmail`, `GetEmail`, `GetEmailByProviderID`, `ListEmails` (paginated, folder + cursor filtering)
- `UpdateEmailReadStatus`, `UpdateEmailFolder`, `UpdateEmail`, `DeleteEmail`
- `GetUnreadCount`, `GetFolderCounts`

### 5.5 IMAP Client (Done)

`internal/email/imap_client.go` — Full IMAP implementation:
- `ConnectIMAP`, `ListMailboxes`, `FetchMessages`, `FetchMessage`
- `SearchEmails`, `MarkAsSeen`, `DeleteMessage`, `MoveMessage`
- IDLE loop (stubbed/polling), full MIME parsing

### 5.6 Node SDK (Done)

`sdk/node/src/index.ts` — OndapileClient with sub-clients:
- `Emails`: list, get, send, update, delete, downloadAttachment, listFolders
- `Webhooks`: list, create, delete, verifySignature
- `Accounts`: list, get, create, delete, reconnect, hostedAuth
- Also: Chats, Messages, Attendees, Calendars (not v1 scope)

### 5.7 What's Missing (Gap Analysis)

| Gap | Category | Impact | Effort |
|-----|----------|--------|--------|
| Gmail adapter email methods stubbed | Adapter | 🔴 Blocker | Medium — Google API calls needed |
| Outlook adapter email methods missing | Adapter | 🔴 Blocker | Medium — Microsoft Graph API calls needed |
| Reply/forward endpoints don't exist | Handler | 🔴 Blocker | Low — new handler + adapter method |
| Draft endpoints don't exist | Handler | 🔴 Blocker | Low — new handler + adapter method |
| Email search endpoint doesn't exist | Handler | 🟡 High | Low — add `?q=` param to ListEmails |
| Attachment download returns 501 | Handler | 🟡 High | Low — wire to adapter method |
| Email tracking (pixel/link) not built | Feature | 🟡 High | Medium — new subsystem |
| Star/flag not in adapter interface | Adapter | 🟡 High | Low — add to Provider interface |
| IMAP update methods missing from adapter | Adapter | 🟡 High | Low — wire to imap_client methods |
| Gmail/Outlook new email polling | Adapter | 🟡 High | Medium — watch/push notifications |
| No CI/CD pipeline | Infra | 🟠 Medium | Low |
| Integration tests need real accounts | Testing | 🟠 Medium | Medium — test fixtures |

---

## 6. Implementation Plan

### Phase A: Complete IMAP Adapter (Foundation)

IMAP is the most complete adapter. Finish it first to establish the pattern for Gmail/Outlook.

| Task | File(s) | Description |
|------|---------|-------------|
| A.1 | `internal/email/adapter.go` | Wire `MarkAsSeen` → adapter `UpdateEmail` for read/unread |
| A.2 | `internal/email/adapter.go` | Wire `MoveMessage` → adapter `UpdateEmail` for folder moves |
| A.3 | `internal/email/adapter.go` | Wire `DeleteMessage` → adapter `DeleteEmail` |
| A.4 | `internal/email/adapter.go` | Implement `DownloadAttachment` (currently stubbed) |
| A.5 | `internal/email/adapter.go` | Implement `SearchEmails` via adapter (already in imap_client) |
| A.6 | `internal/adapter/adapter.go` | Add `StarEmail`, `ReplyEmail`, `ForwardEmail`, `CreateDraft`, `DeleteDraft`, `SearchEmails` to Provider interface |
| A.7 | `internal/api/emails.go` | Add reply/forward handlers: `POST /emails/:id/reply`, `POST /emails/:id/forward` |
| A.8 | `internal/api/drafts.go` | New handler: `POST /drafts`, `DELETE /drafts/:id` |
| A.9 | `internal/api/emails.go` | Wire `?q=` search param in List handler |
| A.10 | `internal/api/emails.go` | Fix DownloadAttachment to call adapter instead of returning 501 |
| A.11 | `internal/api/router.go` | Register new routes: reply, forward, drafts, search |

### Phase B: Complete Gmail Adapter

| Task | File(s) | Description |
|------|---------|-------------|
| B.1 | `internal/gmail/adapter.go` | Implement `SendEmail` using Gmail API `messages.send` |
| B.2 | `internal/gmail/adapter.go` | Implement `ListEmails` using Gmail API `messages.list` + `messages.get` |
| B.3 | `internal/gmail/adapter.go` | Implement `GetEmail` using Gmail API `messages.get` with full format |
| B.4 | `internal/gmail/adapter.go` | Implement `ListFolders` using Gmail API `labels.list` |
| B.5 | `internal/gmail/adapter.go` | Implement `UpdateEmail` (read/unread via `messages.modify`, move via label changes) |
| B.6 | `internal/gmail/adapter.go` | Implement `DeleteEmail` using Gmail API `messages.trash` |
| B.7 | `internal/gmail/adapter.go` | Implement `StarEmail` via Gmail label `STARRED` |
| B.8 | `internal/gmail/adapter.go` | Implement `ReplyEmail`, `ForwardEmail` (threading via `threadId` + `In-Reply-To` header) |
| B.9 | `internal/gmail/adapter.go` | Implement `CreateDraft`, `DeleteDraft` using Gmail API `drafts.*` |
| B.10 | `internal/gmail/adapter.go` | Implement `SearchEmails` using Gmail API `messages.list` with `q` param |
| B.11 | `internal/gmail/adapter.go` | Implement new email polling via Gmail API `history.list` or push notifications |

### Phase C: Complete Outlook Adapter

| Task | File(s) | Description |
|------|---------|-------------|
| C.1 | `internal/outlook/adapter.go` | Implement `SendEmail` using Microsoft Graph `sendMail` |
| C.2 | `internal/outlook/adapter.go` | Implement `ListEmails` using Graph `messages` endpoint |
| C.3 | `internal/outlook/adapter.go` | Implement `GetEmail` using Graph `messages/{id}` |
| C.4 | `internal/outlook/adapter.go` | Implement `ListFolders` using Graph `mailFolders` |
| C.5 | `internal/outlook/adapter.go` | Implement `UpdateEmail` (read via PATCH `isRead`, move via `move` action) |
| C.6 | `internal/outlook/adapter.go` | Implement `DeleteEmail` using Graph `messages/{id}` DELETE |
| C.7 | `internal/outlook/adapter.go` | Implement `StarEmail` via Graph `flag` property |
| C.8 | `internal/outlook/adapter.go` | Implement `ReplyEmail`, `ForwardEmail` using Graph `reply`/`forward` actions |
| C.9 | `internal/outlook/adapter.go` | Implement `CreateDraft`, `DeleteDraft` using Graph `messages` with `isDraft` |
| C.10 | `internal/outlook/adapter.go` | Implement `SearchEmails` using Graph `$search` or `$filter` |
| C.11 | `internal/outlook/adapter.go` | Implement new email polling via Graph `subscriptions` (webhooks) or delta query |

### Phase D: Email Tracking

| Task | File(s) | Description |
|------|---------|-------------|
| D.1 | `internal/tracking/` | New package: tracking pixel server (1x1 transparent GIF) |
| D.2 | `internal/tracking/` | Link wrapping/redirect service |
| D.3 | `internal/api/router.go` | Public routes: `GET /t/:id.gif` (open), `GET /l/:id` (click redirect) |
| D.4 | `internal/api/emails.go` | Inject tracking pixel + wrap links on send when `tracking: true` |
| D.5 | `internal/webhook/dispatcher.go` | Fire `email_opened` and `link_clicked` webhook events |
| D.6 | `internal/store/` | Tracking persistence (opens, clicks, timestamps) |

### Phase E: Webhook Coverage for Email Events

| Task | File(s) | Description |
|------|---------|-------------|
| E.1 | All email adapters | Fire `email_received` when new email detected (IDLE/poll/push) |
| E.2 | `internal/api/emails.go` | Fire `email_sent` after successful send/reply/forward |
| E.3 | `internal/api/emails.go` | Fire `email_moved` after folder move |
| E.4 | Webhook dispatcher | Verify all email event types are dispatchable |

### Phase F: Integration Tests

| Task | File(s) | Description |
|------|---------|-------------|
| F.1 | `tests/integration/email_imap_test.go` | Full 14-operation test suite against IMAP test account |
| F.2 | `tests/integration/email_gmail_test.go` | Full 14-operation test suite against Gmail test account |
| F.3 | `tests/integration/email_outlook_test.go` | Full 14-operation test suite against Outlook test account |
| F.4 | `tests/integration/webhook_email_test.go` | Verify all email webhook events fire correctly |
| F.5 | `tests/integration/tracking_test.go` | Verify open/click tracking fires webhooks |
| F.6 | `.github/workflows/ci.yml` | CI pipeline: lint, test, build |

### Phase G: SDK Updates

| Task | File(s) | Description |
|------|---------|-------------|
| G.1 | `sdk/node/src/emails.ts` | Add `reply`, `forward` methods |
| G.2 | `sdk/node/src/drafts.ts` | New DraftsClient: `create`, `delete` |
| G.3 | `sdk/node/src/emails.ts` | Add `search` method (passes `q` param) |
| G.4 | `sdk/node/src/types.ts` | Add tracking types, draft types |
| G.5 | `sdk/node/src/index.ts` | Export DraftsClient |

---

## 7. Acceptance Criteria

### 7.1 Platform (must pass before email operations)

- [ ] Developer can sign up with email/password
- [ ] Developer can sign up with GitHub OAuth
- [ ] Organization auto-created on signup
- [ ] Developer can create an API key with `full` permission
- [ ] API key authenticates requests to `/api/v1/*` endpoints
- [ ] Developer can create a webhook with a callback URL

### 7.2 Account Connection (per provider: Gmail, Outlook, IMAP)

- [ ] `POST /api/v1/accounts/hosted-auth` returns a hosted auth URL (Gmail, Outlook) or direct connect works (IMAP)
- [ ] End user can authenticate via the hosted auth flow
- [ ] Webhook `account_connected` fires on successful connection
- [ ] `GET /api/v1/accounts` shows the connected account with status `OPERATIONAL`
- [ ] `POST /api/v1/accounts/:id/reconnect` restores a disconnected account

### 7.3 Email Lifecycle (per provider: Gmail, Outlook, IMAP)

- [ ] `GET /api/v1/emails` returns paginated inbox with sender, subject, date, snippet
- [ ] `GET /api/v1/emails/:id` returns full email body (HTML + text), headers, metadata
- [ ] `POST /api/v1/emails` sends an email with to/cc/bcc/subject/body
- [ ] `POST /api/v1/emails/:id/reply` sends a reply preserving threading
- [ ] `POST /api/v1/emails/:id/forward` forwards an email
- [ ] `POST /api/v1/emails` with multipart body sends attachments
- [ ] `GET /api/v1/emails/:id/attachments/:att_id` downloads an attachment
- [ ] `PUT /api/v1/emails/:id` with `read: true/false` toggles read status
- [ ] `PUT /api/v1/emails/:id` with `starred: true/false` toggles star/flag
- [ ] `PUT /api/v1/emails/:id` with `folder_id` moves email to specified folder
- [ ] `GET /api/v1/emails/folders` returns all mailbox folders
- [ ] `POST /api/v1/drafts` creates a draft email
- [ ] `DELETE /api/v1/drafts/:id` deletes a draft
- [ ] `DELETE /api/v1/emails/:id` moves email to trash
- [ ] `GET /api/v1/emails?q=keyword` returns matching emails
- [ ] Webhook `email_received` fires when a new email arrives
- [ ] Webhook `email_sent` fires when an email is sent via API
- [ ] Webhook `email_opened` fires when recipient opens a tracked email
- [ ] Webhook `link_clicked` fires when recipient clicks a tracked link

### 7.4 Testing

- [ ] Automated integration tests exist for all acceptance criteria above
- [ ] Tests run against real test accounts (Gmail, Outlook, IMAP)
- [ ] Tests are executable in CI (GitHub Actions)
- [ ] All tests pass on a clean deployment

---

## 8. Data Model (Existing)

### 8.1 Email Entity

```
Email {
  id, account_id, provider, provider_id (message_id + thread_id)
  subject, body (HTML), body_plain
  from_attendee, to_attendees[], cc_attendees[], bcc_attendees[], reply_to_attendees[]
  date, has_attachments, attachments[]
  folders[], role (INBOX/SENT/DRAFTS/TRASH/SPAM/ARCHIVE/CUSTOM)
  read, read_date, is_complete
  headers[], tracking { opens, clicks, clicked_links[] }, metadata
}
```

### 8.2 Key Database Tables

| Table | Purpose |
|-------|---------|
| `accounts` | Connected provider accounts (id, org_id, provider, status, credentials_enc) |
| `emails` | Stored emails with full metadata |
| `webhooks` | Registered webhook endpoints (url, events[], secret) |
| `webhook_deliveries` | Delivery tracking with retry state |
| `oauth_tokens` | Encrypted OAuth tokens per account |
| `audit_log` | All API operations logged |
| Better Auth tables | user, session, account, verification, organization, member, api_key |

---

## 9. API Surface (v1)

### 9.1 Existing Routes (No Changes Needed)

```
GET    /health
GET    /metrics
GET    /api/v1/accounts
POST   /api/v1/accounts
GET    /api/v1/accounts/:id
DELETE /api/v1/accounts/:id
POST   /api/v1/accounts/:id/reconnect
POST   /api/v1/accounts/hosted-auth
GET    /api/v1/oauth/callback/:provider
GET    /api/v1/emails
POST   /api/v1/emails
GET    /api/v1/emails/:id
PUT    /api/v1/emails/:id
DELETE /api/v1/emails/:id
GET    /api/v1/emails/folders
GET    /api/v1/webhooks
POST   /api/v1/webhooks
DELETE /api/v1/webhooks/:id
GET    /api/v1/audit-log
```

### 9.2 New Routes (Must Build)

```
POST   /api/v1/emails/:id/reply        # Reply to email (threading)
POST   /api/v1/emails/:id/forward      # Forward email
GET    /api/v1/emails/:id/attachments/:att_id  # Fix: currently returns 501
POST   /api/v1/drafts                   # Create draft
DELETE /api/v1/drafts/:id               # Delete draft
GET    /t/:id.gif                       # Tracking pixel (public, no auth)
GET    /l/:id                           # Link redirect (public, no auth)
```

### 9.3 Modified Routes

```
GET    /api/v1/emails                   # Add ?q= search param support
PUT    /api/v1/emails/:id               # Add starred field support
```

---

## 10. Non-Functional Requirements

| Requirement | Target |
|-------------|--------|
| API response time (p95) | < 500ms |
| Webhook delivery (first attempt) | < 5s after event |
| Webhook retry attempts | 3 (10s, 60s, 5min backoff) |
| Rate limiting | 10 req/s sustained, 100 burst per IP |
| Credential encryption | AES-256-GCM |
| Webhook signing | HMAC-SHA256 |
| Multi-tenancy | Organization-scoped via API key |
| Test coverage | Integration tests for all 14 email operations × 3 providers |

---

## 11. Implementation Priority

```
Phase A: Complete IMAP Adapter         ← FIRST — establishes pattern
Phase B: Complete Gmail Adapter        ← SECOND — most common provider
Phase C: Complete Outlook Adapter      ← THIRD — mirrors Gmail pattern
Phase D: Email Tracking                ← FOURTH — new subsystem
Phase E: Webhook Coverage              ← FIFTH — wire events across all phases
Phase F: Integration Tests             ← SIXTH — validates everything
Phase G: SDK Updates                   ← SEVENTH — developer convenience
```

Phases A-C can be partially parallelized (adapter work is independent per provider). Phase E should be done incrementally alongside A-C. Phase G can start as soon as new API endpoints are stable.

---

## 12. Post-v1 Roadmap

| Phase | Feature | Priority |
|-------|---------|----------|
| v1.1 | WhatsApp messaging provider | High |
| v1.2 | LinkedIn messaging (wire existing stubs) | High |
| v2 | Calendar integration (GCal + Outlook Calendar) | Medium |
| v2 | Stripe billing with per-account pricing | Medium |
| v2 | Unified inbox UI | Medium |
| v3 | Telegram, Instagram messaging | Low |
| v3 | AI agent autonomous messaging | Low |
| v3 | Multi-channel outreach sequences | Low |
| v3 | Admin panel, developer docs, Python SDK | Low |

---

## 13. References

- **Deep Interview Spec:** [`.omc/specs/deep-interview-ondapile-v1.md`](../.omc/specs/deep-interview-ondapile-v1.md)
- **Actor Definitions:** [`docs/actors/ACTORS.md`](./actors/ACTORS.md)
- **Legacy Project Spec:** [`docs/PROJECT_SPEC.md`](./PROJECT_SPEC.md) (partially superseded by this PRD)
- **Unipile Reference:** [unipile.com](https://unipile.com/) | [developer.unipile.com](https://developer.unipile.com/)

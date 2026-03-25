# Ondapile Actor Definitions

> Reference document for PRD. Based on comprehensive analysis of [unipile.com](https://unipile.com/) product, API docs, and ondapile codebase audit.
>
> Created: 2026-03-25

---

## Actor Overview

| # | Actor | Type | Ondapile Status |
|---|-------|------|-----------------|
| 1 | Platform Operator | Human (self-hosted admin) | ⚠️ Partial |
| 2 | SaaS Publisher | Human (org/company) | ✅ Mostly implemented |
| 3 | Developer | Human (technical implementer) | ✅ SDK + API keys exist |
| 4 | End User | Human (account connector) | ✅ Hosted auth exists |
| 5 | Org Admin | Human (team manager) | ✅ Team mgmt works, 🔴 billing mocked |
| 6 | Webhook Consumer | System (publisher's server) | ✅ Dispatcher + SDK exist |

---

## Actor 1: Platform Operator (Self-Hosted Admin)

**Unipile equivalent:** Unipile itself (the company). In open-source, this is whoever deploys ondapile.

| Attribute | Detail |
|-----------|--------|
| **Who** | DevOps engineer or founder who deploys and runs the ondapile instance |
| **Goal** | Provide a unified communication API to their users or their own product |
| **Frequency** | Setup once, monitor ongoing |

### Workflows

1. **Deploy Instance** — Docker/bare-metal deployment, configure env vars, run migrations
2. **Configure OAuth Credentials** — Set up Google and Microsoft OAuth apps for email/calendar providers
3. **Set Up Provider Connections** — Configure WhatsApp bridge, IMAP servers, Telegram bot tokens
4. **Monitor Health** — Check `/health` and `/metrics` endpoints, view system logs
5. **Manage Billing Tiers** — Define plan limits (free: 2 accounts, starter: 10, pro: 50)
6. **Manage Users** — Cross-org user management, ban/suspend accounts

### Ondapile Status

| Capability | Status | Location |
|------------|--------|----------|
| Health endpoint | ✅ | `GET /health` |
| Metrics endpoint | ✅ | `GET /metrics` |
| OAuth credential store | ✅ | `internal/oauth/store.go` |
| Admin plugin loaded | ✅ | `auth.ts` → `admin()` plugin |
| Admin panel UI | ❌ Missing | No `/admin` routes exist |
| Instance config UI | ❌ Missing | All config via env vars only |
| Cross-org management | ❌ Missing | No superadmin role |

### Gaps

- No admin dashboard — requires direct DB access for platform management
- No superadmin role — can't manage orgs across the platform
- No instance-level config UI — everything via env vars

---

## Actor 2: SaaS Publisher (API Consumer Organization)

**Unipile equivalent:** The primary customer — a company integrating unified communications into their product.

| Attribute | Detail |
|-----------|--------|
| **Who** | CTO / engineering lead at an ATS, CRM, outreach tool, or AI agent company |
| **Goal** | Integrate unified messaging/email/calendar into their product without building provider adapters |
| **Frequency** | Ongoing after initial integration |

### Sub-Types

| Sub-Type | Industry | Primary Channels | Key Workflows |
|----------|----------|-------------------|---------------|
| ATS Publisher | Recruiting/HR | LinkedIn, Email, Calendar | Recruiter search → InMail → Interview scheduling |
| CRM Publisher | Sales | Email, LinkedIn, WhatsApp | Lead enrichment → Outreach → Follow-up sequences |
| Outreach Publisher | Sales/Marketing | Email, LinkedIn, WhatsApp | Multi-channel sequences → Track opens/clicks |
| AI Agent Publisher | Automation | All channels | Autonomous messaging → Webhook-driven workflows |
| No-Code Builder | Various | All channels | n8n/Make.com workflow automation |

### Workflows

1. **Onboarding** — Sign up → Create org → Get API keys → Read docs
2. **Integration** — Configure webhooks → Generate hosted auth links → Embed in their app
3. **Operations** — Monitor connected accounts → View usage → Manage billing
4. **Scaling** — Upgrade plan → Add team members → Add more connected accounts

### Ondapile Status

| Capability | Status | Location |
|------------|--------|----------|
| Sign up (email/password + GitHub) | ✅ | `auth.ts` |
| Auto-create org on signup | ✅ | `auth.ts` → `databaseHooks.user.create.after` |
| API key management | ✅ | `/dashboard/api-keys/` |
| Webhook management | ✅ | `/dashboard/webhooks/` |
| Hosted auth link generation | ✅ | `POST /api/v1/accounts/hosted-auth` |
| Connected account management | ✅ | `/dashboard/accounts/` |
| Billing/subscription | 🔴 Mocked | `/dashboard/settings/billing` (hardcoded data) |
| Usage monitoring | 🔴 Mocked | Hardcoded API call / webhook counts |

---

## Actor 3: Developer (Technical Implementer)

**Unipile equivalent:** The engineer at the SaaS Publisher who writes the integration code.

| Attribute | Detail |
|-----------|--------|
| **Who** | Backend / fullstack developer consuming the ondapile API |
| **Goal** | Ship a working integration with minimal friction |
| **Frequency** | Intensive during integration, occasional maintenance |

### Workflows

1. **Discovery** — Read docs → Explore API reference → Try Postman/curl examples
2. **Setup** — Install SDK → Configure API key → Set base URL
3. **First Integration** — Connect test account → Send first message → Receive webhook
4. **Production** — Error handling → Retry logic → Rate limit handling → Monitoring
5. **Debugging** — Check audit logs → Verify webhook signatures → Test account reconnection

### Ondapile Status

| Capability | Status | Location |
|------------|--------|----------|
| Node.js SDK | ✅ | `sdk/node/` |
| API key authentication | ✅ | `X-API-KEY` header → `apikey_middleware.go` |
| Webhook SDK client | ✅ | `sdk/node/src/webhooks.ts` |
| Signature verification | ✅ | `WebhooksClient.verifySignature()` |
| Developer docs site | ❌ Missing | No docs site |
| Python SDK | ❌ Missing | — |
| Postman collection | ❌ Missing | — |
| OpenAPI spec | ❌ Missing | — |

### API Surface (Available to Developer)

| Group | Endpoints | Key Operations |
|-------|-----------|----------------|
| Accounts | `/api/v1/accounts/*` | List, create, reconnect, delete, QR, checkpoint |
| Chats | `/api/v1/chats/*` | List, create, get messages, send message |
| Messages | `/api/v1/messages/*` | Get, delete, reactions, attachments |
| Emails | `/api/v1/emails/*` | List, send, get, update, delete, folders |
| Calendars | `/api/v1/calendars/*` | List, get, create/update/delete events |
| Webhooks | `/api/v1/webhooks/*` | List, create, delete |
| Search | `/api/v1/search` | Cross-provider search |
| Audit Log | `/api/v1/audit-log` | Activity history |

---

## Actor 4: End User (Account Connector)

**Unipile equivalent:** The person who connects their real LinkedIn/WhatsApp/Gmail account through the Publisher's app.

| Attribute | Detail |
|-----------|--------|
| **Who** | Recruiter, salesperson, support agent using the Publisher's product |
| **Goal** | Connect real accounts so the SaaS app can send/receive on their behalf |
| **Frequency** | One-time setup per provider, occasional reconnection |
| **Critical note** | This user **never sees ondapile directly** — they interact through the Publisher's white-labeled auth flow |

### Workflows

1. **Connect Account** — Click hosted auth link → Select provider → Authenticate → Account connected
2. **Reconnect Account** — Receive "reconnect needed" notification → Re-authenticate → Account restored
3. **Solve Checkpoint** — 2FA/OTP challenge → Enter code → Connection continues
4. **Disconnect Account** — Remove provider connection from the Publisher's app

### Authentication Methods by Provider

| Provider | Auth Method | UI Element |
|----------|-------------|------------|
| Gmail | Google OAuth 2.0 | OAuth consent screen |
| Google Calendar | Google OAuth 2.0 | OAuth consent screen |
| Outlook | Microsoft OAuth 2.0 | Microsoft login |
| Outlook Calendar | Microsoft OAuth 2.0 | Microsoft login |
| WhatsApp | QR Code scan | QR display page (`/wa/qr/:id`) |
| LinkedIn | Session/credentials | Username + password form |
| Instagram | Session/credentials | Username + password form |
| Telegram | Phone number / bot token | Phone input + OTP |

### Ondapile Status

| Capability | Status | Location |
|------------|--------|----------|
| Hosted auth link generation | ✅ | `POST /api/v1/accounts/hosted-auth` |
| OAuth callback handling | ✅ | `internal/api/oauth_callback.go` |
| WhatsApp QR page | ✅ | `/wa/qr/:id` route |
| Checkpoint solving (2FA) | ✅ | `POST /api/v1/accounts/:id/checkpoint` |
| Account reconnection | ✅ | `POST /api/v1/accounts/:id/reconnect` |
| White-label customization | ❌ Missing | No custom domain / branding support |

---

## Actor 5: Org Admin (Team Manager)

**Unipile equivalent:** The team lead who manages access, billing, and connected accounts for their organization.

| Attribute | Detail |
|-----------|--------|
| **Who** | Team lead or account owner at the SaaS Publisher |
| **Goal** | Manage team access, billing, connected accounts, and API keys |
| **Frequency** | Weekly — team changes, billing reviews, monitoring |

### Workflows

1. **Team Management** — Invite members → Assign roles → Remove members
2. **API Key Management** — Create keys → Set permissions → Rotate/revoke keys
3. **Billing Management** — Select plan → Add payment → View invoices → Monitor usage
4. **Monitoring** — View audit logs → Check connected account health → Review webhook delivery

### Roles & Permissions

| Role | Capabilities |
|------|-------------|
| `owner` | Full access — auto-assigned on org creation, cannot be removed |
| `admin` | Invite/remove members, change roles, manage API keys |
| `member` | Use API keys, view connected accounts |

### API Key Permissions

| Permission | Scope |
|------------|-------|
| `full` | All API operations |
| `read` | Read-only access |
| `email` | Email operations only |
| `calendar` | Calendar operations only |

### Ondapile Status

| Capability | Status | Location |
|------------|--------|----------|
| Invite team members | ✅ | `/dashboard/settings/team` |
| Change member roles | ✅ | Same |
| Remove members | ✅ | Same |
| Create/revoke API keys | ✅ | `/dashboard/api-keys/` |
| API key permissions | ✅ | `full`, `read`, `email`, `calendar` |
| View audit logs | ✅ | `/dashboard/logs/` |
| Billing/subscription | 🔴 Mocked | Hardcoded trial status, mock invoices |
| Usage dashboard | 🔴 Mocked | Hardcoded API call / webhook counts |
| Granular permission guards | ⚠️ Missing | No route-level permission checks (e.g., can `member` manage webhooks?) |

---

## Actor 6: Webhook Consumer (System Actor)

**Unipile equivalent:** The Publisher's backend server that receives real-time push notifications.

| Attribute | Detail |
|-----------|--------|
| **Who** | Publisher's backend server (not a human) |
| **Goal** | Receive and process real-time events for messages, emails, calendar, and account status |
| **Frequency** | Continuous — every message/email/event triggers a webhook |

### Event Taxonomy

| Category | Events |
|----------|--------|
| **Account Status** (8) | `account_connected`, `account_disconnected`, `account_ok`, `account_credentials`, `account_error`, `account_connecting`, `account_creation_success`, `account_reconnected`, `account_sync_success`, `account_deleted` |
| **Messaging** (7) | `message_received`, `message_sent`, `message_reaction`, `message_read`, `message_edited`, `message_deleted`, `message_delivered` |
| **Email** (5) | `email_received`, `email_sent`, `email_moved`, `email_opened`, `link_clicked` |
| **Calendar** (3) | `event_created`, `event_updated`, `event_deleted` |
| **Relationship** (1) | `invitation_accepted` |

### Reliability Contract

| Feature | Unipile Spec | Ondapile Status |
|---------|-------------|-----------------|
| Retry with exponential backoff | 5x retries | ⚠️ Unknown — needs verification |
| HMAC signature verification | ✅ | ✅ `WebhooksClient.verifySignature()` |
| Delivery SLA | < 500ms | ⚠️ Unknown |
| Log retention | 30 days | ⚠️ Unknown |

### Ondapile Status

| Capability | Status | Location |
|------------|--------|----------|
| Webhook dispatcher | ✅ | `webhook_dispatcher.go` |
| Webhook CRUD API | ✅ | `GET/POST/DELETE /api/v1/webhooks` |
| SDK webhook client | ✅ | `sdk/node/src/webhooks.ts` |
| Signature verification | ✅ | `WebhooksClient.verifySignature()` |
| Email tracking (opens/clicks) | ❌ Missing | No open/click tracking implementation |

---

## Actor Relationship Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    PLATFORM OPERATOR                     │
│            (deploys & manages ondapile)                  │
│  ┌─────────┐ ┌──────────┐ ┌────────┐ ┌──────────────┐  │
│  │ Deploy  │ │ OAuth    │ │ Monitor│ │ Billing Tiers│  │
│  │ Instance│ │ Creds    │ │ Health │ │ & Limits     │  │
│  └─────────┘ └──────────┘ └────────┘ └──────────────┘  │
└──────────────────────┬──────────────────────────────────┘
                       │ provides API
         ┌─────────────┼─────────────┐
         ▼             ▼             ▼
┌──────────────┐ ┌──────────┐ ┌──────────────┐
│ SaaS         │ │ SaaS     │ │ SaaS         │
│ Publisher A  │ │ Pub B    │ │ Publisher C  │
│ (ATS)        │ │ (CRM)    │ │ (AI Agent)   │
└──────┬───────┘ └────┬─────┘ └──────┬───────┘
       │              │              │
       │  ┌───────────┘              │
       │  │  Developers write code   │
       ▼  ▼                          ▼
┌──────────────┐              ┌──────────────┐
│  Developer   │              │  Developer   │
│  (integrates │              │  (integrates │
│   SDK/API)   │              │   SDK/API)   │
└──────┬───────┘              └──────┬───────┘
       │                             │
       │  Embeds hosted auth         │
       ▼                             ▼
┌──────────────┐              ┌──────────────┐
│  End Users   │              │  End Users   │
│  (connect    │              │  (connect    │
│   accounts)  │              │   accounts)  │
└──────────────┘              └──────────────┘
  Recruiter                     AI Agent
  connects                      connects
  LinkedIn                      WhatsApp

       ┌──────────────────────────┐
       │    Webhook Consumer      │
       │  (Publisher's server)    │
       │  Receives: 24 event     │
       │  types across messaging,│
       │  email, calendar,       │
       │  account status         │
       └──────────────────────────┘
```

---

## Gap Summary

| Gap | Actors Affected | Severity | Phase |
|-----|----------------|----------|-------|
| Billing mocked | Publisher, Org Admin, Operator | 🔴 Critical | Phase 1 |
| No admin panel | Operator | 🟡 High | Phase 5 |
| No developer docs | Developer | 🟡 High | New |
| LinkedIn messaging disconnected | End User, Publisher | 🟠 Medium | Phase 3 |
| Instagram normalize stubbed | End User, Publisher | 🟠 Medium | Phase 3 |
| Telegram messaging not wired | End User, Publisher | 🟠 Medium | Phase 3 |
| No email tracking (opens/clicks) | Publisher, Webhook Consumer | 🟠 Medium | Future |
| No Python SDK | Developer | 🟢 Low | Future |
| No LinkedIn Recruiter/InMail | Publisher (ATS sub-type) | 🟢 Low | Future |
| No white-label auth branding | End User, Publisher | 🟢 Low | Future |
| No granular permission guards | Org Admin | 🟢 Low | Future |
| No OpenAPI spec | Developer | 🟢 Low | Future |

---

## Sources

- **Unipile site scrape**: Homepage, pricing, features, use cases, auth-on-behalf, webhooks
- **Unipile API docs**: [developer.unipile.com](https://developer.unipile.com/) — full endpoint inventory, SDK reference
- **Unipile Node SDK**: [github.com/unipile/unipile-node-sdk](https://github.com/unipile/unipile-node-sdk) (MIT)
- **Ondapile codebase audit**: Auth config, route definitions, provider adapters, billing UI, webhook system
- **Competitor analysis**: Nylas, Merge.dev, Apideck, Unified.to, Nango

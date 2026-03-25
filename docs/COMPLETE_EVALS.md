# Ondapile Complete Eval Suite

> Every eval needed for the ENTIRE site to be considered complete — not just migration, not just current features, but the full product vision from PRD through post-v1 roadmap.
>
> Organized by: what works today (regression), what's being migrated (migration), what's missing (capability), and what's nice-to-have (enhancement).
>
> Created: 2026-03-25
> Sources: PRD.md §7, ACTORS.md §Gap Summary, user-flows.md, data.md §Tensions, architecture docs

---

## How This Document Works

**Four eval tiers:**

| Tier | Purpose | When to Run | Pass Criteria |
|------|---------|-------------|---------------|
| **Tier 0: Smoke** | Does the app boot? | Every commit | pass^3 = 100% |
| **Tier 1: Regression** | Does existing stuff still work? | Every commit | pass^3 = 100% |
| **Tier 2: Capability** | Does new stuff work? | Per-feature | pass@3 >= 90% |
| **Tier 3: Quality** | Is it production-ready? | Pre-release | avg >= 4/5 (model grader) |

**Each eval has:**
- **ID** — unique, sortable (e.g., `T0.1`, `T1.15`, `T2.3`)
- **Description** — what's being tested
- **Grader** — Code (deterministic) / Model (fuzzy) / Human (manual)
- **Command** — exact command to verify
- **Source** — which doc defines this requirement

---

## Tier 0: Smoke Tests (9 evals)

> Does the app start and respond? Run on every commit, every deploy.

| ID | Description | Grader | Command |
|----|-------------|--------|---------|
| T0.1 | App boots without crash | Code | `bun run build && bun run start` (exit 0) |
| T0.2 | TypeScript compiles clean | Code | `bunx tsc --noEmit` (exit 0) |
| T0.3 | Health endpoint returns ok | Code | `curl -sf localhost:3000/health \| jq -r .status` = "ok" |
| T0.4 | Database connection works | Code | `SELECT 1` via Drizzle succeeds |
| T0.5 | Better Auth handler responds | Code | `POST /api/auth/sign-in/email` returns 200 or 401 (not 500) |
| T0.6 | API v1 rejects unauthenticated | Code | `GET /api/v1/accounts` without auth = 401 |
| T0.7 | Tracking pixel serves GIF | Code | `GET /t/test` = 200, content-type=image/gif |
| T0.8 | Link redirect works | Code | `GET /l/test?url=https://example.com` = 302 |
| T0.9 | Unit tests pass | Code | `bun test` (exit 0) |

**Source:** Baseline platform requirements

---

## Tier 1: Regression Tests (62 evals)

> Does everything that works today continue to work? Run on every commit.

### 1A: Authentication (10 evals)

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T1.1 | Signup with email/password creates user | Code | PRD §7.1 |
| T1.2 | Signup with GitHub OAuth creates user | Code | PRD §7.1 |
| T1.3 | Organization auto-created on signup | Code | PRD §7.1 |
| T1.4 | API key creation returns sk_live_ prefix | Code | PRD §7.1 |
| T1.5 | API key authenticates to /api/v1/* | Code | PRD §7.1 |
| T1.6 | Invalid API key returns 401 | Code | PRD §7.1 |
| T1.7 | Missing auth returns 401 | Code | PRD §7.1 |
| T1.8 | X-API-Key header works | Code | EVALS.md §E1.2 |
| T1.9 | Bearer token works | Code | EVALS.md §E1.2 |
| T1.10 | Session persists across page reload | Code | FRONTEND_EVALS §E2.3 |

### 1B: Account Management (10 evals)

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T1.11 | POST /accounts creates IMAP account | Code | PRD §7.2 |
| T1.12 | GET /accounts returns account list | Code | PRD §5.1 |
| T1.13 | GET /accounts/:id returns single account | Code | PRD §5.1 |
| T1.14 | DELETE /accounts/:id removes account | Code | PRD §5.1 |
| T1.15 | POST /accounts/:id/reconnect works | Code | PRD §7.2 |
| T1.16 | POST /accounts/hosted-auth returns URL | Code | PRD §7.2 |
| T1.17 | OAuth callback creates account | Code | PRD §7.2 |
| T1.18 | Account credentials are encrypted in DB | Code | PRD §3.1 |
| T1.19 | GET /accounts/:id/qr returns QR PNG | Code | ACTORS §4 |
| T1.20 | POST /accounts/:id/checkpoint works | Code | ACTORS §4 |

### 1C: Email Operations (14 evals — one per PRD requirement)

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T1.21 | GET /emails?account_id=X returns list | Code | PRD §7.3 #1 |
| T1.22 | GET /emails/:id returns full email | Code | PRD §7.3 #2 |
| T1.23 | POST /emails sends email | Code | PRD §7.3 #3 |
| T1.24 | POST /emails/:id/reply preserves threading | Code | PRD §7.3 #4 |
| T1.25 | POST /emails/:id/forward works | Code | PRD §7.3 #4 |
| T1.26 | POST /emails with attachments works | Code | PRD §7.3 #5 |
| T1.27 | GET /emails/:id/attachments/:att_id downloads | Code | PRD §7.3 #6 |
| T1.28 | PUT /emails/:id with read=true toggles | Code | PRD §7.3 #7 |
| T1.29 | PUT /emails/:id with starred=true works | Code | PRD §7.3 #8 |
| T1.30 | PUT /emails/:id with folder moves | Code | PRD §7.3 #9 |
| T1.31 | GET /emails/folders returns folder list | Code | PRD §7.3 #10 |
| T1.32 | DELETE /emails/:id deletes | Code | PRD §7.3 #12 |
| T1.33 | GET /emails?q=keyword searches | Code | PRD §7.3 #13 |
| T1.34 | Tracking pixel records open | Code | PRD §7.3 #14 |

### 1D: Webhooks (8 evals)

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T1.35 | POST /webhooks creates with HMAC secret | Code | PRD §5.1 |
| T1.36 | GET /webhooks returns list | Code | PRD §5.1 |
| T1.37 | DELETE /webhooks/:id removes | Code | PRD §5.1 |
| T1.38 | Webhook fires email.sent on send | Code | PRD §7.3 |
| T1.39 | Webhook fires email.sent on reply | Code | PRD §7.3 |
| T1.40 | Webhook fires email.opened on pixel | Code | PRD §7.3 |
| T1.41 | Webhook fires email.clicked on link | Code | PRD §7.3 |
| T1.42 | Webhook signature is valid HMAC-SHA256 | Code | PRD §10 |

### 1E: Dashboard Pages (12 evals — Playwright)

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T1.43 | /auth/signup renders form | Playwright | Architecture §1 |
| T1.44 | /auth/login renders form | Playwright | Architecture §1 |
| T1.45 | /dashboard loads after auth | Playwright | Architecture §1 |
| T1.46 | /dashboard/accounts loads | Playwright | Architecture §1 |
| T1.47 | /dashboard/api-keys loads | Playwright | Architecture §1 |
| T1.48 | /dashboard/webhooks loads | Playwright | Architecture §1 |
| T1.49 | /dashboard/logs loads | Playwright | Architecture §1 |
| T1.50 | /dashboard/settings loads | Playwright | Architecture §1 |
| T1.51 | /dashboard/settings/team loads | Playwright | Architecture §1 |
| T1.52 | /dashboard/settings/billing loads | Playwright | Architecture §1 |
| T1.53 | /dashboard/settings/oauth loads | Playwright | Architecture §1 |
| T1.54 | /dashboard/settings/hosted-auth loads | Playwright | Architecture §1 |

### 1F: Non-Functional (8 evals)

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T1.55 | Rate limiting activates at burst limit | Code | PRD §10 |
| T1.56 | CORS headers present on OPTIONS | Code | Architecture §2 |
| T1.57 | Metrics endpoint returns DB stats | Code | PRD §5.1 |
| T1.58 | Audit log records operations | Code | PRD §5.1 |
| T1.59 | Multi-tenant: org A can't see org B data | Code | PRD §10 |
| T1.60 | Public routes work without auth | Code | Architecture §2 |
| T1.61 | Node SDK tests pass (27/27) | Code | PRD §5.6 |
| T1.62 | No console errors on dashboard navigation | Playwright | FRONTEND_EVALS §E7 |

---

## Tier 2: Capability Evals (48 evals)

> Features that need to be built. Each maps to a gap in the current product.

### 2A: Draft Endpoints (5 evals) — PRD §7.3 #11

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T2.1 | POST /drafts creates draft email | Code | PRD §9.2 |
| T2.2 | GET /drafts lists drafts | Code | PRD §9.2 |
| T2.3 | PUT /drafts/:id updates draft body | Code | user-flows §3c |
| T2.4 | POST /drafts/:id/send sends and removes from drafts | Code | user-flows §3c |
| T2.5 | DELETE /drafts/:id discards draft | Code | PRD §9.2 |

### 2B: Permission Enforcement (6 evals) — ACTORS §5, Architecture §1

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T2.6 | API key with `read` perm can't POST /emails | Code | ACTORS §5 |
| T2.7 | API key with `email` perm can't GET /chats | Code | ACTORS §5 |
| T2.8 | `member` role can view accounts but not create webhooks | Code | ACTORS §5 |
| T2.9 | `admin` role can invite members | Code | ACTORS §5 |
| T2.10 | `owner` role can delete organization | Code | ACTORS §5 |
| T2.11 | Revoked API key returns 401 | Code | ACTORS §5 |

### 2C: Billing (Stripe Integration) (7 evals) — ACTORS §Gap, PRD §12

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T2.12 | Stripe checkout creates subscription | Code | ACTORS §2 |
| T2.13 | Free plan limits to N connected accounts | Code | ACTORS §2 |
| T2.14 | Starter plan allows more accounts | Code | ACTORS §2 |
| T2.15 | Account creation blocked at plan limit | Code | ACTORS §2 |
| T2.16 | /dashboard/settings/billing shows real plan | Playwright | ACTORS §5 |
| T2.17 | Invoice history shows real invoices | Playwright | ACTORS §5 |
| T2.18 | Usage metering shows real API call counts | Playwright | ACTORS §5 |

### 2D: Admin Panel (5 evals) — ACTORS §1

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T2.19 | /admin route exists and loads | Playwright | ACTORS §1 |
| T2.20 | Admin can list all organizations | Code | ACTORS §1 |
| T2.21 | Admin can ban/suspend a user | Code | ACTORS §1 |
| T2.22 | Admin can view cross-org metrics | Code | ACTORS §1 |
| T2.23 | Superadmin role exists and restricts access | Code | ACTORS §1 |

### 2E: Developer Experience (6 evals) — ACTORS §3

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T2.24 | OpenAPI spec is generated and valid | Code | ACTORS §3 |
| T2.25 | API docs site renders at /docs | Playwright | ACTORS §3 |
| T2.26 | Python SDK can list accounts | Code | ACTORS §3 |
| T2.27 | Python SDK can send email | Code | ACTORS §3 |
| T2.28 | Postman collection imports without errors | Code | ACTORS §3 |
| T2.29 | SDK webhook verify works across Python + Node | Code | ACTORS §3 |

### 2F: Messaging Providers (8 evals) — PRD §12 (v1.1-v1.2)

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T2.30 | WhatsApp: connect account via QR | Code | ACTORS §4 |
| T2.31 | WhatsApp: send message returns 200 | Code | user-flows §3d |
| T2.32 | WhatsApp: receive message fires webhook | Code | user-flows §3e |
| T2.33 | LinkedIn: connect account | Code | ACTORS §4 |
| T2.34 | LinkedIn: send message returns 200 | Code | user-flows §3d |
| T2.35 | Telegram: connect account | Code | ACTORS §4 |
| T2.36 | Telegram: send message returns 200 | Code | user-flows §3d |
| T2.37 | Instagram: connect account | Code | ACTORS §4 |

### 2G: Calendar (4 evals) — PRD §12 (v2)

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T2.38 | GET /calendars returns calendar list | Code | PRD §12 |
| T2.39 | POST /calendars/:id/events creates event | Code | PRD §12 |
| T2.40 | Google Calendar: list events works | Code | PRD §12 |
| T2.41 | Outlook Calendar: list events works | Code | PRD §12 |

### 2H: Data Quality (4 evals) — data.md §Tensions

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T2.42 | Webhook deliveries pruned after 30 days | Code | data.md §Tension 3 |
| T2.43 | Audit log has retention policy | Code | data.md §Tension 4 |
| T2.44 | Email thread_id grouping works | Code | user-flows §3c |
| T2.45 | Cursor-based pagination on all list endpoints | Code | user-flows §3c |

### 2I: White-Label & Branding (3 evals) — ACTORS §4

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T2.46 | Hosted auth page shows Publisher's branding | Playwright | ACTORS §4 |
| T2.47 | Custom domain for auth (auth.myapp.com) | Code | ACTORS §4 |
| T2.48 | OAuth consent shows Publisher app name | Human | ACTORS §4 |

---

## Tier 3: Quality Evals (12 evals — Model/Human Graded)

> Is the product production-ready? Run before any release.

| ID | Description | Grader | Rubric |
|----|-------------|--------|--------|
| T3.1 | API response shapes match Unipile format | Model | Compare 10 responses vs Unipile docs. Score 1-5. Pass: avg >= 4.5 |
| T3.2 | Error responses are consistent JSON format | Model | All errors have `{object, status, code, message}`. Score 1-5. Pass: all >= 4 |
| T3.3 | Webhook payloads are backward-compatible | Model | Compare 5 webhook payloads vs Go backend format. Score 1-5. Pass: all >= 4 |
| T3.4 | No `any` types in TypeScript (except JSONB) | Code | `grep -r ": any" src/ \| grep -v jsonb \| wc -l` = 0 |
| T3.5 | No unhandled promise rejections | Code | Run server for 60s with load, check for unhandled rejections |
| T3.6 | Drizzle queries use parameterized inputs | Code | No raw SQL string concatenation |
| T3.7 | All env vars documented in .env.example | Code | Every env var in code exists in .env.example |
| T3.8 | README has quickstart that works | Human | Follow README steps, server runs, first API call succeeds |
| T3.9 | Security: no credentials logged to stdout | Human | Send requests with credentials, check server output |
| T3.10 | Security: encrypted credentials not in API responses | Code | GET /accounts response has NO credentials_enc field |
| T3.11 | Performance: p95 response time < 500ms | Code | Run 1000 requests, measure p95. PRD §10 target |
| T3.12 | Performance: webhook delivery < 5s first attempt | Code | Trigger event, measure time to webhook callback. PRD §10 target |

---

## Eval → Document Traceability

Every eval traces back to a specific requirement:

| Document | Section | Eval IDs |
|----------|---------|----------|
| **PRD.md** | §7.1 Platform acceptance | T1.1–T1.6 |
| **PRD.md** | §7.2 Account connection | T1.11–T1.20 |
| **PRD.md** | §7.3 Email lifecycle (14 ops) | T1.21–T1.34 |
| **PRD.md** | §7.4 Testing | T0.9, T1.61 |
| **PRD.md** | §9.2 New routes | T2.1–T2.5 (drafts) |
| **PRD.md** | §10 Non-functional | T1.55, T1.56, T3.11, T3.12 |
| **PRD.md** | §12 Post-v1 roadmap | T2.30–T2.41 |
| **ACTORS.md** | §1 Platform Operator | T2.19–T2.23 |
| **ACTORS.md** | §2 SaaS Publisher | T2.12–T2.18 |
| **ACTORS.md** | §3 Developer | T2.24–T2.29 |
| **ACTORS.md** | §4 End User | T2.46–T2.48 |
| **ACTORS.md** | §5 Org Admin | T2.6–T2.11 |
| **ACTORS.md** | §Gap Summary | T2.1–T2.48 (all Tier 2) |
| **user-flows.md** | §3c Email ops (NOT YET BUILT) | T2.1–T2.5, T2.44, T2.45 |
| **user-flows.md** | §3d Messaging | T2.30–T2.37 |
| **data.md** | §Tensions | T2.42–T2.45 |
| **Architecture §1** | Frontend pages | T1.43–T1.54 |
| **Architecture §2** | Backend capabilities | T1.55–T1.60 |

---

## Summary

| Tier | Count | Description |
|------|-------|-------------|
| Tier 0: Smoke | 9 | App boots, compiles, basic endpoints |
| Tier 1: Regression | 62 | Everything that works today |
| Tier 2: Capability | 48 | Features that need building |
| Tier 3: Quality | 12 | Production readiness |
| **Total** | **131** | |

### By Grader Type

| Grader | Count |
|--------|-------|
| Code (deterministic) | 107 |
| Playwright (browser) | 17 |
| Model (LLM-judged) | 4 |
| Human (manual review) | 3 |

### By Build Phase

| Phase | Eval IDs | Count |
|-------|----------|-------|
| Current (working today) | T0.*, T1.* | 71 |
| Migration (Go → Hono) | Migration evals (separate doc) | 57 |
| v1 completion (drafts, perms) | T2.1–T2.11 | 11 |
| Billing (Stripe) | T2.12–T2.18 | 7 |
| Admin panel | T2.19–T2.23 | 5 |
| Developer experience | T2.24–T2.29 | 6 |
| Messaging (v1.1-v1.2) | T2.30–T2.37 | 8 |
| Calendar (v2) | T2.38–T2.41 | 4 |
| Data quality | T2.42–T2.45 | 4 |
| White-label (future) | T2.46–T2.48 | 3 |
| Quality gate (pre-release) | T3.* | 12 |

### 2J: Webhook Completeness (4 evals) — Oracle/Metis critique

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T2.49 | Reply fires email_sent webhook | Code | PRD §7.3 |
| T2.50 | Forward fires email_sent webhook | Code | PRD §7.3 |
| T2.51 | Folder move fires email_moved webhook | Code | PRD §7.3 |
| T2.52 | IMAP polling → email_received webhook | Code | PRD §7.3 |

### 2K: Hosted Auth UX (3 evals) — Oracle critique

| ID | Description | Grader | Source |
|----|-------------|--------|--------|
| T2.53 | /connect/$token renders provider selection | Playwright | Architecture §1 |
| T2.54 | /oauth/success renders close-window message | Playwright | Architecture §1 |
| T2.55 | Admin routes respond (not 404/500) | Code | Oracle critique |
---

## Running Evals

```bash
# Tier 0: Smoke (30 seconds)
bun test --grep "T0"

# Tier 1: Regression (2 minutes)
bun test --grep "T1"
bunx playwright test --grep "T1"

# Tier 2: Capability — per feature
bun test --grep "T2.A"  # Drafts
bun test --grep "T2.B"  # Permissions
bun test --grep "T2.C"  # Billing

# Tier 3: Quality (pre-release)
bun test --grep "T3"

# ALL evals
bun test && bunx playwright test
```

---

## Agent Execution Protocol

An AI agent building ondapile should:

1. **Before any work:** Run Tier 0 (smoke) + Tier 1 (regression) to confirm baseline
2. **Pick a feature:** Choose a T2.x capability eval group
3. **Implement:** Build until those T2.x evals pass
4. **Verify regression:** Re-run ALL Tier 0 + Tier 1 to confirm nothing broke
5. **Commit:** Only if Tier 0 + Tier 1 + targeted T2.x all pass
6. **Before release:** Run Tier 3 quality evals

The evals ARE the definition of done. If all 138 pass, the product is complete.

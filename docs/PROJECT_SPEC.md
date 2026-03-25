# Ondapile — Project Spec & Completion Roadmap

## Product Vision

Self-hosted Unipile alternative. A unified communication API that lets users:
1. Sign in (email/password or GitHub OAuth)
2. Get auto-assigned to an organization
3. Add billing (Stripe subscription, per-account pricing)
4. Create API keys
5. Connect provider accounts (Gmail, GCal, Outlook, WhatsApp, IMAP, LinkedIn, Instagram, Telegram)
6. Use the REST API to send/receive messages, emails, calendar events across all providers

---

## Architecture

| Layer | Tech | Port |
|-------|------|------|
| Frontend | TanStack Start (React 19) + Vite + shadcn/ui + TanStack Query v5 | 3000 |
| Auth | Better Auth (JS) with org, admin, apiKey, stripe plugins | via frontend SSR |
| Backend API | Go + Gin | 8080 |
| Database | PostgreSQL (pgx/v5, raw SQL) + pgvector | 5432 |
| Cache | Redis | 6380 |
| Email (dev) | Mailpit | 1025 |

---

## Current State (What's Done)

### ✅ Auth
- [x] Email/password signup + login
- [x] GitHub OAuth
- [x] Email verification flow
- [x] Password reset flow
- [x] Auto-create org + owner membership on signup (databaseHooks)
- [x] API key creation (sk_live_ prefix, org-scoped)
- [x] Admin plugin installed
- [x] Session cookie auth (tanstackStartCookies)

### ✅ Frontend Pages
- [x] /auth/login, /auth/signup, /auth/forgot-password, /auth/verify-email
- [x] /dashboard (index)
- [x] /dashboard/create-org
- [x] /dashboard/accounts (list + detail)
- [x] /dashboard/api-keys (create, list, revoke)
- [x] /dashboard/webhooks (list + detail)
- [x] /dashboard/logs (audit log)
- [x] /dashboard/settings/billing (MOCK SHELL — no real billing)
- [x] /dashboard/settings/hosted-auth
- [x] /dashboard/settings/oauth
- [x] /dashboard/settings/team
- [x] /connect/$token (OAuth connect wizard)

### ✅ Backend API (Go)
- [x] Full CRUD: accounts, chats, messages, attendees, emails, calendars, webhooks
- [x] DualAuthMiddleware (API key OR session cookie)
- [x] Rate limiting (10 req/s sustained, 100 burst)
- [x] HMAC-SHA256 webhook signatures
- [x] AES-256-GCM credential encryption
- [x] OAuth callback handler + token persistence
- [x] Hosted auth endpoint
- [x] Semantic search (pgvector)
- [x] Audit logging

### ✅ Provider Adapters (Go)
- [x] **WhatsApp** — FULLY IMPLEMENTED (whatsmeow, QR flow, send/receive, media, events)
- [x] **Gmail** — FULLY IMPLEMENTED (OAuth, send/list/get email, attachments)
- [x] **Google Calendar** — FULLY IMPLEMENTED (calendar + event CRUD)
- [x] **Outlook** — FULLY IMPLEMENTED (email + calendar via Microsoft Graph)
- [x] **Email IMAP/SMTP** — FULLY IMPLEMENTED (go-imap/v2, go-mail)

### ⚠️ Partial Provider Adapters
- [ ] **LinkedIn** — OAuth works, messaging code EXISTS in messaging.go but NOT wired to adapter interface
- [ ] **Instagram** — API calls work, normalize.go is STUBBED (returns empty structs)
- [ ] **Telegram** — Connect/disconnect work, messaging NOT implemented, normalize stubbed

### ✅ Database (11 migrations)
- [x] accounts, chats, messages, webhooks, webhook_deliveries
- [x] attendees, emails, oauth_tokens
- [x] calendars, calendar_events
- [x] audit_log
- [x] pgvector for embeddings
- [x] Organization ID scoping (multi-tenant)
- [x] Better Auth tables (user, session, account, verification, organization, member, api_key)

### ✅ Infrastructure
- [x] Docker + docker-compose (redis + ondapile)
- [x] 27 Go integration tests
- [x] 35 shadcn/ui components

---

## What's Missing (Roadmap)

### PHASE 1: Billing (Critical Path)
**Goal**: Users can subscribe, manage billing, and features are gated by plan.
**Port from**: kriasoft/react-starter-kit `@better-auth/stripe` pattern

#### 1.1 Stripe Integration (Backend — Better Auth Plugin)

**What to do:**
- Install `@better-auth/stripe` package in frontend
- Add `stripe()` plugin to Better Auth server config (`frontend/src/lib/auth.ts`)
- Add `stripeClient()` plugin to Better Auth client config (`frontend/src/lib/auth-client.ts`)
- Create Stripe products + prices in Stripe Dashboard
- Configure plans: free (0 accounts), starter (10 accounts), pro (50 accounts)

**Files to modify:**
- `frontend/src/lib/auth.ts` — add stripe plugin with plan config
- `frontend/src/lib/auth-client.ts` — add stripeClient plugin
- `frontend/package.json` — add @better-auth/stripe, stripe deps

**Eval — PASS when:**
```bash
# 1. Stripe plugin loads without errors
curl -s http://localhost:3000/api/auth/ok | jq .status
# Expected: "ok"

# 2. Subscription table exists in database
psql ondapile -c "SELECT column_name FROM information_schema.columns WHERE table_name='subscription' ORDER BY ordinal_position;"
# Expected: id, plan, referenceId, stripeCustomerId, stripeSubscriptionId, status, periodStart, periodEnd, ...

# 3. Stripe webhook endpoint responds
curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/api/auth/stripe/webhook
# Expected: 400 (no signature) — NOT 404

# 4. Checkout session can be created (requires valid Stripe keys)
# Manual test: Click "Upgrade to Starter" → redirected to Stripe Checkout
```

#### 1.2 Database Migration for Subscriptions

**What to do:**
- Better Auth stripe plugin auto-creates the `subscription` table
- Verify table schema matches expected columns
- Add plan_limits config

**New file:**
- `frontend/src/lib/plans.ts` — plan definitions with account limits

```typescript
export const planLimits = {
  free: { accounts: 2 },
  starter: { accounts: 10 },
  pro: { accounts: 50 },
} as const;
export type PlanName = keyof typeof planLimits;
```

**Eval — PASS when:**
```bash
# subscription table has correct columns
psql ondapile -c "\d subscription"
# Expected: id, plan, reference_id, stripe_customer_id, stripe_subscription_id, status, period_start, period_end, ...
```

#### 1.3 Billing UI (Frontend)

**What to do:**
- Replace mock `billing.tsx` with real Stripe-integrated UI
- Show current plan, subscription status, period end date
- "Upgrade" buttons that create Stripe Checkout sessions
- "Manage Billing" button that opens Stripe Customer Portal
- Query subscription status via Better Auth client

**Files to modify:**
- `frontend/src/routes/dashboard/settings/billing.tsx` — replace entirely

**Eval — PASS when:**
```
# Manual browser test:
1. Navigate to /dashboard/settings/billing
2. Shows "Free plan" for users without subscription
3. "Upgrade to Starter" button → redirects to Stripe Checkout
4. After successful payment → shows "Starter plan (active)"
5. "Manage Billing" → opens Stripe Customer Portal
6. Cancel subscription → shows "Access until [date]"
```

#### 1.4 Account Limit Enforcement (Go Backend)

**What to do:**
- Go backend must check org's subscription plan before allowing account creation
- Query subscription table to get current plan + limits
- Count existing accounts for the org
- Reject account creation if limit exceeded

**Files to modify:**
- `internal/api/accounts.go` — add plan check in POST /accounts handler
- `internal/store/` — add subscription query function

**New migration:**
- None needed (subscription table managed by Better Auth)

**Eval — PASS when:**
```bash
# 1. Free plan user (2 account limit) tries to create 3rd account
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "X-API-Key: sk_live_..." \
  -H "Content-Type: application/json" \
  -d '{"provider":"IMAP","identifier":"test@example.com","credentials":{"host":"imap.example.com","port":993,"username":"test","password":"pass"}}'
# Expected: 403 with message "Account limit reached for your plan. Upgrade to add more accounts."

# 2. After upgrading to starter (10 limit), same request succeeds
# Expected: 201 Created
```

#### 1.5 Environment Variables

**What to add to .env / docker-compose:**
```env
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_PUBLISHABLE_KEY=pk_test_...
STRIPE_STARTER_PRICE_ID=price_...
STRIPE_PRO_PRICE_ID=price_...
```

---

### PHASE 2: Google OAuth for Provider Connections (Critical Path)

**Goal**: Users can connect Gmail and Google Calendar accounts via OAuth.

#### 2.1 Google OAuth in Provider Connection Flow

**What to do:**
- Google OAuth is already partially set up (oauth_callback.go maps "google" → "GMAIL")
- Need: Google Cloud Console project with Gmail + Calendar API scopes enabled
- Need: OAuth consent screen configured
- Frontend OAuth settings page already exists at /dashboard/settings/oauth

**This is primarily a configuration task, not code:**
1. Create Google Cloud project
2. Enable Gmail API + Calendar API
3. Create OAuth 2.0 credentials
4. Add scopes: `gmail.readonly`, `gmail.send`, `gmail.modify`, `calendar.readonly`, `calendar.events`
5. Set redirect URI: `http://localhost:8080/api/v1/oauth/callback/google`
6. Enter Client ID + Secret in /dashboard/settings/oauth

**Eval — PASS when:**
```bash
# 1. Hosted auth returns a Google OAuth URL
curl -X POST http://localhost:8080/api/v1/accounts/hosted-auth \
  -H "X-API-Key: sk_live_..." \
  -H "Content-Type: application/json" \
  -d '{"provider":"GMAIL"}'
# Expected: 200 with {"url": "https://accounts.google.com/o/oauth2/v2/auth?..."}

# 2. After OAuth flow completes, account appears in list
curl http://localhost:8080/api/v1/accounts -H "X-API-Key: sk_live_..."
# Expected: Account with provider "GMAIL" and status "connected"

# 3. Email list works
curl http://localhost:8080/api/v1/emails -H "X-API-Key: sk_live_..."
# Expected: 200 with array of emails
```

#### 2.2 Microsoft OAuth for Outlook

**Same pattern as Google:**
1. Azure AD app registration
2. Enable Mail.ReadWrite + Calendars.ReadWrite
3. Set redirect URI: `http://localhost:8080/api/v1/oauth/callback/microsoft`
4. Enter Client ID + Secret in /dashboard/settings/oauth

**Eval — PASS when:**
```bash
# Same as Gmail but with provider "OUTLOOK"
curl -X POST http://localhost:8080/api/v1/accounts/hosted-auth \
  -H "X-API-Key: sk_live_..." \
  -d '{"provider":"OUTLOOK"}'
# Expected: 200 with Microsoft OAuth URL
```

---

### PHASE 3: Provider Adapter Completion (High Priority)

#### 3.1 LinkedIn — Wire Up Messaging

**What to do:**
- `internal/linkedin/messaging.go` has working implementations of listConversations, getConversation, listMessages, sendMessage, createConversation
- `internal/linkedin/adapter.go` returns ErrNotSupported for all messaging methods
- Need to wire adapter methods to call the messaging.go functions

**Files to modify:**
- `internal/linkedin/adapter.go` — replace ErrNotSupported stubs with calls to messaging.go functions

**Eval — PASS when:**
```bash
# With a connected LinkedIn account:
curl http://localhost:8080/api/v1/chats?account_id=LINKEDIN_ACCOUNT_ID \
  -H "X-API-Key: sk_live_..."
# Expected: 200 with array of LinkedIn conversations (NOT "provider does not support this operation")
```

#### 3.2 Instagram — Complete Normalization

**What to do:**
- `internal/instagram/normalize.go` has stub functions returning empty structs
- Need to implement proper normalization that maps Instagram Graph API responses to model types
- Reference: gmail/normalize.go or outlook/normalize.go for patterns

**Files to modify:**
- `internal/instagram/normalize.go` — implement normalizeChat, normalizeMessage, normalizeAttendee

**Eval — PASS when:**
```bash
# With a connected Instagram account:
curl http://localhost:8080/api/v1/chats?account_id=INSTAGRAM_ACCOUNT_ID \
  -H "X-API-Key: sk_live_..."
# Expected: 200 with properly normalized chat objects (non-empty id, name, timestamps)
```

#### 3.3 Telegram — Implement Messaging

**What to do:**
- `internal/telegram/client.go` has a working Bot API HTTP client
- `internal/telegram/adapter.go` returns ErrNotSupported for all messaging
- `internal/telegram/normalize.go` is stubbed
- Need to implement: ListChats, GetChat, ListMessages, SendMessage using Telegram Bot API
- Need to implement normalize functions

**Files to modify:**
- `internal/telegram/adapter.go` — implement messaging methods using client.go
- `internal/telegram/normalize.go` — implement normalization

**Eval — PASS when:**
```bash
# With a connected Telegram bot account:
curl http://localhost:8080/api/v1/chats?account_id=TELEGRAM_ACCOUNT_ID \
  -H "X-API-Key: sk_live_..."
# Expected: 200 with Telegram chat list

curl -X POST http://localhost:8080/api/v1/chats/CHAT_ID/messages \
  -H "X-API-Key: sk_live_..." \
  -d '{"text":"Hello from API"}'
# Expected: 200 with sent message object
```

---

### PHASE 4: Production Readiness (High Priority)

#### 4.1 Production Email (Swap Mailpit → Resend)

**What to do:**
- Install `resend` npm package in frontend
- Replace nodemailer SMTP config with Resend API in `frontend/src/lib/auth.ts`
- Add RESEND_API_KEY env var

**Files to modify:**
- `frontend/src/lib/auth.ts` — replace nodemailer transport with Resend
- `frontend/package.json` — add resend package

**Eval — PASS when:**
```bash
# 1. Sign up with a real email → receive verification email in inbox (not Mailpit)
# 2. Reset password → receive reset email in inbox
# 3. Invite team member → receive invitation email in inbox
```

#### 4.2 Google OAuth for Sign-In (Not Provider Connection)

**What to do:**
- Add Google as a social provider in Better Auth config
- This is separate from Google OAuth for Gmail/GCal provider connections

**Files to modify:**
- `frontend/src/lib/auth.ts` — add google to socialProviders
- `frontend/src/routes/auth/login.tsx` — add "Sign in with Google" button
- `frontend/src/routes/auth/signup.tsx` — add "Sign up with Google" button

**Eval — PASS when:**
```
# Manual browser test:
1. Click "Sign in with Google" on login page
2. Redirected to Google consent screen
3. After consent → redirected to /dashboard
4. Session is active, user profile shows Google name + avatar
```

---

### PHASE 5: Dashboard Polish (Medium Priority)

#### 5.1 Admin Panel UI

**What to port from**: zexahq/better-auth-starter admin components

**What to do:**
- Create /dashboard/admin route
- User management table (list, ban/unban, delete, role assignment)
- Uses Better Auth admin plugin (already installed)

**New files:**
- `frontend/src/routes/dashboard/admin/index.tsx`
- `frontend/src/routes/dashboard/admin/users.tsx`

**Eval — PASS when:**
```
# Manual browser test:
1. Admin user navigates to /dashboard/admin/users
2. Sees paginated user list with name, email, role, status
3. Can ban a user → user cannot log in
4. Can unban → user can log in again
5. Can change role (user ↔ admin)
6. Non-admin users get 403 on /dashboard/admin
```

#### 5.2 Team/Organization Management

**What to do:**
- /dashboard/settings/team page should list org members
- Invite new members via email
- Remove members
- Change member roles

**Files to modify:**
- `frontend/src/routes/dashboard/settings/team.tsx` — verify/complete implementation

**Eval — PASS when:**
```
# Manual browser test:
1. Navigate to /dashboard/settings/team
2. See list of org members with roles
3. Invite new member by email → invitation sent
4. Accept invitation → new member appears
5. Change member role → reflected immediately
6. Remove member → member loses access
```

---

### PHASE 6: CI/CD & Testing (Medium Priority)

#### 6.1 GitHub Actions CI

**New file:** `.github/workflows/ci.yml`

```yaml
- lint (frontend ESLint + Go vet)
- type-check (tsc --noEmit)
- test-backend (go test ./tests/integration/...)
- test-frontend (vitest run)
- build (docker build)
```

#### 6.2 Frontend Testing Setup

**What to do:**
- Configure Vitest in frontend
- Add tests for auth flows, API key CRUD, billing page

**Eval — PASS when:**
```bash
cd frontend && npx vitest run
# Expected: All tests pass, exit code 0
```

---

## Testing Methods for AI Agents

### Automated Verification Commands

Every feature should be verifiable by running these commands:

```bash
# === Auth ===
# Sign up works
curl -s -X POST http://localhost:3000/api/auth/sign-up/email \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","email":"test@test.com","password":"Test1234!"}' | jq .user.id
# Expected: non-null user ID

# Sign in works
curl -s -X POST http://localhost:3000/api/auth/sign-in/email \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"Test1234!"}' -c cookies.txt | jq .user.id
# Expected: non-null user ID

# Session valid
curl -s http://localhost:3000/api/auth/get-session -b cookies.txt | jq .user.email
# Expected: "test@test.com"

# === Organizations ===
# User has org
curl -s http://localhost:3000/api/auth/organization/list -b cookies.txt | jq 'length'
# Expected: >= 1

# === API Keys ===
# Create API key
curl -s -X POST http://localhost:3000/api/auth/api-key/create \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name":"test-key"}' | jq .key
# Expected: starts with "sk_live_"

# List API keys
curl -s http://localhost:3000/api/auth/api-key/list -b cookies.txt | jq 'length'
# Expected: >= 1

# === Go Backend Health ===
curl -s http://localhost:8080/health | jq .status
# Expected: "ok"

# === Go Backend Auth ===
curl -s http://localhost:8080/api/v1/accounts \
  -H "X-API-Key: sk_live_YOUR_KEY" | jq 'length'
# Expected: 200 status, array response

# === Billing (Phase 1) ===
# Subscription table exists
psql ondapile -c "SELECT 1 FROM information_schema.tables WHERE table_name='subscription'" -t
# Expected: "1"

# Stripe webhook endpoint exists (not 404)
curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:3000/api/auth/stripe/webhook
# Expected: 400 (not 404)

# === Account Limits (Phase 1.4) ===
# Check plan for org
psql ondapile -c "SELECT plan, status FROM subscription WHERE reference_id = 'ORG_ID'" -t
# Expected: plan name + status

# === Provider Connections ===
# List connected accounts
curl -s http://localhost:8080/api/v1/accounts -H "X-API-Key: sk_live_..." | jq '.[].provider'
# Expected: array of provider names

# === Integration Test Suite ===
cd /Users/kevinhill/Coding/Projects/ondapile
CGO_ENABLED=1 go test ./tests/integration/... -v -count=1 2>&1 | tail -5
# Expected: "PASS" with exit code 0
```

### E2E Browser Test Script (Future)

```typescript
// Playwright test outline
test('full user journey', async ({ page }) => {
  // 1. Sign up
  await page.goto('/auth/signup');
  await page.fill('[name=email]', 'e2e@test.com');
  await page.fill('[name=password]', 'Test1234!');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/dashboard');

  // 2. Verify org exists
  await page.goto('/dashboard/settings/team');
  await expect(page.locator('text=owner')).toBeVisible();

  // 3. Create API key
  await page.goto('/dashboard/api-keys');
  await page.click('text=Create API Key');
  await page.fill('[name=name]', 'test-key');
  await page.click('text=Create');
  await expect(page.locator('text=sk_live_')).toBeVisible();

  // 4. Check billing page (after Phase 1)
  await page.goto('/dashboard/settings/billing');
  await expect(page.locator('text=Free plan')).toBeVisible();

  // 5. Connect account (after Phase 2)
  await page.goto('/dashboard/accounts');
  await page.click('text=Connect Account');
  // ... provider-specific flow
});
```

---

## Priority Order for Implementation

```
Phase 1: Billing (Stripe)          ← CRITICAL — revenue enabler
  1.1 Stripe plugin setup
  1.2 Subscription migration
  1.3 Billing UI
  1.4 Account limit enforcement
  1.5 Environment variables

Phase 2: Google/Microsoft OAuth    ← CRITICAL — enables core product
  2.1 Google OAuth (Gmail + GCal)
  2.2 Microsoft OAuth (Outlook)

Phase 3: Provider Completion       ← HIGH — full product coverage
  3.1 LinkedIn messaging wiring
  3.2 Instagram normalization
  3.3 Telegram messaging

Phase 4: Production Readiness      ← HIGH — launch requirements
  4.1 Production email (Resend)
  4.2 Google sign-in

Phase 5: Dashboard Polish          ← MEDIUM — nice to have
  5.1 Admin panel UI
  5.2 Team management

Phase 6: CI/CD & Testing          ← MEDIUM — developer experience
  6.1 GitHub Actions
  6.2 Frontend testing
```

---

## What to Port from Boilerplates

### From kriasoft/react-starter-kit:
1. **Stripe plugin pattern** — `@better-auth/stripe` configuration with plans, authorizeReference, and optional env var loading
2. **Billing tRPC router** — subscription query that returns plan + limits (adapt to Go API)
3. **Billing UI** — BillingCard component with upgrade/manage buttons
4. **Plan limits** — simple object mapping plan names to account limits
5. **Billing query options** — TanStack Query wrapper for subscription data

### From zexahq/better-auth-starter:
1. **Admin panel components** — users-table, user-actions, ban/unban/delete/role dialogs
2. **Admin API route** — paginated user listing with sort/filter
3. **Dashboard layout pattern** — admin sidebar with role-based nav

### NOT porting (incompatible or unnecessary):
- tRPC (we use Go REST API)
- Drizzle ORM (we use raw pgx SQL)
- Cloudflare Workers deployment (we use Docker)
- Astro marketing site (not needed yet)
- Jotai state management (we use TanStack Query)
- Next.js App Router patterns (we use TanStack Start)

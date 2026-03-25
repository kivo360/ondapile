# Phase 1: Frontend Structure

> **Status:** Draft
> **Date:** 2026-03-25
> **Source:** Codebase audit of `frontend/src/routes/`, PRD.md, ACTORS.md

---

## Overview

Ondapile's frontend is a **platform management dashboard**, not an email client. There is no inbox UI, no message composer, no calendar view. The frontend serves three purposes:

1. **Auth gateway** — Sign up, log in, manage sessions
2. **Publisher dashboard** — Manage API keys, webhooks, connected accounts, settings
3. **End-user auth flows** — Hosted auth connect page, OAuth success, WhatsApp QR

The Developer (Actor 3) and Webhook Consumer (Actor 6) never touch the frontend — they interact exclusively via SDK/API.

---

## Pages

### Landing Page
- **Route:** `/`
- **Actor(s):** Anonymous (pre-auth)
- **Purpose:** Entry point — redirect to dashboard if authenticated, show login/signup links if not
- **Key components:** Hero section or simple redirect logic
- **State requirements:** Auth session check
- **Entry points:** Direct URL, bookmarks
- **Status:** ✅ Exists (`routes/index.tsx`)

---

### Login
- **Route:** `/auth/login`
- **Actor(s):** SaaS Publisher, Org Admin
- **Purpose:** Authenticate with email/password or GitHub OAuth
- **Key components:** Login form (email + password), GitHub OAuth button, "Forgot password" link, "Sign up" link
- **State requirements:** None (pre-auth)
- **Entry points:** Landing page redirect, direct URL, signup page link
- **Status:** ✅ Exists (`routes/auth/login.tsx`)

### Sign Up
- **Route:** `/auth/signup`
- **Actor(s):** SaaS Publisher (new registration)
- **Purpose:** Create account — auto-creates organization
- **Key components:** Registration form (name, email, password), GitHub OAuth button, "Already have account" link
- **State requirements:** None (pre-auth)
- **Entry points:** Landing page, login page link
- **Status:** ✅ Exists (`routes/auth/signup.tsx`)

### Forgot Password
- **Route:** `/auth/forgot-password`
- **Actor(s):** SaaS Publisher, Org Admin
- **Purpose:** Request password reset email
- **Key components:** Email input, submit button, success message
- **State requirements:** None (pre-auth)
- **Entry points:** Login page link
- **Status:** ✅ Exists (`routes/auth/forgot-password.tsx`)

### Verify Email
- **Route:** `/auth/verify-email`
- **Actor(s):** SaaS Publisher (post-signup)
- **Purpose:** Confirm email address via verification link
- **Key components:** Verification status message, resend link
- **State requirements:** Verification token from URL params
- **Entry points:** Email link (from signup verification email)
- **Status:** ✅ Exists (`routes/auth/verify-email.tsx`)

---

### Dashboard Home
- **Route:** `/dashboard`
- **Actor(s):** SaaS Publisher, Org Admin
- **Purpose:** Overview of platform status — connected accounts, recent activity, quick stats
- **Key components:** Account count card, API call stats card, webhook delivery stats card, recent activity feed
- **State requirements:** Authenticated session, current organization
- **Entry points:** Post-login redirect, sidebar navigation
- **Status:** ✅ Exists (`routes/dashboard/index.tsx`) — stats are mocked

### Dashboard Layout
- **Route:** `/dashboard` (layout wrapper)
- **Actor(s):** SaaS Publisher, Org Admin
- **Purpose:** Shared layout with sidebar navigation, org switcher, user menu
- **Key components:** Sidebar (nav links), top bar (org name, user avatar), content area
- **State requirements:** Authenticated session, organization list
- **Entry points:** All `/dashboard/*` routes render inside this layout
- **Status:** ✅ Exists (`routes/dashboard/route.tsx`)

### Create Organization
- **Route:** `/dashboard/create-org`
- **Actor(s):** SaaS Publisher (first-time or multi-org)
- **Purpose:** Create a new organization for multi-tenant API key scoping
- **Key components:** Org name input, submit button
- **State requirements:** Authenticated session
- **Entry points:** Post-signup (if no org exists), org switcher dropdown
- **Status:** ✅ Exists (`routes/dashboard/create-org.tsx`)

---

### Connected Accounts List
- **Route:** `/dashboard/accounts`
- **Actor(s):** SaaS Publisher, Org Admin
- **Purpose:** View all end-user email accounts connected via hosted auth
- **Key components:** Account table (provider icon, email, status badge, connected date), filter by provider, filter by status
- **State requirements:** `GET /api/v1/accounts` — paginated account list scoped to org
- **Entry points:** Sidebar navigation
- **Status:** ✅ Exists (`routes/dashboard/accounts/index.tsx`)

### Connected Account Detail
- **Route:** `/dashboard/accounts/$accountId`
- **Actor(s):** SaaS Publisher, Org Admin
- **Purpose:** View account details, trigger reconnect, delete account
- **Key components:** Account info card (provider, email, status, credentials), reconnect button, delete button, sync history
- **State requirements:** `GET /api/v1/accounts/:id` — single account detail
- **Entry points:** Click row in accounts list
- **Status:** ✅ Exists (`routes/dashboard/accounts/$accountId.tsx`)

---

### API Keys
- **Route:** `/dashboard/api-keys`
- **Actor(s):** Org Admin, SaaS Publisher
- **Purpose:** Create, view, and revoke API keys for SDK/API access
- **Key components:** API key table (name, key prefix, permission, created date), create key button, revoke button, copy-key-on-create modal
- **State requirements:** Better Auth API key plugin data (scoped to org)
- **Entry points:** Sidebar navigation
- **Status:** ✅ Exists (`routes/dashboard/api-keys/index.tsx`)

---

### Webhooks List
- **Route:** `/dashboard/webhooks`
- **Actor(s):** SaaS Publisher, Org Admin
- **Purpose:** Manage webhook endpoints — create, view, delete
- **Key components:** Webhook table (URL, events subscribed, status, created date), create webhook button, delete button
- **State requirements:** `GET /api/v1/webhooks` — webhook list scoped to org
- **Entry points:** Sidebar navigation
- **Status:** ✅ Exists (`routes/dashboard/webhooks/index.tsx`)

### Webhook Detail
- **Route:** `/dashboard/webhooks/$webhookId`
- **Actor(s):** SaaS Publisher, Org Admin
- **Purpose:** View webhook details, delivery history, test webhook
- **Key components:** Webhook info (URL, events, secret), recent deliveries table (status code, latency, timestamp), test button
- **State requirements:** Webhook detail + delivery history (from `webhook_deliveries` table)
- **Entry points:** Click row in webhooks list
- **Status:** ✅ Exists (`routes/dashboard/webhooks/$webhookId.tsx`)

---

### Audit Logs
- **Route:** `/dashboard/logs`
- **Actor(s):** Org Admin
- **Purpose:** View all API operations for compliance and debugging
- **Key components:** Log table (timestamp, action, actor, resource, status), filter by action type, date range filter
- **State requirements:** `GET /api/v1/audit-log` — paginated, org-scoped
- **Entry points:** Sidebar navigation
- **Status:** ✅ Exists (`routes/dashboard/logs/index.tsx`)

---

### Settings — General
- **Route:** `/dashboard/settings`
- **Actor(s):** SaaS Publisher, Org Admin
- **Purpose:** Organization-level settings (name, defaults)
- **Key components:** Org name edit, danger zone (delete org)
- **State requirements:** Organization detail
- **Entry points:** Sidebar navigation → Settings section
- **Status:** ✅ Exists (`routes/dashboard/settings/index.tsx`)

### Settings — Team
- **Route:** `/dashboard/settings/team`
- **Actor(s):** Org Admin
- **Purpose:** Invite team members, assign roles, remove members
- **Key components:** Member table (name, email, role badge), invite form (email + role select), role change dropdown, remove button
- **State requirements:** Better Auth organization members
- **Entry points:** Settings sub-navigation
- **Status:** ✅ Exists (`routes/dashboard/settings/team.tsx`)

### Settings — Billing
- **Route:** `/dashboard/settings/billing`
- **Actor(s):** Org Admin
- **Purpose:** View subscription plan, usage, invoices
- **Key components:** Current plan card, usage meters (accounts, API calls), invoice table, upgrade button
- **State requirements:** Billing data (currently mocked — hardcoded trial status)
- **Entry points:** Settings sub-navigation
- **Status:** 🔴 Mocked (`routes/dashboard/settings/billing.tsx` — hardcoded data, no Stripe)

### Settings — OAuth Credentials
- **Route:** `/dashboard/settings/oauth`
- **Actor(s):** Platform Operator
- **Purpose:** Configure Google/Microsoft OAuth app credentials for provider connections
- **Key components:** Google OAuth form (client ID, client secret, redirect URI), Microsoft OAuth form (same), test connection button
- **State requirements:** OAuth credential store (`internal/oauth/store.go`)
- **Entry points:** Settings sub-navigation
- **Status:** ✅ Exists (`routes/dashboard/settings/oauth.tsx`)

### Settings — Hosted Auth
- **Route:** `/dashboard/settings/hosted-auth`
- **Actor(s):** SaaS Publisher
- **Purpose:** Configure and preview the hosted auth experience that end users see
- **Key components:** Hosted auth link generator, preview iframe, customization options (logo, colors — future)
- **State requirements:** `POST /api/v1/accounts/hosted-auth` — generates auth URL
- **Entry points:** Settings sub-navigation
- **Status:** ✅ Exists (`routes/dashboard/settings/hosted-auth.tsx`)

---

### Hosted Auth Connect (End User)
- **Route:** `/connect/$token`
- **Actor(s):** End User
- **Purpose:** The hosted auth page end users see when connecting their email account. Renders provider selection and auth flow.
- **Key components:** Provider selection (Gmail, Outlook, IMAP), provider-specific auth forms, loading/success/error states
- **State requirements:** Valid hosted auth token (from URL param `$token`), decoded to determine org and allowed providers
- **Entry points:** Hosted auth link embedded in publisher's app (generated via API)
- **Status:** ✅ Exists (`routes/connect/$token.tsx`)

---

### OAuth Success (Server-Rendered)
- **Route:** `/oauth/success`
- **Actor(s):** End User
- **Purpose:** Post-OAuth redirect — shows "Account Connected, you can close this window"
- **Key components:** Success checkmark, close-window message
- **State requirements:** None — static HTML served by Go backend
- **Entry points:** OAuth provider redirect (Google/Microsoft callback → Go handler → redirect here)
- **Status:** ✅ Exists (inline HTML in `router.go`)

### WhatsApp QR Page (Server-Rendered)
- **Route:** `/wa/qr/:id`
- **Actor(s):** End User
- **Purpose:** Display WhatsApp QR code for account linking
- **Key components:** QR code display, refresh button, status polling, success/error messages
- **State requirements:** Account ID from URL, QR code data from `GET /api/v1/accounts/:id/qr`
- **Entry points:** Hosted auth flow (when end user selects WhatsApp)
- **Status:** ✅ Exists (`accountH.QRPage` in Go)

---

## Modals

### Create API Key
- **Trigger:** "Create Key" button on `/dashboard/api-keys`
- **Actor(s):** Org Admin, SaaS Publisher
- **Purpose:** Generate a new API key with name and permission scope
- **Inputs:** Key name (text), permission scope (select: full, read, email, calendar)
- **Outcomes:**
  - Confirm → Creates key via Better Auth, shows full key value ONCE for copying
  - Cancel → Returns to API keys list
- **Parent page(s):** `/dashboard/api-keys`

### Create Webhook
- **Trigger:** "Create Webhook" button on `/dashboard/webhooks`
- **Actor(s):** SaaS Publisher, Org Admin
- **Purpose:** Register a new webhook endpoint
- **Inputs:** Callback URL (text), event types (multi-select from 24 event types)
- **Outcomes:**
  - Confirm → Creates webhook via `POST /api/v1/webhooks`, returns webhook secret
  - Cancel → Returns to webhooks list
- **Parent page(s):** `/dashboard/webhooks`

### Delete Confirmation
- **Trigger:** Delete button on any resource (account, webhook, API key)
- **Actor(s):** SaaS Publisher, Org Admin
- **Purpose:** Confirm destructive action before deletion
- **Inputs:** None (confirmation only — shows resource name/identifier)
- **Outcomes:**
  - Confirm → Executes DELETE request, removes from list, shows toast
  - Cancel → Closes modal, no action
- **Parent page(s):** `/dashboard/accounts/$accountId`, `/dashboard/webhooks`, `/dashboard/api-keys`

### Invite Team Member
- **Trigger:** "Invite" button on `/dashboard/settings/team`
- **Actor(s):** Org Admin
- **Purpose:** Invite a new team member to the organization
- **Inputs:** Email address (text), role (select: admin, member)
- **Outcomes:**
  - Confirm → Sends invitation via Better Auth org plugin
  - Cancel → Returns to team page
- **Parent page(s):** `/dashboard/settings/team`

### Reconnect Account
- **Trigger:** "Reconnect" button on `/dashboard/accounts/$accountId` (when status is disconnected/error)
- **Actor(s):** SaaS Publisher, Org Admin
- **Purpose:** Re-authenticate a disconnected provider account
- **Inputs:** None (triggers `POST /api/v1/accounts/:id/reconnect`)
- **Outcomes:**
  - Confirm → Initiates reconnect flow, may open OAuth window or show credentials form
  - Cancel → Returns to account detail
- **Parent page(s):** `/dashboard/accounts/$accountId`

---

## User Flows (Implementation)

### Flow: Sign Up & First Integration — Actor: SaaS Publisher

1. `GET /` → Landing page, user is anonymous → redirected to login
2. Action: Clicks "Sign up" link → navigates to signup
3. `GET /auth/signup` → User fills name, email, password → submits form
4. Action: Better Auth creates user → auto-creates organization → redirects
5. `GET /auth/verify-email` → User checks email, clicks verification link
6. `GET /auth/login` → User logs in with verified credentials
7. `GET /dashboard` → Dashboard home — empty state (no accounts, no keys)
8. Action: Clicks "API Keys" in sidebar
9. `GET /dashboard/api-keys` → Empty state → clicks "Create Key"
10. Modal: **Create API Key** → enters name "Production", selects "full" → confirms
11. `GET /dashboard/api-keys` → Copies key value (shown once) → key appears in table
12. Action: Clicks "Webhooks" in sidebar
13. `GET /dashboard/webhooks` → Empty state → clicks "Create Webhook"
14. Modal: **Create Webhook** → enters URL, selects events → confirms
15. `GET /dashboard/webhooks` → Webhook appears in table with secret

### Flow: Connect End-User Account — Actor: End User (via SaaS Publisher's App)

1. Publisher's app generates hosted auth link via `POST /api/v1/accounts/hosted-auth`
2. End user clicks link in publisher's app → opens in browser
3. `GET /connect/$token` → Hosted auth page — sees provider options (Gmail, Outlook, IMAP)
4. Action: Selects "Gmail" → redirected to Google OAuth consent screen
5. External: Google OAuth → user grants access → Google redirects to callback
6. `GET /api/v1/oauth/callback/google` → Go handler processes OAuth, creates account
7. `GET /oauth/success` → User sees "✅ Account Connected — you can close this window"
8. Webhook `account_connected` fires → publisher's server receives notification

### Flow: Connect WhatsApp — Actor: End User

1. End user clicks hosted auth link → selects WhatsApp on connect page
2. `GET /connect/$token` → Initiates WhatsApp connection → creates pending account
3. `GET /wa/qr/:id` → QR code displayed → user scans with WhatsApp mobile app
4. Action: QR scanned → connection established (polled via `GET /api/v1/accounts/:id`)
5. Page updates to show "Connected" → user closes window

### Flow: Monitor Platform — Actor: Org Admin

1. `GET /dashboard` → Reviews stats cards (accounts connected, API calls, webhook deliveries)
2. Action: Clicks "Accounts" in sidebar
3. `GET /dashboard/accounts` → Reviews account statuses — spots one with "Error" badge
4. Action: Clicks errored account row
5. `GET /dashboard/accounts/$accountId` → Sees error details → clicks "Reconnect"
6. Modal: **Reconnect Account** → confirms → reconnect flow initiated
7. Action: Clicks "Logs" in sidebar
8. `GET /dashboard/logs` → Reviews recent API operations, filters by error status

### Flow: Manage Team — Actor: Org Admin

1. `GET /dashboard/settings/team` → Views current team members and roles
2. Action: Clicks "Invite" button
3. Modal: **Invite Team Member** → enters email, selects "member" role → confirms
4. Action: Invitation sent — invitee appears in table with "Pending" status
5. Action: Changes existing member's role → selects "admin" from dropdown
6. Action: Removes a team member → Delete Confirmation modal → confirms

### Flow: Configure OAuth Credentials — Actor: Platform Operator

1. `GET /dashboard/settings/oauth` → Sees empty OAuth credential forms
2. Action: Fills in Google OAuth credentials (client ID, client secret, redirect URI)
3. Action: Fills in Microsoft OAuth credentials
4. Action: Saves → credentials encrypted with AES-256-GCM and stored
5. Result: Gmail and Outlook providers now available in hosted auth connect page

---

## Tensions & Observations

### What This Phase Implies for Backend (Phase 2)

1. **Dashboard stats are mocked** — The dashboard home (`/dashboard`) shows API call counts, webhook delivery stats, and account counts. Backend needs aggregation endpoints or the frontend must compute from raw data. This is currently hardcoded.

2. **Billing is a UI shell** — `/dashboard/settings/billing` exists but renders hardcoded data. No billing backend exists. v1 PRD explicitly excludes Stripe. This page should either be removed or show a clear "Free tier — billing coming soon" message.

3. **No admin panel** — The Platform Operator (Actor 1) uses the same dashboard as SaaS Publishers. There's no `/admin` route for cross-org management, instance configuration, or superadmin operations. The PRD lists this as post-v1.

4. **Webhook detail needs delivery history** — `/dashboard/webhooks/$webhookId` needs to display delivery attempts from `webhook_deliveries` table. This requires a backend endpoint that doesn't exist yet (delivery history query).

### Actor View Splits

The dashboard layout serves both SaaS Publisher and Org Admin with the same sidebar. The only role-specific page is team management (admin-only invite/role-change). Currently there are **no route-level permission guards** — any authenticated org member can access all dashboard pages. The PRD flags this as a gap (§5.7, line 239: "No route-level permission checks").

### Flows That May Cause UX Friction

- **Hosted auth connect** (`/connect/$token`) — This is the most critical UX flow because it's the one page end users actually see. Any friction here (slow load, unclear provider options, failed OAuth) directly impacts publisher adoption. This page should be the most polished.

- **API key creation** — The key value is only shown once. If the user misses it, they must revoke and create a new one. Consider adding a "copy to clipboard" interaction with visual confirmation.

---

## Pages NOT in v1 (Confirmed Exclusions)

Per PRD §1.4 ("What v1 is NOT"):

| Excluded Page | Reason |
|---------------|--------|
| `/inbox` or `/messages` | No unified inbox UI — API only |
| `/calendar` | No calendar UI — API only |
| `/admin` | No admin panel — Platform Operator uses env vars |
| `/docs` | No developer docs site — future |
| `/billing/checkout` | No Stripe — billing mocked |
| `/sequences` | No outreach sequences |

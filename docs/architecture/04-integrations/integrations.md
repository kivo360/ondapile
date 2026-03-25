# Phase 4: Integrations

> **Status:** Draft
> **Date:** 2026-03-25

---

## Overview

Ondapile integrates with 8 external providers and depends on PostgreSQL as its only infrastructure requirement. Unlike Nylas/Unipile (which are cloud services), ondapile is self-hosted, so all OAuth credentials and API keys belong to the operator.

---

## External Service Map

### Integration: Google APIs (Gmail + Calendar)
- **Purpose:** Email read/send/manage via Gmail API, calendar CRUD via Calendar API
- **Direction:** Bidirectional (read inbox + send email + manage labels)
- **Auth method:** OAuth 2.0 (operator registers Google Cloud app, end users consent)
- **Data exchanged:** Email messages (MIME), labels, calendar events, attachments
- **Failure impact:** Gmail/GCal accounts show INTERRUPTED, email operations fail for those accounts
- **Cost model:** Free (Gmail API quota: 250 units/user/second)
- **Alternatives:** IMAP/SMTP for basic email (already supported)
- **Status:** ✅ OAuth works, ⚠️ adapter methods partially stubbed

### Integration: Microsoft Graph (Outlook + Calendar)
- **Purpose:** Email read/send/manage via Graph API, calendar CRUD
- **Direction:** Bidirectional
- **Auth method:** OAuth 2.0 (operator registers Azure AD app, end users consent)
- **Data exchanged:** Email messages (Graph JSON), folders, calendar events
- **Failure impact:** Outlook accounts show INTERRUPTED
- **Cost model:** Free (Graph API: 10K requests/10 min per app)
- **Alternatives:** IMAP/SMTP for basic email
- **Status:** ✅ OAuth works, ⚠️ adapter methods partially stubbed

### Integration: IMAP/SMTP Servers
- **Purpose:** Generic email for any IMAP-compatible server
- **Direction:** Bidirectional (IMAP read + SMTP send)
- **Auth method:** Direct credentials (username/password, app-specific passwords)
- **Data exchanged:** Email messages (RFC 5322 MIME), folder listings
- **Failure impact:** Account shows INTERRUPTED, polling stops
- **Cost model:** Free (operator's mail server)
- **Alternatives:** Gmail/Outlook OAuth for Google/Microsoft accounts
- **Status:** ✅ Most complete adapter

### Integration: WhatsApp (via whatsmeow)
- **Purpose:** WhatsApp messaging — send/receive messages, media, QR auth
- **Direction:** Bidirectional
- **Auth method:** QR code scanning (WhatsApp Web protocol via whatsmeow library)
- **Data exchanged:** Messages (text, media, voice notes), contact info
- **Failure impact:** WhatsApp account disconnects, messages stop flowing
- **Cost model:** Free (no API fees — uses WhatsApp Web protocol, NOT Business API)
- **Alternatives:** WhatsApp Business API (Cloud/On-Premises), but requires Meta approval
- **Status:** ✅ Fully implemented (not v1 scope but working)
- **⚠️ Risk:** WhatsApp Web protocol is unofficial. WhatsApp may block/throttle.

### Integration: LinkedIn API
- **Purpose:** Messaging, profile data, connections, InMail
- **Direction:** Bidirectional
- **Auth method:** OAuth 2.0 (official) or session cookies (unofficial)
- **Data exchanged:** Messages, profiles, connection requests
- **Failure impact:** LinkedIn account disconnects
- **Cost model:** Free API access (with rate limits)
- **Status:** ⚠️ OAuth works, messaging code exists but not wired to adapter

### Integration: Instagram Graph API
- **Purpose:** Direct messaging
- **Direction:** Bidirectional
- **Auth method:** Facebook/Instagram OAuth 2.0
- **Data exchanged:** DMs, media
- **Status:** ⚠️ API calls work, normalize functions stubbed

### Integration: Telegram Bot API
- **Purpose:** Bot messaging
- **Direction:** Bidirectional
- **Auth method:** Bot token (from @BotFather)
- **Data exchanged:** Messages, updates
- **Status:** ⚠️ HTTP client exists, messaging not wired

### Integration: GitHub OAuth (for sign-in)
- **Purpose:** Social sign-in for the dashboard
- **Direction:** Inbound only (auth)
- **Auth method:** OAuth 2.0 (via Better Auth social provider)
- **Data exchanged:** User profile (name, email, avatar)
- **Failure impact:** GitHub sign-in button doesn't work (email/password still works)
- **Cost model:** Free
- **Status:** ✅ Working

---

## Infrastructure Requirements

### Infra: PostgreSQL Database
- **Capability:** Primary data store, webhook queue, audit log
- **Requirements:** PostgreSQL 14+ with pgvector extension, 1GB+ disk
- **Cost:** Free (self-hosted) or ~$15/mo (managed — Supabase, Neon, RDS)
- **Status:** ✅ Required, working

### Infra: Application Server
- **Capability:** Run Go binary + serve HTTP
- **Requirements:** 512MB RAM minimum, single core sufficient for low traffic
- **Cost:** ~$5-20/mo (VPS — Hetzner, DigitalOcean, Fly.io)
- **Status:** ✅ Single binary, Docker support

### Infra: Redis (listed but unused)
- **Capability:** Caching, rate limiting, session store
- **Requirements:** Listed in PRD tech stack but NOT used in codebase
- **Cost:** Would be ~$10/mo managed
- **Status:** ❌ Not used — all functions handled by PostgreSQL
- **Decision:** Remove from tech stack or add for rate limiting if needed at scale

### Infra: Email Delivery (for auth emails)
- **Capability:** Send verification, password reset, invitation emails
- **Requirements:** SMTP relay or email API
- **Current:** Mailpit (dev only — local SMTP trap)
- **Production:** Needs Resend, SendGrid, or similar
- **Status:** 🔴 Dev only — no production email delivery configured

---

## Tensions

1. **WhatsApp Web protocol risk** — whatsmeow uses the unofficial WhatsApp Web protocol. This works today but Meta could break it at any time. The alternative (WhatsApp Business API) requires Meta business verification and costs per message.

2. **LinkedIn session auth risk** — LinkedIn doesn't officially support messaging via API for most apps. Session-based auth (cookies) works but violates LinkedIn ToS and sessions expire frequently.

3. **No production email delivery** — Auth emails (verification, password reset, invitations) currently go to Mailpit (dev). Need to add Resend or similar for production. This is a Phase 4 item in PROJECT_SPEC.md.

4. **Redis listed but unused** — PRD lists Redis in the tech stack (port 6380) but the codebase doesn't use it. Either remove from docs or add for rate limiting/caching if needed.

5. **8 external provider APIs** — Each with different auth methods, rate limits, error patterns, and data formats. The adapter pattern handles this but debugging provider-specific issues requires deep knowledge of each API.

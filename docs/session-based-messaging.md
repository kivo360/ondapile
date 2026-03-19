# Session-Based Messaging + Provider API Access — Future Work

**Date:** March 19, 2026
**Status:** Not started — Phase 4 candidate
**Priority:** High (this is the core Unipile differentiator)

---

## The Problem

Several providers don't offer official API access for personal DMs:

| Provider | Official DM API? | How Unipile Does It |
|----------|-----------------|---------------------|
| LinkedIn | **No** — no OAuth endpoint for personal messaging | Session cookies from browser login |
| Instagram | **Partial** — only Business/Creator accounts via Graph API | Session cookies for personal accounts |
| X (Twitter) | **Yes** — pay-as-you-go API (Basic tier $200/mo, Pro $5k/mo) | Session cookies (to avoid cost) |

The OAuth adapters we built handle auth, profiles, posting, and connections — but personal messaging requires either session-based access or paid API tiers.

**Note:** X/Twitter re-introduced DM API access via paid tiers in 2024-2025. The Basic tier ($200/mo) includes DM read/write. This makes X the only provider where we have a choice: official paid API (clean, reliable) vs session-based (free, fragile). For self-hosted users who want to avoid the cost, session-based remains an option.

## The Pattern

Session-based adapters work like WhatsApp/wuzapi — maintain an authenticated browser session and communicate through the platform's internal APIs:

```
User provides credentials
  → Adapter logs in via platform's web auth
    → Captures session cookies / tokens
      → Makes requests using internal (non-public) API endpoints
        → Normalizes responses into ondapile models
```

This is the same fundamental approach as:
- **wuzapi/whatsmeow** — maintains Signal protocol session with WhatsApp servers
- **Unipile** — maintains browser sessions for LinkedIn, Instagram, X
- **Puppeteer-based scrapers** — automate browser login, extract cookies

## What Would Need to Be Built

### Per-Provider Session Adapter

Each provider needs its own session management because the internal APIs differ:

**LinkedIn Session Adapter:**
- Login via `https://www.linkedin.com/uas/login`
- Capture `li_at` and `JSESSIONID` cookies
- Use internal Voyager API (`https://www.linkedin.com/voyager/api/`) for messaging
- Handle 2FA/CAPTCHA challenges
- Session refresh on expiry

**Instagram Session Adapter:**
- Login via Instagram web or mobile API
- Capture session cookies
- Use internal API endpoints for DMs
- Handle 2FA challenges

**X/Twitter Session Adapter:**
- Login via Twitter web
- Capture auth tokens
- Use internal GraphQL API for DMs

### Shared Infrastructure

- **Proxy rotation** — per-account proxy to avoid IP-based detection (config already supports this via `proxy_config` on accounts)
- **Session persistence** — store encrypted cookies in PostgreSQL (similar to OAuth token store)
- **Rate limiting** — respect platform-specific limits to avoid detection
- **Reconnection** — detect expired sessions, prompt re-auth
- **Fingerprinting** — rotate User-Agent, TLS fingerprint to look like real browsers

### Risk Considerations

| Risk | Severity | Mitigation |
|------|----------|-----------|
| ToS violation | Medium | Users self-host and assume responsibility (same as Unipile's model) |
| Session invalidation | High | Detect and prompt re-auth via `account.status_changed` webhook |
| Rate limiting / bans | High | Per-account proxies, human-like request patterns |
| API changes | Medium | Internal APIs change without notice — need monitoring |
| 2FA challenges | Medium | Support TOTP, SMS, email verification in `SolveCheckpoint` |

## Architecture Decision

Two options for session-based adapters:

### Option A: Go + net/http (like current adapters)
- Directly make HTTP requests with captured cookies
- Lighter weight, no browser dependency
- Harder to handle JavaScript-heavy login flows
- **Best for:** LinkedIn (Voyager API is well-documented by reverse engineers)

### Option B: Headless browser (Playwright/Rod)
- Use a headless browser for login + session capture
- Handles JavaScript, CAPTCHAs, complex auth flows
- Heavier resource usage
- Rod (Go) or Playwright (via CLI)
- **Best for:** Instagram, X (complex login flows with bot detection)

### Recommendation: Hybrid
- LinkedIn: Option A (Go + net/http with Voyager API)
- Instagram: Option B (headless browser for login, then switch to direct HTTP)
- X/Twitter: Option A (official paid API — oauth2 + net/http, same pattern as Gmail/Outlook)

## References

| Resource | URL | Notes |
|----------|-----|-------|
| LinkedIn Voyager API (community docs) | Search GitHub for "linkedin-api" | Multiple Python implementations exist |
| linkedin-api (Python) | github.com/tomquirk/linkedin-api | Most popular reverse-engineered LinkedIn client |
| instagrapi (Python) | github.com/subzeroid/instagrapi | Instagram private API client |
| Unipile architecture | unipile.com | Commercial reference — observe their auth wizard flow |
| Rod (Go headless browser) | github.com/go-rod/rod | Chromium automation for Go |
| X/Twitter API pricing | developer.x.com/en/portal/products | Basic $200/mo, Pro $5k/mo — includes DM read/write |
| X API v2 DM docs | developer.x.com/en/docs/twitter-api/direct-messages | Official DM endpoints |

## When to Tackle This

**Prerequisites:**
- OAuth adapters stable and tested (current Phase 2 — done)
- Proxy config working per-account
- Webhook dispatch proven reliable
- Docker deployment tested

**Estimated effort:** 2-3 weeks per provider for production-quality session adapter
**Suggested order:** X/Twitter first (official API, same oauth2+net/http pattern — fastest to build), then LinkedIn (best documented internal API), then Instagram

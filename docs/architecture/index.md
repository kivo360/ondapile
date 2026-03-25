# Architecture Index

> Generated: 2026-03-25
> Source: [PRD.md](../PRD.md) | [ACTORS.md](../actors/ACTORS.md) | [user-flows.md](../user-flows.md)

---

## Status

| Phase | Status | Last Updated |
|-------|--------|--------------|
| Frontend | Draft | 2026-03-25 |
| Backend | Draft | 2026-03-25 |
| Data | Draft | 2026-03-25 |
| Integrations | Draft | 2026-03-25 |

---

## Entity Cross-Reference

| Entity/Concept | Defined In | Referenced In |
|----------------|-----------|---------------|
| Actor: Platform Operator | ACTORS.md | frontend.md (settings/oauth, settings/hosted-auth) |
| Actor: SaaS Publisher | ACTORS.md | frontend.md (signup, dashboard, api-keys, webhooks) |
| Actor: Developer | ACTORS.md | user-flows.md (SDK usage — no frontend pages) |
| Actor: End User | ACTORS.md | frontend.md (connect/$token, /oauth/success, /wa/qr/:id) |
| Actor: Org Admin | ACTORS.md | frontend.md (settings/team, settings/billing, logs) |
| Actor: Webhook Consumer | ACTORS.md | user-flows.md (system actor — no frontend pages) |
| Email Entity | PRD.md §8.1 | backend.md (API surface, adapter system), data.md (pending) |
| Account Entity | PRD.md §8.2 | frontend.md (accounts list/detail), backend.md (adapter layer, reconnect) |
| Webhook Entity | PRD.md §8.2 | frontend.md (webhooks list/detail), backend.md (dispatcher subsystem) |
| API Key Entity | PRD.md §8.2 | frontend.md (api-keys), backend.md (DualAuthMiddleware) |
| Provider Interface | adapter.go | backend.md (adapter layer, 8 provider packages) |
| Webhook Dispatcher | dispatcher.go | backend.md (webhook subsystem), data.md (pending) |
| OAuth Token Store | oauth/store.go | backend.md (OAuth subsystem), data.md (pending) |

---

## Dependency Graph

| If This Changes... | Check These Files... |
|--------------------|---------------------|
| Actor added/removed | frontend.md, backend.md, data.md |
| New API endpoint | backend.md, frontend.md (if dashboard needs UI) |
| New provider added | backend.md, integrations.md, data.md |
| Auth model changed | frontend.md, backend.md, integrations.md |
| New webhook event | backend.md, data.md |
| Constraint tightened | All phase files |

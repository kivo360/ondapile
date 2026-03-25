# Ondapile Meta Plan

> The master reference for the entire project — from current state to migration completion.
> Every document, every eval, every phase, linked and ordered.
>
> Created: 2026-03-25

---

## Document Map

```
docs/
├── META_PLAN.md                      ← YOU ARE HERE
│
├── ─── AGENT PROTOCOL ─────────────
├── DEV_METHODOLOGY.md                Development methodology — build order, habits, playbook (458 lines)
├── AGENT_RULES.md                    5 error-prevention rules + execution protocol (195 lines)
│
├── ─── PRODUCT ────────────────────
├── PRD.md                            Product requirements (467 lines)
├── PROJECT_SPEC.md                   Legacy spec (partially superseded by PRD)
├── actors/ACTORS.md                  6 actors with interaction maps (362 lines)
├── user-flows.md                     Full SDK flows per actor (995 lines)
│
├── ─── COMPETITIVE ────────────────
├── nylas-vs-unipile-comparison.md    Platform comparison with SDK examples (811 lines)
├── python-usage-guide.md             Python pseudocode usage guide (516 lines)
│
├── ─── ARCHITECTURE ───────────────
├── architecture/
│   ├── index.md                      Cross-phase index (48 lines)
│   ├── 01-frontend/frontend.md       22 pages, 5 modals, 6 flows (400 lines)
│   ├── 01-frontend/navigation.mermaid  Navigation diagram (108 lines)
│   ├── 02-backend/backend.md         8 capabilities, 9 modules (374 lines)
│   ├── 03-data/data.md               11 entities, ER diagram (347 lines)
│   └── 04-integrations/integrations.md  8 providers, infra (132 lines)
│
├── ─── EVALS ─────────────────────
├── COMPLETE_EVALS.md                 Master eval suite — 131 deterministic evals, 4 tiers (370 lines)
├── SCENARIO_EVALS.md                 8 agent-executable scenarios — 64 steps, 45 assertions, 33 fuzzy (716 lines)
├── EVALS.md                          Go backend evals (527 lines, 245+ tests)
├── FRONTEND_EVALS.md                 Playwright browser evals (758 lines, 29 tests)
│
├── ─── MIGRATION ──────────────────
├── MIGRATION_PLAN.md                 Go → Hono/Drizzle plan, 7 phases (430 lines)
└── MIGRATION_EVALS.md                57 evals for migration (898 lines)
```

**Total documentation: ~10,000 lines across 20 documents.**

---

## Current State (Pre-Migration)

| Component | Status | Tech |
|-----------|--------|------|
| Frontend (dashboard) | ✅ Working | TanStack Start, React 19, shadcn/ui |
| Auth (Better Auth) | ✅ Working | org, admin, apiKey plugins |
| Go Backend (API) | ✅ Working, 49 endpoints | Go, Gin, pgx raw SQL |
| IMAP Adapter | ✅ Working | Go, custom IMAP client |
| Gmail Adapter | ⚠️ Partial | Go, Gmail API |
| Outlook Adapter | ⚠️ Partial | Go, Microsoft Graph |
| WhatsApp Adapter | ✅ Working (not v1) | Go, whatsmeow |
| Email Tracking | ✅ Working | Go, pixel + link handlers |
| Webhook Dispatcher | ✅ Working | Go, PostgreSQL queue |
| Node SDK | ✅ Working, 27 tests | TypeScript |
| Integration Tests | ✅ 200+ tests pass | Go test + real PostgreSQL |

---

## Target State (Post-Migration)

| Component | Tech | Change |
|-----------|------|--------|
| Frontend (dashboard) | TanStack Start, React 19 | **Unchanged** |
| Auth (Better Auth) | Native Drizzle adapter | **Simplified** (no dual auth) |
| API Server | Hono (same process as frontend) | **Rewritten** from Go |
| ORM | Drizzle | **New** (replaces pgx raw SQL) |
| IMAP Adapter | imapflow | **Rewritten** from Go |
| SMTP | nodemailer | **Rewritten** from Go |
| Provider HTTP | fetch API | **Rewritten** from Go net/http |
| Email Tracking | Hono handlers + Drizzle | **Rewritten** |
| Webhook Dispatcher | Custom async + PostgreSQL | **Rewritten** |
| Tests | vitest + Playwright | **Unified** (was Go test + vitest + Playwright) |

---

## Migration Execution Order

```
Phase 0: Setup (Foundation)
├── 10 evals (P0.1 – P0.10)
├── Deliverable: Hono boots, Better Auth works, schema matches DB
└── Dependency: None

Phase 1: Core API (Accounts + Auth)
├── 12 evals (P1.1 – P1.12)
├── Deliverable: Account CRUD, auth middleware, health, metrics
└── Dependency: Phase 0

Phase 2: Email Operations
├── 10 evals (P2.1 – P2.10)
├── Deliverable: All 14 email operations, IMAP/SMTP adapters
└── Dependency: Phase 1

Phase 3: Webhooks + Tracking
├── 8 evals (P3.1 – P3.8)
├── Deliverable: Webhook CRUD + dispatch, tracking pixel/link, audit log
└── Dependency: Phase 1

Phase 4: Messaging + Calendar Routes
├── 5 evals (P4.1 – P4.5)
├── Deliverable: Chat, message, attendee, calendar, search endpoints
└── Dependency: Phase 1

Phase 5: Frontend Integration
├── 5 evals (P5.1 – P5.5)
├── Deliverable: Frontend talks to single TS server, Go backend removed
└── Dependency: Phases 2, 3, 4

Phase 6: Cleanup
├── 4 evals (P6.1 – P6.4)
├── Deliverable: Go code deleted, build clean, TypeScript clean
└── Dependency: Phase 5

Fuzzy Evals (run after each phase)
├── 4 evals (F1 – F4)
├── Deliverable: API compatibility verified by model grader
└── Dependency: Run after each phase
```

### Parallelization

```
Phase 0 ════════╗
                 ╠═► Phase 1 ════════╗
                                      ╠═► Phase 2 ════╗
                                      ╠═► Phase 3 ════╣  (parallel)
                                      ╠═► Phase 4 ════╣
                                                       ╠═► Phase 5 ═► Phase 6
```

Phases 2, 3, and 4 can run in parallel after Phase 1 completes.

---

## AI Agent Execution Protocol

For each phase, an AI agent should:

1. **Read** `MIGRATION_PLAN.md` for the phase's task list
2. **Read** `MIGRATION_EVALS.md` for the phase's eval definitions
3. **Implement** each task in the phase
4. **Run** `bun test tests/evals/phaseN.test.ts` after each task
5. **Fix** any failing evals before moving to next task
6. **Run** all phase evals when all tasks are done
7. **Run** all previous phase evals (regression check)
8. **Commit** with message: `Phase N: [description]`
9. **Report** eval results

### Agent Prompt Template

```
You are implementing Phase [N] of the ondapile migration (Go → Hono/Drizzle).

## Context
- Read docs/MIGRATION_PLAN.md for the task list
- Read docs/MIGRATION_EVALS.md for the eval definitions
- The Drizzle schema is defined in server/src/db/schema.ts
- The Hono app is in server/src/app.ts
- Better Auth config is in server/src/lib/auth.ts

## Tasks for Phase [N]
[List from MIGRATION_PLAN.md]

## Evals that must pass
[List from MIGRATION_EVALS.md]

## Rules — MANDATORY (read docs/AGENT_RULES.md)
1. **Verify before you write** — test assumptions with 3-line proofs, not 50-line implementations
2. **One thing per agent** — one function + its test = max scope per task
3. **Verify state after every mutation** — grep after edit, diff before commit
4. **Check existing code before creating new** — search before you create helpers/types
5. **Run tests after every change** — not at the end, after EVERY change
6. Run evals frequently: `bun test tests/evals/phaseN.test.ts`
7. Fix failing evals before proceeding to next task
8. When all phase evals pass, run ALL previous phase evals
9. Commit only when all evals pass
10. Do not modify eval files — they are the contract
```

---

## Eval Summary

| Phase | Deterministic | Fuzzy | Total | Pass Criteria |
|-------|--------------|-------|-------|---------------|
| 0: Setup | 10 | — | 10 | pass^3 = 100% |
| 1: Core API | 12 | — | 12 | pass^3 = 100% |
| 2: Email Ops | 10 | — | 10 | pass^3 = 100% |
| 3: Webhooks | 8 | — | 8 | pass^3 = 100% |
| 4: Routes | 5 | — | 5 | pass^3 = 100% |
| 5: Frontend | 5 | — | 5 | pass^3 = 100% |
| 6: Cleanup | 4 | — | 4 | pass^3 = 100% |
| Cross-phase | — | 4 | 4 | avg >= 4/5 |
| **Total** | **54** | **4** | **58** | |

Plus existing evals that must still pass post-migration:
- Node SDK tests: 27
- Playwright browser evals: 29

**Grand total: 114 evals**

---

## Key Libraries Reference

| Purpose | Library | Install |
|---------|---------|---------|
| API framework | hono | `bun add hono @hono/node-server` |
| ORM | drizzle-orm | `bun add drizzle-orm pg` |
| ORM tooling | drizzle-kit | `bun add -D drizzle-kit` |
| Auth | better-auth | `bun add better-auth` |
| IMAP | imapflow | `bun add imapflow` |
| SMTP | nodemailer | `bun add nodemailer` |
| Validation | zod | `bun add zod @hono/zod-validator` |
| Encryption | Node crypto | built-in |
| Tests | vitest | `bun add -D vitest` |
| E2E tests | @playwright/test | `bun add -D @playwright/test` |
| QR codes | qrcode | `bun add qrcode` |
| Webhook signing | Node crypto (HMAC) | built-in |

---

## File Structure (Target)

```
ondapile/
├── frontend/                    # TanStack Start (unchanged)
│   ├── src/
│   │   ├── routes/              # Pages
│   │   ├── components/          # UI components
│   │   └── lib/
│   │       ├── auth.ts          # Better Auth server (UPDATED: Drizzle adapter)
│   │       ├── auth-client.ts   # Better Auth client (unchanged)
│   │       └── api-client.ts    # UPDATED: same-origin, no port switch
│   └── tests/
│       └── e2e/                 # Playwright browser evals
│
├── server/                      # NEW: Hono API server
│   ├── src/
│   │   ├── app.ts              # Hono app + route mounting
│   │   ├── index.ts            # Entry point (serve)
│   │   ├── db/
│   │   │   ├── schema.ts       # Drizzle schema (ALL tables)
│   │   │   └── client.ts       # Drizzle client (pg Pool)
│   │   ├── lib/
│   │   │   ├── auth.ts         # Better Auth config (Drizzle adapter)
│   │   │   ├── crypto.ts       # AES-256-GCM encryption
│   │   │   └── webhook.ts      # Webhook dispatcher
│   │   ├── routes/
│   │   │   ├── accounts.ts     # Account CRUD
│   │   │   ├── emails.ts       # Email operations
│   │   │   ├── webhooks.ts     # Webhook CRUD
│   │   │   ├── chats.ts        # Chat endpoints
│   │   │   ├── messages.ts     # Message endpoints
│   │   │   ├── attendees.ts    # Attendee endpoints
│   │   │   ├── calendars.ts    # Calendar endpoints
│   │   │   ├── tracking.ts     # Pixel + link tracking
│   │   │   └── search.ts       # Semantic search
│   │   ├── adapters/
│   │   │   ├── types.ts        # Provider interface (TypeScript)
│   │   │   ├── registry.ts     # Provider registry
│   │   │   ├── imap/           # IMAP adapter (imapflow)
│   │   │   ├── gmail/          # Gmail adapter (fetch)
│   │   │   ├── outlook/        # Outlook adapter (fetch)
│   │   │   ├── whatsapp/       # WhatsApp adapter
│   │   │   ├── linkedin/       # LinkedIn adapter
│   │   │   ├── telegram/       # Telegram adapter
│   │   │   └── gcal/           # Google Calendar adapter
│   │   └── middleware/
│   │       ├── auth.ts         # API key validation (Better Auth native)
│   │       ├── cors.ts         # CORS
│   │       └── rate-limit.ts   # Rate limiting
│   └── drizzle/                # Migration files (generated by drizzle-kit)
│
├── sdk/node/                    # Node SDK (unchanged)
├── tests/
│   └── evals/                   # Migration eval files
│       ├── phase0.test.ts
│       ├── phase1.test.ts
│       ├── phase2.test.ts
│       ├── phase3.test.ts
│       ├── phase4.test.ts
│       ├── phase5.spec.ts       # Playwright
│       ├── phase6.test.ts
│       └── helpers.ts           # createTestContext, etc.
│
├── docs/                        # All documentation
├── .env.example
├── drizzle.config.ts
├── vitest.config.ts
├── playwright.config.ts
└── package.json
```

---

## Success Definition

The migration is complete when:

- [ ] All 54 deterministic evals pass (pass^3 = 100%)
- [ ] All 4 fuzzy evals score >= 4/5 average
- [ ] All 27 Node SDK tests pass
- [ ] All 29 Playwright browser evals pass
- [ ] `bun run build` succeeds
- [ ] `bunx tsc --noEmit` succeeds
- [ ] No Go files in the repository
- [ ] No references to port 8080 in frontend code
- [ ] Single `bun dev` starts the entire app
- [ ] Documentation updated (README, .env.example, deployment docs)

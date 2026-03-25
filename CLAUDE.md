# Ondapile Project Rules

> Project-level instructions. Always loaded for this repo.

## Development Methodology (MANDATORY)

Read `docs/DEV_METHODOLOGY.md` for the full methodology. These are the non-negotiable rules:

### Build Order: Type → Test → Implement → Verify

Never reverse this sequence. For every feature:

1. **Type** — Define Zod schemas or TypeScript interfaces first (2 min)
2. **Test** — Write the eval that defines "done" (3 min). Run it. It should FAIL.
3. **Implement** — Fill in the logic. The type constrains the shape, the test constrains the behavior. (5 min)
4. **Verify** — Run the test. It passes. Run smoke + regression. Still passes. (10 sec)
5. **Integrate** — Wire to the layer above. Commit. (2 min)

### Seven Habits

1. **Read 2 min before writing** — Read the closest existing code. Copy its pattern.
2. **Copy, don't invent** — Never create new patterns. Find existing code and adapt it.
3. **Smallest vertical slice** — Ship one complete feature (DB + API + test), not one complete layer.
4. **One moving part** — Change one thing, test, repeat. Never change 3 things then debug.
5. **Inside-out** — DB schema → API handler → Frontend page. Not the reverse.
6. **Fail fast, fix fast** — The moment a test fails, stop and fix it. Don't skip ahead.
7. **Evidence over confidence** — Run it and see. Never say "this should work."

### Error Prevention (from docs/AGENT_RULES.md)

- **Verify before write** — Test assumptions with 3-line proofs before writing 50-line implementations
- **Verify state after mutation** — `grep` after edit, `git diff` before commit
- **Check existing code first** — `grep -r "functionName" src/` before creating a new one
- **Run tests after EVERY change** — Not at the end. After every single change.
- **One thing per agent** — One function + its test = max scope per delegation

## Project Architecture

- **Frontend**: TanStack Start (port 3000) — React 19, shadcn/ui, Better Auth client
- **Backend**: Go + Gin (port 8080) — being migrated to Hono + Drizzle (see docs/MIGRATION_PLAN.md)
- **Auth**: Better Auth (org, admin, apiKey plugins) — Drizzle adapter
- **Database**: PostgreSQL (single instance, shared between BA and API)
- **SDK**: Node.js (`sdk/node/`)

## Key Reference Docs

| Doc | Purpose | When to Read |
|-----|---------|-------------|
| `docs/META_PLAN.md` | Master index of all docs | Start of any session |
| `docs/DEV_METHODOLOGY.md` | How to build fast | Before implementing |
| `docs/AGENT_RULES.md` | How to avoid errors | Before implementing |
| `docs/PRD.md` | What to build | Before planning |
| `docs/COMPLETE_EVALS.md` | Definition of "done" (131 evals) | Before writing tests |
| `docs/SCENARIO_EVALS.md` | Multi-step behavior tests (8 scenarios) | Before integration testing |
| `docs/MIGRATION_PLAN.md` | Go → Hono/Drizzle migration plan | During migration work |
| `docs/MIGRATION_EVALS.md` | Migration-specific evals (57 evals) | During migration work |

## Eval-Driven Development

Every feature maps to evals in `docs/COMPLETE_EVALS.md`. The workflow:

```
1. Find the eval ID for your feature (e.g., T2.1 for draft endpoints)
2. Write the vitest/playwright test matching that eval
3. Run it — it fails (red)
4. Implement until it passes (green)
5. Run Tier 0 + Tier 1 (regression) — still green
6. Commit
```

## Tech Choices

- **Package manager**: `bun` (never npm/yarn/pnpm)
- **Formatting**: Biome (`bunx --bun biome ...`)
- **ORM**: Drizzle (post-migration) / pgx raw SQL (current Go)
- **Validation**: Zod (TypeScript) / struct binding (Go)
- **Testing**: vitest + Playwright (TypeScript) / go test (Go)
- **Data fetching (frontend)**: useSuspenseQuery (never useEffect for fetching)

# Ondapile Development Methodology

> How to build features fast with fewer errors.
> Not a list of rules — a way of thinking that makes the rules unnecessary.
>
> Created: 2026-03-25

---

## The Core Insight

The reason AI agents (and humans) waste time isn't lack of knowledge — it's **wrong sequencing**. You write code before understanding the shape. You build the UI before the API exists. You implement before testing. Every time you sequence wrong, you pay a tax in rework.

The methodology below is a specific sequencing that eliminates most rework.

---

## The Build Order: Inside-Out, Contract-First

```
1. TYPES        Define the shape of data       (2 min)
2. TEST         Write what "done" looks like   (3 min)
3. IMPLEMENT    Fill in the logic              (5 min)
4. VERIFY       Run the test                   (10 sec)
5. INTEGRATE    Wire to the layer above        (2 min)
```

This order works for every layer — database, API, frontend, integration.

### Why this order works

**Types first** means you discover 80% of errors at compile time, not runtime. If the type is wrong, nothing downstream can be right. Defining the type takes 2 minutes. Finding a type mismatch at runtime takes 20 minutes.

**Test second** means you know what "done" looks like before you start. You can't wander if the destination is defined.

**Implement third** means you're filling in blanks, not exploring. The type defines the shape, the test defines the behavior, implementation is just connecting the two.

**Verify fourth** means you know instantly if it works. Not "probably works" — actually works.

**Integrate last** means each layer is proven before it touches the next. The API handler works before the UI calls it. The DB query works before the handler uses it.

---

## Applied to Ondapile: The Development Sequence

### For a new API endpoint (e.g., `POST /api/v1/drafts`)

```
Step 1: TYPE (2 min)
  Define the Zod schema for the request and response:
  
  const CreateDraftRequest = z.object({
    account_id: z.string(),
    to: z.array(EmailAttendeeSchema).optional(),
    subject: z.string().optional(),
    body_html: z.string(),
  });
  
  const DraftResponse = z.object({
    id: z.string(),
    account_id: z.string(),
    subject: z.string().nullable(),
    body_html: z.string(),
    created_at: z.string(),
  });

Step 2: TEST (3 min)
  Write the eval BEFORE implementation:
  
  it('POST /api/v1/drafts creates draft', async () => {
    const res = await app.request('/api/v1/drafts', {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ account_id: testAccountId, body_html: '<p>Draft</p>' }),
    });
    expect(res.status).toBe(201);
    const data = await res.json();
    expect(data.id).toMatch(/^eml_/);
  });
  
  Run it. It fails. Good.

Step 3: IMPLEMENT (5 min)
  Write the handler. The type tells you what to accept.
  The test tells you what to return.
  
  .post('/drafts', zValidator('json', CreateDraftRequest), async (c) => {
    const body = c.req.valid('json');
    const [draft] = await db.insert(emails).values({ ... }).returning();
    return c.json(draft, 201);
  });

Step 4: VERIFY (10 sec)
  Run the test. It passes.

Step 5: INTEGRATE (2 min)
  Mount in the router.
  Run Tier 0 + Tier 1 evals. Everything still passes.
  Commit.
```

Total time: ~12 minutes for a complete, tested, working endpoint.

### For a frontend page (e.g., dashboard/drafts)

```
Step 1: TYPE (2 min)
  The API response type already exists from the API step above.
  Define the component props:
  
  interface DraftsPageProps {
    drafts: Draft[];
    onCreateDraft: () => void;
    onDeleteDraft: (id: string) => void;
  }

Step 2: TEST (3 min)
  Write a Playwright eval:
  
  test('Drafts page loads', async ({ page }) => {
    await page.goto('/dashboard/drafts');
    await expect(page.getByText(/drafts/i)).toBeVisible();
  });
  
  Run it. It 404s. Good.

Step 3: IMPLEMENT (5 min)
  Create the route file. Use useSuspenseQuery to fetch from the API.
  The API already works (step 5 of the API sequence).

Step 4: VERIFY (10 sec)
  Run the Playwright eval. It passes.

Step 5: INTEGRATE (2 min)
  Add to sidebar navigation. Run full Playwright suite.
  Commit.
```

### For a provider adapter (e.g., Gmail.ListFolders)

```
Step 1: TYPE (2 min)
  The Provider interface already defines the method signature.
  Check it:
  
  ListFolders(ctx: Context, accountId: string): Promise<string[]>

Step 2: TEST (3 min)
  Write a test with a mock provider that returns known data:
  
  it('Gmail.ListFolders returns labels', async () => {
    // Mock the Google API response
    mockFetch('https://gmail.googleapis.com/gmail/v1/users/me/labels', {
      labels: [{ name: 'INBOX' }, { name: 'SENT' }, { name: 'DRAFTS' }],
    });
    
    const folders = await gmailAdapter.listFolders(ctx, 'acc_test');
    expect(folders).toEqual(['INBOX', 'SENT', 'DRAFTS']);
  });

Step 3: IMPLEMENT (5 min)
  async listFolders(ctx, accountId) {
    const client = await this.getHttpClient(ctx, accountId);
    const res = await client.get('/gmail/v1/users/me/labels');
    return res.labels.map(l => l.name);
  }

Step 4: VERIFY (10 sec)
  Test passes.

Step 5: INTEGRATE (2 min)
  The handler already calls prov.ListFolders() — it just works now.
  Run email evals. Commit.
```

---

## The Seven Habits That Actually Speed Things Up

### 1. Read for 2 Minutes Before Writing Anything

Before implementing, read:
- The existing code closest to what you're building (copy its pattern)
- The type definition you'll be working with
- The test/eval that defines "done"

**Why:** 2 minutes reading saves 20 minutes debugging. Every time.

### 2. Copy, Don't Invent

Never create a new pattern. Find the closest existing code and copy its structure.

```
WRONG: "I'll create a new way to handle this endpoint"
RIGHT: "I'll copy emails.ts and change the resource name"
```

For ondapile specifically:
- New endpoint? Copy the pattern from `routes/emails.ts`
- New adapter method? Copy from the IMAP adapter (most complete)
- New dashboard page? Copy from `/dashboard/api-keys/`
- New test? Copy from `phase_b_gmail_test.go`

### 3. Smallest Vertical Slice

Don't build "the data layer," then "the API layer," then "the UI layer." Build one complete vertical slice:

```
WRONG:                          RIGHT:
  Day 1: All 10 DB schemas       Day 1: Draft schema + API + test
  Day 2: All 10 API endpoints    Day 2: Permission schema + API + test
  Day 3: All 10 UI pages         Day 3: Billing schema + API + test
  Day 4: Wire everything          (each day ships a working feature)
  Day 5: Debug 30 integration bugs
```

The vertical slice approach means each commit is a complete, tested feature. If day 3 is wrong, you only redo day 3, not everything.

### 4. Type → Test → Implement (Never Reverse)

If you write implementation before the test, you'll write the test to match the implementation (which defeats the purpose). If you write implementation before the type, you'll discover type errors at runtime.

The sequence is non-negotiable: **Type → Test → Implement → Verify**.

### 5. One Moving Part at a Time

When debugging, change ONE thing, then test. Not three things.

When building, complete ONE endpoint, then move to the next. Not half-build five endpoints.

When refactoring, touch ONE file, then run tests. Not refactor a whole directory.

```
WRONG: "I'll refactor the adapter interface and update all 8 adapters"
       → 8 files changed, 3 compilation errors, unclear which change broke what

RIGHT: "I'll add ListFolders to the interface, update IMAP adapter, run tests.
        Then update Gmail adapter, run tests. Then Outlook, run tests."
       → Each step is independently verifiable
```

### 6. Fail Fast, Fix Fast

The moment something doesn't work, stop and fix it. Don't skip ahead hoping it'll work later.

```
WRONG: "The test fails but I'll fix it after I finish the other endpoints"
       → 3 days later: 12 failing tests, unclear which broke what

RIGHT: "The test fails. Let me fix it now." (5 min fix)
       → Every commit has passing tests
```

### 7. Evidence Over Confidence

Never say "this should work." Run it and see.

```
WRONG: "I changed the route, so it should match now"
       (It didn't match. 45 minutes debugging.)

RIGHT: "I changed the route. Let me run a quick curl to verify."
       (curl shows it works. 10 seconds.)
```

---

## Feature Development Playbook

### Step 0: Before You Touch Code

```
□ Read the eval for this feature (COMPLETE_EVALS.md or SCENARIO_EVALS.md)
□ Read the user flow for this feature (user-flows.md)
□ Read the architecture doc for this layer (architecture/)
□ Identify the closest existing code to copy from
□ Run baseline tests — save the current pass count
```

### Step 1: Define the Contract

```
□ Write Zod schemas for request/response (or TypeScript interfaces)
□ Define the Drizzle schema changes (if new table/columns)
□ Run tsc --noEmit to verify types compile
```

### Step 2: Write the Eval

```
□ Write the vitest eval (from COMPLETE_EVALS.md or MIGRATION_EVALS.md)
□ Run it — confirm it FAILS (red)
□ If it passes without code changes, the eval is testing the wrong thing
```

### Step 3: Implement (Inside-Out)

```
□ Database: Drizzle schema change + migration
□ Run eval — still fails (no handler yet), but schema works
□ Handler: Hono route with Zod validation
□ Run eval — passes (green)
□ Provider adapter (if needed): implement the method
□ Run eval — still passes
```

### Step 4: Verify

```
□ Run the feature's specific eval — PASS
□ Run Tier 0 smoke tests — PASS
□ Run Tier 1 regression tests — PASS
□ git diff — only expected files changed
□ Commit
```

### Step 5: Integrate with UI (if applicable)

```
□ Frontend route file (copy from closest existing page)
□ Playwright eval — PASS
□ Navigation link added
□ Full Playwright suite — PASS
□ Commit
```

---

## Problem Selection Order

When you have N features to build, the order matters enormously. Build in this order:

### Priority 1: Things other things depend on

```
Schema changes → before any handler that reads/writes the new table
Auth middleware → before any protected endpoint
Adapter interface → before any provider implementation
```

### Priority 2: Things that are most similar to existing code

```
Draft endpoints (copy from email endpoints) → faster than calendar endpoints (new pattern)
Gmail adapter (copy from IMAP adapter) → faster than WhatsApp adapter (different protocol)
```

### Priority 3: Things that have the tightest feedback loops

```
API endpoints (testable via app.request in 10ms) → before UI pages (testable via Playwright in 3s)
Database queries (testable via Drizzle in 5ms) → before webhooks (testable via HTTP in 1s)
```

### Priority 4: Things that unblock other work

```
Account CRUD (unblocks all provider work) → before email operations
Webhook CRUD (unblocks webhook testing) → before tracking
Auth middleware (unblocks all API tests) → before any endpoint
```

### Applied to Ondapile Migration:

```
1. Drizzle schema (unblocks everything, most copy-able from existing SQL)
2. Better Auth + Drizzle adapter (unblocks auth, tightest loop)
3. Auth middleware (unblocks all API endpoints)
4. Account CRUD (most similar to existing, unblocks providers)
5. Email endpoints (copy from accounts, unblocks provider work)
6. IMAP adapter (most complete existing code to copy)
7. Gmail adapter (copy from IMAP pattern)
8. Outlook adapter (copy from Gmail pattern)
9. Webhook CRUD (simple, unblocks webhook testing)
10. Webhook dispatcher (medium complexity, unblocks tracking)
11. Tracking pixel/link (simple handlers, depends on dispatcher)
12. Chat/message/calendar endpoints (copy from email pattern)
13. Frontend integration (last — everything it calls already works)
```

Each step is 10-15 minutes if you follow the Type → Test → Implement sequence.

---

## When Things Go Wrong

### "I'm stuck on a bug"

```
1. cat the file at the error line — is the code what I think it is?
2. console.log the actual value — is it what I assumed?
3. If stuck for 5 min: revert and try a different approach
4. If stuck for 15 min: write a minimal reproduction (3 lines max)
5. If stuck for 30 min: stop. Describe the problem. Ask for help.
```

### "Tests pass but the feature doesn't work"

```
1. The test is wrong (testing the wrong thing)
2. Run the scenario eval (S1-S8) to test the real behavior
3. Add a missing assertion to the test
```

### "I changed one thing and 10 tests broke"

```
1. git stash (undo everything)
2. Re-apply changes one at a time
3. After each change, run tests
4. The test that breaks = the change that's wrong
```

### "The agent timed out"

```
1. The task was too big. Split it.
2. One function + its test = max scope for one agent
3. Re-submit with a narrower prompt
```

---

## Speed Benchmarks

If you're following this methodology, each feature should take roughly:

| Feature Type | Expected Time | If Taking Longer |
|-------------|--------------|------------------|
| New API endpoint | 10-15 min | Types/test not defined first |
| New adapter method | 5-10 min | Didn't copy from existing adapter |
| New dashboard page | 10-15 min | Didn't copy from existing page |
| New database table | 5 min | Schema definition is straightforward |
| Bug fix | 5-15 min | If > 15 min, you're fixing symptoms not causes |
| Integration test | 3-5 min | Copy from existing test, change the resource |

If a feature takes 2x the expected time, you're probably:
1. Not copying from existing code (inventing instead)
2. Not writing the test first (debugging instead of verifying)
3. Changing too many things at once (can't isolate the problem)

---

## Summary: The 60-Second Version

```
1. Read before write (2 min saves 20 min)
2. Type → Test → Implement → Verify (never reverse)
3. Copy, don't invent (find the closest existing code)
4. One thing at a time (one function, one test, one commit)
5. Inside-out (DB → API → UI, not UI → API → DB)
6. Smallest vertical slice (ship a complete feature, not a complete layer)
7. Evidence over confidence (run it, don't assume)
```

These seven habits turn a 2-hour feature into a 15-minute feature.
Not by typing faster. By not doing the wrong thing first.

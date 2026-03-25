# Agent Error Prevention Rules

> Mandatory protocol for any AI agent working on ondapile.
> These rules exist because every one was learned from a real error in this codebase.
>
> Created: 2026-03-25

---

## The Five Rules

### Rule 1: Verify Before You Write

Before writing ANY code, run a check that proves your assumption.

```
WRONG: Assume a library behaves a certain way → write 50 lines → debug for 30 min
RIGHT: Write a 3-line test → run it → see actual behavior → then write code
```

**Specific checks:**
- Route parameters: test what the framework actually captures before wiring handlers
- Struct fields: `console.log` or `grep` the type definition before accessing `.field`
- Import paths: verify the import resolves before using it
- DB columns: check the migration/schema file before writing queries

**This rule prevents:** 45-min debugging sessions caused by wrong assumptions.

### Rule 2: One Thing Per Agent

If a task can't be described in one sentence with one file, split it.

```
WRONG: "Implement 5 Gmail methods + create e2e tests + update MockProvider"
       → Agent times out at 30 min, partial work, blown context

RIGHT: "Implement Gmail.ListFolders — pattern: see IMAP adapter line 662"
       → Finishes in 5 min, easy to verify, easy to retry if wrong
```

**Sizing guide:**
- 1 function + its test = good scope
- 1 endpoint (handler + route + test) = maximum scope
- 1 full adapter (10 methods) = too much — split into 3-4 tasks
- "Port the email system" = way too much — split into 10+ tasks

### Rule 3: Verify State After Every Mutation

Every time you change something, verify the change took effect.

```bash
# After editing a file:
grep "the thing I changed" path/to/file  # Is it actually there?

# After git commit:
git show HEAD -- path/to/file | grep "the change"  # Did the commit include it?

# After running a migration:
psql -c "\d tablename" | grep "new_column"  # Does the column exist?

# After installing a package:
node -e "require('package')"  # Does the import resolve?
```

**This rule prevents:** Ghost edits (file not saved), stale git commits (wrong version committed), missing dependencies.

### Rule 4: Check Existing Code Before Creating New Code

Search before you create. Duplicates cause compilation errors and confusion.

```bash
# Before creating a helper function:
grep -r "function helperName" src/ tests/

# Before adding a type:
grep -r "type TypeName" src/

# Before creating a test file:
ls tests/  # What already exists?

# Before adding a dependency:
grep "package-name" package.json  # Already installed?
```

**This rule prevents:** Duplicate function definitions, duplicate type names, redundant dependencies.

### Rule 5: Run Tests After Every Change

Not at the end. After EVERY change. One change = one test run.

```bash
# After editing handler code:
bun test --grep "the test for this handler"

# After editing schema:
bun test --grep "T0"  # Smoke tests catch schema issues fast

# After editing anything:
bun test  # Full suite — must still pass

# If you made 3 changes and tests fail:
git stash  # Undo all 3
# Re-apply one at a time, testing between each
```

**This rule prevents:** Debugging 3 interleaved failures when you don't know which change broke what.

---

## Applied: Agent Execution Protocol

```
FOR EACH TASK:

  1. READ the eval definition for this task
     - What does "done" look like?
     - What assertions must pass?

  2. READ the existing code closest to what you need
     - What patterns does the codebase use?
     - What types/imports/helpers already exist?

  3. RUN baseline tests BEFORE making changes
     - Save the output
     - If tests already fail, note which ones (don't fix them unless asked)

  4. IMPLEMENT one function at a time
     - After each function: type check (tsc --noEmit or go build)
     - After each endpoint: run its specific eval
     - After each file: run phase evals

  5. VERIFY your changes
     - grep the file to confirm edits persisted
     - git diff to confirm you changed what you intended
     - No unintended changes to other files

  6. RUN all evals
     - Phase-specific evals (all must pass)
     - Tier 0 smoke tests (regression check)
     - Tier 1 regression tests (nothing broke)

  7. COMMIT only when all evals pass
     - git add only the files you intentionally changed
     - git diff --staged to review before committing
     - Commit message matches the task description

  8. IF SOMETHING FAILS:
     - Read the ACTUAL error (don't guess from memory)
     - cat the failing file to see what's really there
     - Fix the root cause, not the symptom
     - Re-run the failing eval + 3 adjacent evals
     - If stuck after 3 attempts: stop, describe the problem, ask for help
```

---

## Anti-Patterns (Things That ALWAYS Cause Errors)

| Anti-Pattern | What Happens | Instead |
|--------------|-------------|---------|
| "I'll test at the end" | 5 interleaved failures, 30 min debugging | Test after each change |
| "This library probably works like X" | It doesn't, 45 min wasted | Write a 3-line proof first |
| "I'll edit the file and commit" | Edit not persisted, stale commit | grep after edit, diff before commit |
| "I already know this struct's fields" | Wrong field path, runtime error | grep the type definition |
| "One agent can handle all of this" | Timeout at 30 min, partial work | One function per task |
| "I'll create a helper for that" | Duplicate of existing helper | grep first |
| "The test probably passes now" | It doesn't, false confidence | Run it, read the output |
| "I'll fix this and that while I'm here" | Scope creep, untestable changes | One thing at a time |

---

## Error Recovery Protocol

When an eval fails:

```
1. READ the error output completely (don't skim)
2. IDENTIFY: is this YOUR change or a pre-existing failure?
3. If YOUR change:
   a. cat the file at the failing line — is the code what you expect?
   b. Check types — is the function signature what you assumed?
   c. Check the test — is it testing what you think it's testing?
   d. Fix the root cause (not a band-aid)
   e. Re-run JUST that eval
   f. Then re-run ALL evals
4. If PRE-EXISTING:
   a. Note it in your report
   b. Do NOT fix it unless explicitly asked
   c. Continue with your task
5. If STUCK after 3 fix attempts:
   a. Stop
   b. Document: what you tried, what the error says, what you think the cause is
   c. Ask for help — don't keep guessing
```

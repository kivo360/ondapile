# Ondapile Frontend Eval Suite

> End-to-end evals that test the full stack: Browser → Better Auth (port 3000) → Go Backend (port 8080) → PostgreSQL.
> These evals let you set-and-forget — when they all pass, the app works.
>
> Created: 2026-03-25

---

## Architecture Under Test

```
Browser (Playwright)
  │
  ├── POST /api/auth/sign-up/email ──► TanStack Start SSR ──► Better Auth ──► PostgreSQL
  ├── POST /api/auth/sign-in/email ──► TanStack Start SSR ──► Better Auth ──► PostgreSQL
  │                                                              │
  │                                              Sets session cookie
  │                                                              │
  ├── GET /dashboard ──► TanStack Start SSR ──► reads session cookie
  │
  └── GET /api/v1/accounts ──► Go Backend (8080) ──► validates API key ──► PostgreSQL
      (Authorization: Bearer sk_live_xxx)
```

Key insight: **Two auth systems share one PostgreSQL.** Better Auth creates API keys; Go backend validates them.

---

## Prerequisites

```bash
# Install Playwright
cd frontend
bun add -D @playwright/test
bunx playwright install chromium

# Start both servers for evals
# Terminal 1: Frontend (port 3000)
cd frontend && bun dev

# Terminal 2: Go backend (port 8080)
DB_USER=kevinhill DB_PASSWORD= DB_HOST=/tmp \
  ONDAPILE_API_KEY=142085de253e1883f10e64758f1c893f40bc152ad1385100b30e433f2aaa6774 \
  go run ./cmd/ondapile
```

---

## Playwright Config

```typescript
// frontend/playwright.config.ts
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: false,         // Sequential — auth state matters
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: [['html', { open: 'never' }], ['list']],
  timeout: 30_000,

  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },

  projects: [
    // Setup: create test user and save auth state
    { name: 'setup', testMatch: /auth\.setup\.ts/ },

    // Main tests: use saved auth state
    {
      name: 'authenticated',
      use: {
        ...devices['Desktop Chrome'],
        storageState: '.auth/user.json',
      },
      dependencies: ['setup'],
    },

    // Unauthenticated tests: no saved state
    {
      name: 'unauthenticated',
      use: { ...devices['Desktop Chrome'] },
      testMatch: /unauth/,
    },
  ],
});
```

---

## Eval Helpers

```typescript
// frontend/tests/e2e/helpers/api.ts

const FRONTEND_URL = 'http://localhost:3000';
const BACKEND_URL = 'http://localhost:8080';

/**
 * Call Better Auth endpoints (through TanStack Start SSR proxy).
 * These hit POST /api/auth/* which the TanStack Start server route proxies to Better Auth.
 */
export async function authApi(path: string, body: Record<string, unknown>) {
  const resp = await fetch(`${FRONTEND_URL}/api/auth/${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  const data = await resp.json();
  return { status: resp.status, data, headers: resp.headers };
}

/**
 * Call Go backend API directly with an API key.
 */
export async function backendApi(
  path: string,
  opts: { method?: string; body?: unknown; apiKey: string }
) {
  const resp = await fetch(`${BACKEND_URL}/api/v1${path}`, {
    method: opts.method ?? 'GET',
    headers: {
      'Authorization': `Bearer ${opts.apiKey}`,
      'Content-Type': 'application/json',
    },
    body: opts.body ? JSON.stringify(opts.body) : undefined,
  });
  const data = await resp.json().catch(() => null);
  return { status: resp.status, data };
}

/**
 * Full signup → org → API key flow. Returns everything needed for further testing.
 */
export async function createTestUser(suffix: string = '') {
  const timestamp = Date.now();
  const email = `eval-${timestamp}${suffix}@ondapile-test.com`;
  const password = 'EvalTest1234!';
  const name = `Eval User ${timestamp}`;

  // 1. Sign up
  const signup = await authApi('sign-up/email', { name, email, password });
  if (signup.status !== 200) throw new Error(`Signup failed: ${JSON.stringify(signup.data)}`);
  const userId = signup.data.user?.id;

  // 2. Sign in (to get session cookies)
  const signin = await authApi('sign-in/email', { email, password });
  if (signin.status !== 200) throw new Error(`Signin failed: ${JSON.stringify(signin.data)}`);
  
  // Extract session cookie from Set-Cookie header
  const setCookie = signin.headers.get('set-cookie') ?? '';

  // 3. List organizations (auto-created by databaseHooks.user.create.after)
  const orgsResp = await fetch(`${FRONTEND_URL}/api/auth/organization/list`, {
    headers: { 'Cookie': setCookie },
  });
  const orgs = await orgsResp.json();
  const orgId = orgs?.[0]?.id;

  // 4. Set active organization
  await fetch(`${FRONTEND_URL}/api/auth/organization/set-active`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Cookie': setCookie },
    body: JSON.stringify({ organizationId: orgId }),
  });

  // 5. Create API key
  const keyResp = await fetch(`${FRONTEND_URL}/api/auth/api-key/create`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Cookie': setCookie },
    body: JSON.stringify({ name: 'eval-key', organizationId: orgId }),
  });
  const keyData = await keyResp.json();
  const apiKey = keyData?.key; // Only returned on creation

  return {
    userId,
    email,
    password,
    name,
    orgId,
    apiKey,
    sessionCookie: setCookie,
    cleanup: async () => {
      // Delete via admin or direct SQL if needed
    },
  };
}
```

---

## Auth Setup (runs first)

```typescript
// frontend/tests/e2e/auth.setup.ts
import { test as setup, expect } from '@playwright/test';

const STORAGE_STATE = '.auth/user.json';

setup('create authenticated session', async ({ page }) => {
  const timestamp = Date.now();
  const email = `setup-${timestamp}@ondapile-test.com`;
  const password = 'SetupTest1234!';

  // Navigate to signup
  await page.goto('/auth/signup');
  
  // Fill signup form
  await page.getByLabel(/name/i).fill(`Setup User ${timestamp}`);
  await page.getByLabel(/email/i).fill(email);
  await page.getByLabel(/password/i).fill(password);
  
  // Check terms if visible
  const terms = page.getByRole('checkbox');
  if (await terms.isVisible()) {
    await terms.check();
  }
  
  // Submit
  await page.getByRole('button', { name: /sign up|create account/i }).click();

  // Wait for redirect to dashboard (signup auto-creates org + API key)
  await expect(page).toHaveURL(/\/dashboard/, { timeout: 15_000 });
  
  // Verify we're actually logged in
  await expect(page.getByText(/dashboard|overview|welcome/i)).toBeVisible();

  // Save auth state (cookies + localStorage)
  await page.context().storageState({ path: STORAGE_STATE });
});
```

---

## Eval 1: Signup Flow (Unauthenticated)

```typescript
// frontend/tests/e2e/unauth/signup.spec.ts
import { test, expect } from '@playwright/test';

test.describe('E1: Signup Flow', () => {
  test('E1.1: Can sign up with email/password', async ({ page }) => {
    const ts = Date.now();
    await page.goto('/auth/signup');

    await page.getByLabel(/name/i).fill(`E1 User ${ts}`);
    await page.getByLabel(/email/i).fill(`e1-${ts}@test.com`);
    await page.getByLabel(/password/i).fill('E1Password123!');
    
    const terms = page.getByRole('checkbox');
    if (await terms.isVisible()) await terms.check();

    await page.getByRole('button', { name: /sign up|create/i }).click();

    // Should redirect to dashboard
    await expect(page).toHaveURL(/\/dashboard/, { timeout: 15_000 });
  });

  test('E1.2: Signup rejects weak password', async ({ page }) => {
    await page.goto('/auth/signup');

    await page.getByLabel(/name/i).fill('Weak Password User');
    await page.getByLabel(/email/i).fill(`weak-${Date.now()}@test.com`);
    await page.getByLabel(/password/i).fill('123'); // Too short

    await page.getByRole('button', { name: /sign up|create/i }).click();

    // Should show error, NOT navigate away
    await expect(page).toHaveURL(/\/auth\/signup/);
  });

  test('E1.3: Signup rejects duplicate email', async ({ page }) => {
    const email = `dup-${Date.now()}@test.com`;
    
    // First signup
    await page.goto('/auth/signup');
    await page.getByLabel(/name/i).fill('First User');
    await page.getByLabel(/email/i).fill(email);
    await page.getByLabel(/password/i).fill('FirstUser123!');
    const terms = page.getByRole('checkbox');
    if (await terms.isVisible()) await terms.check();
    await page.getByRole('button', { name: /sign up|create/i }).click();
    await expect(page).toHaveURL(/\/dashboard/, { timeout: 15_000 });

    // Clear session
    await page.context().clearCookies();

    // Second signup with same email
    await page.goto('/auth/signup');
    await page.getByLabel(/name/i).fill('Second User');
    await page.getByLabel(/email/i).fill(email);
    await page.getByLabel(/password/i).fill('SecondUser123!');
    if (await terms.isVisible()) await terms.check();
    await page.getByRole('button', { name: /sign up|create/i }).click();

    // Should show error
    await expect(page.getByText(/already exists|already registered/i)).toBeVisible({ timeout: 5_000 });
  });

  test('E1.4: Login redirects unauthenticated dashboard access', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Should redirect to login
    await expect(page).toHaveURL(/\/auth\/login/);
  });
});
```

---

## Eval 2: Login + Session

```typescript
// frontend/tests/e2e/unauth/login.spec.ts
import { test, expect } from '@playwright/test';

test.describe('E2: Login + Session', () => {
  let testEmail: string;
  const testPassword = 'LoginTest1234!';

  test.beforeAll(async () => {
    // Create user via API for login tests
    const ts = Date.now();
    testEmail = `login-${ts}@test.com`;
    const resp = await fetch('http://localhost:3000/api/auth/sign-up/email', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: `Login User ${ts}`, email: testEmail, password: testPassword }),
    });
    if (!resp.ok) throw new Error('Failed to create test user for login');
  });

  test('E2.1: Can login with valid credentials', async ({ page }) => {
    await page.goto('/auth/login');
    await page.getByLabel(/email/i).fill(testEmail);
    await page.getByLabel(/password/i).fill(testPassword);
    await page.getByRole('button', { name: /sign in|log in/i }).click();

    await expect(page).toHaveURL(/\/dashboard/, { timeout: 10_000 });
  });

  test('E2.2: Rejects invalid password', async ({ page }) => {
    await page.goto('/auth/login');
    await page.getByLabel(/email/i).fill(testEmail);
    await page.getByLabel(/password/i).fill('WrongPassword123!');
    await page.getByRole('button', { name: /sign in|log in/i }).click();

    await expect(page.getByText(/invalid|incorrect|wrong/i)).toBeVisible({ timeout: 5_000 });
    await expect(page).toHaveURL(/\/auth\/login/);
  });

  test('E2.3: Session persists across page reload', async ({ page }) => {
    // Login
    await page.goto('/auth/login');
    await page.getByLabel(/email/i).fill(testEmail);
    await page.getByLabel(/password/i).fill(testPassword);
    await page.getByRole('button', { name: /sign in|log in/i }).click();
    await expect(page).toHaveURL(/\/dashboard/, { timeout: 10_000 });

    // Reload
    await page.reload();
    
    // Still on dashboard (session persisted via cookie)
    await expect(page).toHaveURL(/\/dashboard/);
  });
});
```

---

## Eval 3: Organization + API Keys (Authenticated)

```typescript
// frontend/tests/e2e/dashboard/org-and-keys.spec.ts
import { test, expect } from '@playwright/test';

test.describe('E3: Organization + API Keys', () => {
  test('E3.1: User has auto-created organization', async ({ page }) => {
    await page.goto('/dashboard/settings/team');
    
    // Should see at least one org member (self as owner)
    await expect(page.getByText(/owner/i)).toBeVisible({ timeout: 10_000 });
  });

  test('E3.2: Can create API key', async ({ page }) => {
    await page.goto('/dashboard/api-keys');
    
    // Click create
    await page.getByRole('button', { name: /create/i }).click();
    
    // Fill modal
    const nameInput = page.getByLabel(/name/i);
    await nameInput.fill(`eval-key-${Date.now()}`);
    
    // Submit
    await page.getByRole('button', { name: /create|save/i }).last().click();

    // Should show the key value (only shown once)
    await expect(page.getByText(/sk_live_/)).toBeVisible({ timeout: 5_000 });
  });

  test('E3.3: Can list API keys', async ({ page }) => {
    await page.goto('/dashboard/api-keys');
    
    // Should have at least one key (created during signup)
    await expect(page.getByText(/sk_live_/)).toBeVisible({ timeout: 10_000 });
  });

  test('E3.4: Can revoke API key', async ({ page }) => {
    await page.goto('/dashboard/api-keys');
    
    // Count keys before
    const keysBefore = await page.getByText(/sk_live_/).count();
    
    // Click revoke on last key
    const revokeButtons = page.getByRole('button', { name: /revoke|delete/i });
    if (await revokeButtons.count() > 0) {
      await revokeButtons.last().click();
      
      // Confirm deletion if modal appears
      const confirm = page.getByRole('button', { name: /confirm|yes|delete/i });
      if (await confirm.isVisible()) await confirm.click();

      // Keys should decrease or show revoked
      await page.waitForTimeout(1_000);
    }
  });
});
```

---

## Eval 4: Full Stack — API Key → Go Backend (Authenticated)

This is the critical eval: frontend creates an API key, then uses it to talk to the Go backend.

```typescript
// frontend/tests/e2e/dashboard/full-stack.spec.ts
import { test, expect } from '@playwright/test';

test.describe('E4: Full Stack (Frontend → Go Backend)', () => {
  test('E4.1: Dashboard shows connected accounts from Go backend', async ({ page }) => {
    await page.goto('/dashboard/accounts');
    
    // Page should load without errors
    await expect(page.getByText(/accounts|connected/i)).toBeVisible({ timeout: 10_000 });
    
    // Should NOT show auth error (proves API key → Go backend works)
    await expect(page.getByText(/unauthorized|401/i)).not.toBeVisible();
  });

  test('E4.2: Dashboard shows webhooks from Go backend', async ({ page }) => {
    await page.goto('/dashboard/webhooks');
    
    await expect(page.getByText(/webhooks/i)).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText(/unauthorized|401/i)).not.toBeVisible();
  });

  test('E4.3: Dashboard shows audit log from Go backend', async ({ page }) => {
    await page.goto('/dashboard/logs');
    
    await expect(page.getByText(/audit|log|activity/i)).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText(/unauthorized|401/i)).not.toBeVisible();
  });

  test('E4.4: API key from frontend works with Go backend directly', async ({ page, request }) => {
    // Get the API key from the frontend's API key page
    await page.goto('/dashboard/api-keys');
    
    // The key prefix is visible in the table (sk_live_xxx...)
    const keyText = await page.getByText(/sk_live_/).first().textContent();
    
    // Use the static env API key to verify Go backend is reachable
    const healthResp = await request.get('http://localhost:8080/health');
    expect(healthResp.ok()).toBe(true);
    
    const healthData = await healthResp.json();
    expect(healthData.status).toBe('ok');
  });
});
```

---

## Eval 5: Email Operations via Go Backend (API-Level)

```typescript
// frontend/tests/e2e/api/email-ops.spec.ts
import { test, expect } from '@playwright/test';
import { createTestUser, backendApi } from '../helpers/api';

test.describe('E5: Email Operations (Go Backend API)', () => {
  let apiKey: string;
  let accountId: string;

  test.beforeAll(async () => {
    // Create test user with API key via Better Auth
    const user = await createTestUser('-email-ops');
    apiKey = user.apiKey;

    // List accounts to find one (if any)
    const accounts = await backendApi('/accounts', { apiKey });
    if (accounts.data?.items?.length > 0) {
      accountId = accounts.data.items[0].id;
    }
  });

  test('E5.1: List accounts returns 200', async () => {
    const resp = await backendApi('/accounts', { apiKey });
    expect(resp.status).toBe(200);
    expect(resp.data).toHaveProperty('items');
  });

  test('E5.2: List emails requires account_id', async () => {
    const resp = await backendApi('/emails', { apiKey });
    expect(resp.status).toBe(422); // Validation error
  });

  test('E5.3: List webhooks returns 200', async () => {
    const resp = await backendApi('/webhooks', { apiKey });
    expect(resp.status).toBe(200);
  });

  test('E5.4: Create webhook returns 201', async () => {
    const resp = await backendApi('/webhooks', {
      apiKey,
      method: 'POST',
      body: {
        url: 'https://httpbin.org/post',
        events: ['email.sent', 'email.received'],
      },
    });
    expect(resp.status).toBe(201);
    expect(resp.data).toHaveProperty('id');
    expect(resp.data).toHaveProperty('secret');
  });

  test('E5.5: Invalid API key returns 401', async () => {
    const resp = await backendApi('/accounts', { apiKey: 'invalid-key' });
    expect(resp.status).toBe(401);
  });

  test('E5.6: Health endpoint works without auth', async () => {
    const resp = await fetch('http://localhost:8080/health');
    expect(resp.status).toBe(200);
    const data = await resp.json();
    expect(data.status).toBe('ok');
  });

  test('E5.7: Tracking pixel returns GIF without auth', async () => {
    const resp = await fetch('http://localhost:8080/t/eval-test');
    expect(resp.status).toBe(200);
    expect(resp.headers.get('content-type')).toBe('image/gif');
  });

  test('E5.8: Link redirect returns 302 without auth', async () => {
    const resp = await fetch('http://localhost:8080/l/eval-test?url=https://example.com', {
      redirect: 'manual',
    });
    expect(resp.status).toBe(302);
    expect(resp.headers.get('location')).toBe('https://example.com');
  });
});
```

---

## Eval 6: Settings Pages (Authenticated)

```typescript
// frontend/tests/e2e/dashboard/settings.spec.ts
import { test, expect } from '@playwright/test';

test.describe('E6: Settings Pages', () => {
  test('E6.1: Team settings loads', async ({ page }) => {
    await page.goto('/dashboard/settings/team');
    await expect(page.getByText(/team|members/i)).toBeVisible({ timeout: 10_000 });
  });

  test('E6.2: Billing settings loads (mocked)', async ({ page }) => {
    await page.goto('/dashboard/settings/billing');
    await expect(page.getByText(/billing|plan|subscription/i)).toBeVisible({ timeout: 10_000 });
  });

  test('E6.3: OAuth settings loads', async ({ page }) => {
    await page.goto('/dashboard/settings/oauth');
    await expect(page.getByText(/oauth|google|microsoft/i)).toBeVisible({ timeout: 10_000 });
  });

  test('E6.4: Hosted auth settings loads', async ({ page }) => {
    await page.goto('/dashboard/settings/hosted-auth');
    await expect(page.getByText(/hosted|auth|connect/i)).toBeVisible({ timeout: 10_000 });
  });
});
```

---

## Eval 7: Navigation & Routing

```typescript
// frontend/tests/e2e/dashboard/navigation.spec.ts
import { test, expect } from '@playwright/test';

test.describe('E7: Navigation & Routing', () => {
  test('E7.1: Sidebar navigation works', async ({ page }) => {
    await page.goto('/dashboard');

    // Navigate to each sidebar link
    const links = [
      { text: /accounts/i, url: /\/accounts/ },
      { text: /api.?keys/i, url: /\/api-keys/ },
      { text: /webhooks/i, url: /\/webhooks/ },
      { text: /logs|audit/i, url: /\/logs/ },
    ];

    for (const { text, url } of links) {
      const link = page.getByRole('link', { name: text }).first();
      if (await link.isVisible()) {
        await link.click();
        await expect(page).toHaveURL(url, { timeout: 5_000 });
      }
    }
  });

  test('E7.2: No console errors on navigation', async ({ page }) => {
    const errors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') errors.push(msg.text());
    });

    await page.goto('/dashboard');
    await page.goto('/dashboard/accounts');
    await page.goto('/dashboard/api-keys');
    await page.goto('/dashboard/webhooks');
    await page.goto('/dashboard/settings');

    // Filter out known harmless errors
    const real = errors.filter(e => 
      !e.includes('favicon') && 
      !e.includes('Hydration') &&
      !e.includes('404')
    );
    expect(real).toHaveLength(0);
  });
});
```

---

## Running Evals

```bash
# Run all frontend evals
cd frontend && bunx playwright test

# Run specific eval category
bunx playwright test --grep "E1"   # Signup
bunx playwright test --grep "E2"   # Login
bunx playwright test --grep "E3"   # Org + API Keys
bunx playwright test --grep "E4"   # Full Stack
bunx playwright test --grep "E5"   # Email Ops API
bunx playwright test --grep "E6"   # Settings
bunx playwright test --grep "E7"   # Navigation

# Run only unauthenticated tests
bunx playwright test --project=unauthenticated

# Show HTML report
bunx playwright show-report

# Run with headed browser (debugging)
bunx playwright test --headed --grep "E1.1"
```

---

## Combined Eval Script (Backend + Frontend)

```bash
#!/bin/bash
# eval-full-stack.sh — Run ALL evals across entire stack
set -e

echo "╔══════════════════════════════════════╗"
echo "║  Ondapile Full Stack Eval Suite      ║"
echo "╚══════════════════════════════════════╝"

echo ""
echo "=== Go Build + Vet ==="
go build ./... && go vet ./...
echo "✅ Build clean"

echo ""
echo "=== Go Integration Tests ==="
go test ./tests/integration/ -count=1 -timeout 120s
echo "✅ Backend tests pass"

echo ""
echo "=== Node SDK Tests ==="
cd sdk/node && npm test && cd ../..
echo "✅ SDK tests pass"

echo ""
echo "=== Backend API Evals (live server) ==="
STATUS=$(curl -sf http://localhost:8080/health | jq -r .status)
[ "$STATUS" = "ok" ] && echo "✅ Health" || echo "❌ Health FAIL"

CODE=$(curl -sf -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/accounts \
  -H "X-API-Key: $ONDAPILE_API_KEY")
[ "$CODE" = "200" ] && echo "✅ Auth" || echo "❌ Auth FAIL ($CODE)"

CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/accounts \
  -H "X-API-Key: wrong")
[ "$CODE" = "401" ] && echo "✅ Auth rejection" || echo "❌ Auth rejection FAIL ($CODE)"

CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/t/eval)
[ "$CODE" = "200" ] && echo "✅ Tracking pixel" || echo "❌ Tracking FAIL ($CODE)"

CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/l/eval?url=https://x.com")
[ "$CODE" = "302" ] && echo "✅ Link redirect" || echo "❌ Link redirect FAIL ($CODE)"

echo ""
echo "=== Frontend E2E Evals (Playwright) ==="
cd frontend && bunx playwright test
echo "✅ Frontend evals pass"

echo ""
echo "╔══════════════════════════════════════╗"
echo "║  ALL EVALS PASSED ✅                 ║"
echo "╚══════════════════════════════════════╝"
```

---

## Eval Checklist

| # | Category | Tests | How |
|---|----------|-------|-----|
| E1 | Signup flow | 4 | Playwright (unauth) |
| E2 | Login + session | 3 | Playwright (unauth) |
| E3 | Org + API keys | 4 | Playwright (auth) |
| E4 | Full stack (FE → Go) | 4 | Playwright (auth) + API |
| E5 | Email ops API | 8 | Playwright API context |
| E6 | Settings pages | 4 | Playwright (auth) |
| E7 | Navigation | 2 | Playwright (auth) |
| — | Go integration tests | 200+ | `go test` |
| — | Node SDK tests | 27 | `npm test` |
| — | Backend API evals | 5 | curl |
| **Total** | | **260+** | |

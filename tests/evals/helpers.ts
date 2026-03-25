import type { Hono } from 'hono';

export interface TestContext {
  apiKey: string;
  orgId: string;
  userId: string;
  cookie: string;
  headers: Record<string, string>;
}

/**
 * Creates a test context by:
 * 1. Signing up a unique user
 * 2. Signing in to get session cookie
 * 3. Listing orgs to get orgId
 * 4. Creating API key with orgId
 */
export async function createTestContext(app: Hono): Promise<TestContext> {
  const email = `eval-test-${Date.now()}@test.com`;
  const password = 'TestEval1234!';

  // Sign up
  const signupRes = await app.request('/api/auth/sign-up/email', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: 'Eval User', email, password }),
  });

  if (signupRes.status !== 200) {
    throw new Error(`Signup failed: ${signupRes.status} ${await signupRes.text()}`);
  }

  const signupData = await signupRes.json();
  const userId = signupData.user?.id;

  // Sign in
  const signinRes = await app.request('/api/auth/sign-in/email', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });

  if (signinRes.status !== 200) {
    throw new Error(`Signin failed: ${signinRes.status} ${await signinRes.text()}`);
  }

  const cookie = signinRes.headers.get('set-cookie');
  if (!cookie) {
    throw new Error('No cookie returned from signin');
  }

  // Parse cookie for use in subsequent requests
  const cookieValue = cookie.split(';')[0].split('=').slice(1).join('=');

  // List orgs
  const orgsRes = await app.request('/api/auth/organization/list', {
    headers: { Cookie: cookie },
  });

  if (orgsRes.status !== 200) {
    throw new Error(`List orgs failed: ${orgsRes.status} ${await orgsRes.text()}`);
  }

  const orgs = await orgsRes.json();
  if (!orgs || orgs.length === 0) {
    throw new Error('No organization found after signup');
  }

  const orgId = orgs[0].id;

  // Create API key
  const keyRes = await app.request('/api/auth/api-key/create', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    body: JSON.stringify({ name: 'eval-key', organizationId: orgId }),
  });

  if (keyRes.status !== 200) {
    throw new Error(`API key creation failed: ${keyRes.status} ${await keyRes.text()}`);
  }

  const keyData = await keyRes.json();

  return {
    apiKey: keyData.key,
    orgId,
    userId,
    cookie,
    headers: {
      Authorization: `Bearer ${keyData.key}`,
      Cookie: cookie,
    },
  };
}

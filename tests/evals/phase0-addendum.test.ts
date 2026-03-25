// tests/evals/phase0-addendum.test.ts
import { describe, it, expect } from 'vitest';
import { app } from '../../server/src/app';

describe('Phase 0 Addendum', () => {
  it('P0.11: Created API key authenticates to /api/v1/*', async () => {
    const email = `e2e-${Date.now()}@test.com`;
    await app.request('/api/auth/sign-up/email', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'E2E Test', email, password: 'Test1234!' }),
    });
    const signin = await app.request('/api/auth/sign-in/email', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password: 'Test1234!' }),
    });
    const cookie = signin.headers.get('set-cookie')!;
    const orgs = await app.request('/api/auth/organization/list', {
      headers: { Cookie: cookie },
    });
    const orgId = (await orgs.json())[0].id;
    const keyResp = await app.request('/api/auth/api-key/create', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Cookie: cookie },
      body: JSON.stringify({ name: 'e2e-key', organizationId: orgId }),
    });
    const { key } = await keyResp.json();
    expect(key).toMatch(/^sk_live_/);
    // USE THE KEY immediately
    const apiResp = await app.request('/api/v1/accounts', {
      headers: { Authorization: `Bearer ${key}` },
    });
    expect(apiResp.status).toBe(200);
  });

  it('P0.12: Admin plugin endpoints respond', async () => {
    const res = await app.request('/api/auth/admin/list-users');
    expect([200, 401, 403]).toContain(res.status);
  });
});

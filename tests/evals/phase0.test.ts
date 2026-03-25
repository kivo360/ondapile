// tests/evals/phase0.test.ts
import { describe, it, expect } from 'vitest';
import { app } from '../../server/src/app';
import { db } from '../../server/src/db';
import { accounts, emails, webhooks, auditLog } from '../../server/src/db/schema';
import { sql } from 'drizzle-orm';

describe('Phase 0: Foundation', () => {
  // ── Schema ──
  it('P0.1: Drizzle schema has all app tables', () => {
    const tables = [accounts, emails, webhooks, auditLog];
    for (const t of tables) {
      expect(t).toBeDefined();
    }
  });

  it('P0.2: Database connection works', async () => {
    const result = await db.execute(sql`SELECT 1 as ok`);
    expect(result.rows[0].ok).toBe(1);
  });

  it('P0.3: All tables exist in PostgreSQL', async () => {
    const result = await db.execute(sql`
      SELECT table_name FROM information_schema.tables
      WHERE table_schema = 'public'
      ORDER BY table_name
    `);
    const tableNames = result.rows.map((r: any) => r.table_name);
    const required = ['accounts', 'emails', 'chats', 'messages', 'webhooks',
      'webhook_deliveries', 'attendees', 'oauth_tokens', 'audit_log',
      'calendars', 'calendar_events'];
    for (const t of required) {
      expect(tableNames).toContain(t);
    }
  });

  it('P0.4: Better Auth tables exist', async () => {
    const result = await db.execute(sql`
      SELECT table_name FROM information_schema.tables
      WHERE table_schema = 'public'
      ORDER BY table_name
    `);
    const names = result.rows.map((r: any) => r.table_name);
    expect(names).toContain('user');
    expect(names).toContain('session');
    expect(names).toContain('organization');
    expect(names).toContain('member');
    expect(names).toContain('apikey');
  });

  // ── Hono App ──
  it('P0.5: Hono app boots without error', () => {
    expect(app).toBeDefined();
    expect(typeof app.request).toBe('function');
  });

  it('P0.6: Health endpoint returns ok', async () => {
    const res = await app.request('/health');
    expect(res.status).toBe(200);
    expect(await res.json()).toEqual({ status: 'ok' });
  });

  // ── Better Auth ──
  it('P0.7: Better Auth signup creates user', async () => {
    const res = await app.request('/api/auth/sign-up/email', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: 'Eval User',
        email: `eval-p0-${Date.now()}@test.com`,
        password: 'EvalTest1234!',
      }),
    });
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data.user).toBeDefined();
    expect(data.user.id).toBeTruthy();
  });

  it('P0.8: Better Auth signin returns session', async () => {
    const email = `eval-p0-signin-${Date.now()}@test.com`;
    // Create user first
    await app.request('/api/auth/sign-up/email', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'Signin Test', email, password: 'Test1234!' }),
    });
    // Sign in
    const res = await app.request('/api/auth/sign-in/email', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password: 'Test1234!' }),
    });
    expect(res.status).toBe(200);
    const cookie = res.headers.get('set-cookie');
    expect(cookie).toBeTruthy();
  });

  it('P0.9: Organization auto-created on signup', async () => {
    const email = `eval-p0-org-${Date.now()}@test.com`;
    await app.request('/api/auth/sign-up/email', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'Org Test', email, password: 'Test1234!' }),
    });
    // Sign in to get session
    const signin = await app.request('/api/auth/sign-in/email', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password: 'Test1234!' }),
    });
    const cookie = signin.headers.get('set-cookie')!;
    // List orgs
    const orgs = await app.request('/api/auth/organization/list', {
      headers: { Cookie: cookie },
    });
    expect(orgs.status).toBe(200);
    const data = await orgs.json();
    expect(data.length).toBeGreaterThanOrEqual(1);
  });

  it('P0.10: API key creation works', async () => {
    const email = `eval-p0-key-${Date.now()}@test.com`;
    await app.request('/api/auth/sign-up/email', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'Key Test', email, password: 'Test1234!' }),
    });
    const signin = await app.request('/api/auth/sign-in/email', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password: 'Test1234!' }),
    });
    const cookie = signin.headers.get('set-cookie')!;
    // Get org
    const orgs = await app.request('/api/auth/organization/list', {
      headers: { Cookie: cookie },
    });
    const orgId = (await orgs.json())[0].id;
    // Create key
    const key = await app.request('/api/auth/api-key/create', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Cookie: cookie },
      body: JSON.stringify({ name: 'eval-key', organizationId: orgId }),
    });
    expect(key.status).toBe(200);
    const keyData = await key.json();
    expect(keyData.key).toMatch(/^sk_live_/);
  });
});

// tests/evals/phase1.test.ts
import { describe, it, expect, beforeAll } from 'vitest';
import { app } from '../../server/src/app';
import { db } from '../../server/src/db/client';
import { accounts } from '../../server/src/db/schema';
import { eq } from 'drizzle-orm';
import { createTestContext } from './helpers';

describe('Phase 1: Core API', () => {
  let apiKey: string;

  beforeAll(async () => {
    const ctx = await createTestContext(app);
    apiKey = ctx.apiKey;
  });

  // ── Auth Middleware ──
  it('P1.1: Valid API key returns 200', async () => {
    const res = await app.request('/api/v1/accounts', {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);
  });

  it('P1.2: Invalid API key returns 401', async () => {
    const res = await app.request('/api/v1/accounts', {
      headers: { Authorization: 'Bearer invalid-key' },
    });
    expect(res.status).toBe(401);
  });

  it('P1.3: Missing auth returns 401', async () => {
    const res = await app.request('/api/v1/accounts');
    expect(res.status).toBe(401);
  });

  it('P1.4: X-API-Key header works', async () => {
    const res = await app.request('/api/v1/accounts', {
      headers: { 'X-API-Key': apiKey },
    });
    expect(res.status).toBe(200);
  });

  // ── Health & Metrics ──
  it('P1.5: GET /health returns ok', async () => {
    const res = await app.request('/health');
    expect(res.status).toBe(200);
    expect(await res.json()).toEqual({ status: 'ok' });
  });

  it('P1.6: GET /metrics returns stats', async () => {
    const res = await app.request('/metrics');
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data).toHaveProperty('accounts');
  });

  // ── Account CRUD ──
  it('P1.7: POST /accounts creates account', async () => {
    const res = await app.request('/api/v1/accounts', {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        provider: 'IMAP',
        identifier: `eval-${Date.now()}@test.com`,
        name: 'Eval IMAP',
        credentials: { imap_host: 'imap.gmail.com', imap_port: '993',
          smtp_host: 'smtp.gmail.com', smtp_port: '587',
          username: 'test@test.com', password: 'test' },
      }),
    });
    expect(res.status).toBe(201);
    const data = await res.json();
    expect(data.id).toMatch(/^acc_/);
  });

  it('P1.8: GET /accounts returns list', async () => {
    const res = await app.request('/api/v1/accounts', {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data).toHaveProperty('items');
    expect(Array.isArray(data.items)).toBe(true);
  });

  it('P1.9: GET /accounts/:id returns single account', async () => {
    // Create first
    const create = await app.request('/api/v1/accounts', {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider: 'IMAP', identifier: `eval-get-${Date.now()}@test.com`,
        name: 'Get Test', credentials: {} }),
    });
    const { id } = await create.json();

    const res = await app.request(`/api/v1/accounts/${id}`, {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data.id).toBe(id);
  });

  it('P1.10: DELETE /accounts/:id deletes account', async () => {
    const create = await app.request('/api/v1/accounts', {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider: 'IMAP', identifier: `eval-del-${Date.now()}@test.com`,
        name: 'Delete Test', credentials: {} }),
    });
    const { id } = await create.json();

    const res = await app.request(`/api/v1/accounts/${id}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);

    // Verify deleted
    const get = await app.request(`/api/v1/accounts/${id}`, {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(get.status).toBe(404);
  });

  // ── CORS ──
  it('P1.11: OPTIONS returns CORS headers', async () => {
    const res = await app.request('/api/v1/accounts', {
      method: 'OPTIONS',
      headers: { Origin: 'http://localhost:3000' },
    });
    expect(res.headers.get('access-control-allow-origin')).toBeTruthy();
  });

  // ── Encryption ──
  it('P1.12: Credentials are encrypted in DB', async () => {
    const create = await app.request('/api/v1/accounts', {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider: 'IMAP', identifier: `eval-enc-${Date.now()}@test.com`,
        name: 'Encrypt Test', credentials: { password: 'secret123' } }),
    });
    const { id } = await create.json();

    // Read directly from DB — credentials should be encrypted, not plaintext
    const [row] = await db.select({ creds: accounts.credentialsEnc })
      .from(accounts).where(eq(accounts.id, id));
    expect(row.creds).toBeTruthy();
    // Buffer should not contain the plaintext password
    const credsStr = row.creds!.toString('utf8');
    expect(credsStr).not.toContain('secret123');
  });
});

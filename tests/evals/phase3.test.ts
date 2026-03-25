// tests/evals/phase3.test.ts
import { describe, it, expect, beforeAll } from 'vitest';
import { app } from '../../server/src/app';
import { db } from '../../server/src/db/client';
import { emails } from '../../server/src/db/schema';
import { eq } from 'drizzle-orm';
import { createTestContext } from './helpers';

describe('Phase 3: Webhooks + Tracking', () => {
  let apiKey: string;
  let trackingAccountId: string;

  beforeAll(async () => {
    const ctx = await createTestContext(app);
    apiKey = ctx.apiKey;
    // Create an account for tracking tests (FK constraint requires valid account_id)
    const acc = await app.request('/api/v1/accounts', {
      method: 'POST',
      headers: { Authorization: `Bearer ${ctx.apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider: 'IMAP', identifier: `eval-tracking-${Date.now()}@test.com`,
        name: 'Tracking Test', credentials: {} }),
    });
    trackingAccountId = (await acc.json()).id;
  });

  // ── Webhook CRUD ──
  it('P3.1: POST /webhooks creates webhook', async () => {
    const res = await app.request('/api/v1/webhooks', {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ url: 'https://httpbin.org/post', events: ['email.sent'] }),
    });
    expect(res.status).toBe(201);
    const data = await res.json();
    expect(data.id).toMatch(/^whk_/);
    expect(data.secret).toBeTruthy();
  });

  it('P3.2: GET /webhooks returns list', async () => {
    const res = await app.request('/api/v1/webhooks', {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data.items.length).toBeGreaterThanOrEqual(1);
  });

  it('P3.3: DELETE /webhooks/:id deletes', async () => {
    const create = await app.request('/api/v1/webhooks', {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ url: 'https://httpbin.org/post', events: ['email.sent'] }),
    });
    const { id } = await create.json();
    const res = await app.request(`/api/v1/webhooks/${id}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);
  });

  // ── Tracking ──
  it('P3.4: GET /t/:id returns GIF', async () => {
    const res = await app.request('/t/eval-test');
    expect(res.status).toBe(200);
    expect(res.headers.get('content-type')).toBe('image/gif');
  });

  it('P3.5: GET /l/:id redirects', async () => {
    const res = await app.request('/l/eval-test?url=https://example.com', {
      redirect: 'manual',
    });
    expect(res.status).toBe(302);
    expect(res.headers.get('location')).toBe('https://example.com');
  });

  it('P3.6: GET /l/:id without url returns 400', async () => {
    const res = await app.request('/l/eval-test');
    expect(res.status).toBe(400);
  });

  it('P3.7: Tracking pixel records open in DB', async () => {
    // Seed email with valid account (FK constraint)
    const [email] = await db.insert(emails).values({
      accountId: trackingAccountId, provider: 'IMAP',
      subject: 'Track Me', metadata: {},
    }).returning();
    // Hit pixel
    await app.request(`/t/${email.id}`);
    // Verify tracking updated
    const [updated] = await db.select().from(emails).where(eq(emails.id, email.id));
    expect(updated.tracking).toBeDefined();
    const tracking = updated.tracking as Record<string, unknown>;
    expect(tracking.opens).toBeGreaterThanOrEqual(1);
  });

  // ── Audit Log ──
  it('P3.8: GET /audit-log returns entries', async () => {
    const res = await app.request('/api/v1/audit-log', {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);
  });
});

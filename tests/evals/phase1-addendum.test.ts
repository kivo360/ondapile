// tests/evals/phase1-addendum.test.ts
import { describe, it, expect, beforeAll } from 'vitest';
import { app } from '../../server/src/app';
import { createTestContext } from './helpers';

describe('Phase 1 Addendum', () => {
  let apiKey: string;

  beforeAll(async () => {
    const ctx = await createTestContext(app);
    apiKey = ctx.apiKey;
  });

  it('P1.13: Reconnect returns account', async () => {
    const create = await app.request('/api/v1/accounts', {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider: 'IMAP', identifier: `recon-${Date.now()}@test.com`, name: 'Reconnect Test', credentials: {} }),
    });
    const { id } = await create.json();
    const res = await app.request(`/api/v1/accounts/${id}/reconnect`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect([200, 202]).toContain(res.status);
  });

  it('P1.14: Hosted auth returns valid URL', async () => {
    const res = await app.request('/api/v1/accounts/hosted-auth', {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider: 'gmail', redirect_url: 'https://example.com/cb' }),
    });
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data.url).toMatch(/^https?:\/\//);
  });

  it('P1.15: OAuth callback route exists', async () => {
    const res = await app.request('/api/v1/oauth/callback/gmail');
    expect([302, 400]).toContain(res.status);
    expect(res.status).not.toBe(404);
    expect(res.status).not.toBe(500);
  });

  it('P1.17: QR returns appropriate response', async () => {
    const create = await app.request('/api/v1/accounts', {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider: 'WHATSAPP', identifier: `qr-${Date.now()}`, name: 'QR Test', credentials: {} }),
    });
    const { id } = await create.json();
    const res = await app.request(`/api/v1/accounts/${id}/qr`, {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect([200, 400]).toContain(res.status);
  });

  it('P1.18: Checkpoint accepts code', async () => {
    const create = await app.request('/api/v1/accounts', {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider: 'IMAP', identifier: `cp-${Date.now()}@test.com`, name: 'CP Test', credentials: {} }),
    });
    const { id } = await create.json();
    const res = await app.request(`/api/v1/accounts/${id}/checkpoint`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ code: '123456' }),
    });
    expect([200, 400]).toContain(res.status);
    expect(res.status).not.toBe(404);
  });
});

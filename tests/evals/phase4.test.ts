// tests/evals/phase4.test.ts
import { describe, it, expect, beforeAll } from 'vitest';
import { app } from '../../server/src/app';
import { createTestContext } from './helpers';

describe('Phase 4: Messaging + Calendar Routes', () => {
  let apiKey: string;

  beforeAll(async () => {
    const ctx = await createTestContext(app);
    apiKey = ctx.apiKey;
  });

  // Just verify routes exist and return appropriate responses
  it('P4.1: GET /chats returns 200', async () => {
    const res = await app.request('/api/v1/chats', {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);
  });

  it('P4.2: GET /messages returns 200', async () => {
    const res = await app.request('/api/v1/messages', {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);
  });

  it('P4.3: GET /attendees returns 200', async () => {
    const res = await app.request('/api/v1/attendees', {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);
  });

  it('P4.4: GET /calendars returns 200', async () => {
    const res = await app.request('/api/v1/calendars', {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);
  });

  it('P4.5: POST /search returns 200', async () => {
    const res = await app.request('/api/v1/search', {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ query: 'test' }),
    });
    expect(res.status).toBe(200);
  });
});

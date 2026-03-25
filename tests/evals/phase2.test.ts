// tests/evals/phase2.test.ts
import { describe, it, expect, beforeAll } from 'vitest';
import { app } from '../../server/src/app';
import { db } from '../../server/src/db/client';
import { emails } from '../../server/src/db/schema';
import { createTestContext } from './helpers';

describe('Phase 2: Email Operations', () => {
  let apiKey: string;
  let accountId: string;
  let testEmailId: string;

  beforeAll(async () => {
    const ctx = await createTestContext(app);
    apiKey = ctx.apiKey;
    // Create IMAP account
    const acc = await app.request('/api/v1/accounts', {
      method: 'POST',
      headers: { Authorization: `Bearer ${ctx.apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider: 'IMAP', identifier: `eval-email-${Date.now()}@test.com`,
        name: 'Email Test', credentials: {} }),
    });
    accountId = (await acc.json()).id;
    // Seed a test email directly in DB
    const [email] = await db.insert(emails).values({
      accountId, provider: 'IMAP', subject: 'Eval Test Email',
      body: '<p>Hello</p>', bodyPlain: 'Hello',
      fromAttendee: { displayName: 'Sender', identifier: 'sender@test.com', identifierType: 'EMAIL_ADDRESS' },
      toAttendees: [{ displayName: 'Recipient', identifier: 'recip@test.com', identifierType: 'EMAIL_ADDRESS' }],
      metadata: {},
    }).returning();
    testEmailId = email.id;
  });

  it('P2.1: GET /emails?account_id=X returns list', async () => {
    const res = await app.request(`/api/v1/emails?account_id=${accountId}`, {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data.items.length).toBeGreaterThanOrEqual(1);
  });

  it('P2.2: GET /emails requires account_id', async () => {
    const res = await app.request('/api/v1/emails', {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(422);
  });

  it('P2.3: GET /emails/:id returns email', async () => {
    const res = await app.request(`/api/v1/emails/${testEmailId}?account_id=${accountId}`, {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data.subject).toBe('Eval Test Email');
  });

  it('P2.4: PUT /emails/:id updates read status', async () => {
    const res = await app.request(`/api/v1/emails/${testEmailId}`, {
      method: 'PUT',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ read: true }),
    });
    expect(res.status).toBe(200);
  });

  it('P2.5: PUT /emails/:id updates starred', async () => {
    const res = await app.request(`/api/v1/emails/${testEmailId}`, {
      method: 'PUT',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ starred: true }),
    });
    expect(res.status).toBe(200);
  });

  it('P2.6: GET /emails/folders returns folders', async () => {
    const res = await app.request(`/api/v1/emails/folders?account_id=${accountId}`, {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);
  });

  it('P2.7: POST /emails/:id/reply returns email', async () => {
    const res = await app.request(`/api/v1/emails/${testEmailId}/reply`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ account_id: accountId, body_html: '<p>Reply</p>' }),
    });
    expect(res.status).toBe(200);
  });

  it('P2.8: POST /emails/:id/forward returns email', async () => {
    const res = await app.request(`/api/v1/emails/${testEmailId}/forward`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        account_id: accountId,
        to: [{ identifier: 'fwd@test.com', identifier_type: 'EMAIL_ADDRESS' }],
        body_html: '<p>FYI</p>',
      }),
    });
    expect(res.status).toBe(200);
  });

  it('P2.9: DELETE /emails/:id deletes email', async () => {
    // Create a throwaway email
    const [e] = await db.insert(emails).values({
      accountId, provider: 'IMAP', subject: 'Delete Me', metadata: {},
    }).returning();
    const res = await app.request(`/api/v1/emails/${e.id}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);
  });

  it('P2.10: GET /emails?q=keyword searches', async () => {
    const res = await app.request(`/api/v1/emails?account_id=${accountId}&q=Eval`, {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(res.status).toBe(200);
  });
});

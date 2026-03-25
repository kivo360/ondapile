import { Hono } from 'hono';
import { db } from '../db/client';
import { emails, accounts } from '../db/schema';
import { eq, and } from 'drizzle-orm';

export const draftsRoute = new Hono();

// Helper: verify account belongs to org
async function verifyAccountOwnership(accountId: string, orgId: string): Promise<boolean> {
  const [acc] = await db.select({ id: accounts.id })
    .from(accounts)
    .where(and(eq(accounts.id, accountId), eq(accounts.organizationId, orgId)));
  return !!acc;
}

// POST /drafts - create draft
draftsRoute.post('/drafts', async (c) => {
  const orgId = c.get('orgId') as string;
  const body = await c.req.json();
  const accountId = body.account_id;

  if (accountId && !(await verifyAccountOwnership(accountId, orgId))) {
    return c.json({ error: 'Account not found' }, 404);
  }

  const [draft] = await db.insert(emails).values({
    accountId: accountId || '',
    provider: 'IMAP',
    subject: body.subject || '',
    body: body.body_html || '',
    toAttendees: body.to || [],
    role: 'draft',
    metadata: {},
  }).returning();

  return c.json(draft, 201);
});

// GET /drafts - list drafts
draftsRoute.get('/drafts', async (c) => {
  const orgId = c.get('orgId') as string;
  const accountId = c.req.query('account_id');

  if (accountId && !(await verifyAccountOwnership(accountId, orgId))) {
    return c.json({ error: 'Account not found' }, 404);
  }

  let rows;
  if (accountId) {
    rows = await db.select().from(emails)
      .where(and(eq(emails.accountId, accountId), eq(emails.role, 'draft')));
  } else {
    rows = await db.select().from(emails)
      .where(eq(emails.role, 'draft'));
  }

  return c.json({ items: rows });
});

// GET /drafts/:id
draftsRoute.get('/drafts/:id', async (c) => {
  const id = c.req.param('id');

  const [draft] = await db.select().from(emails)
    .where(and(eq(emails.id, id), eq(emails.role, 'draft')));

  if (!draft) return c.json({ error: 'Not found' }, 404);
  return c.json(draft);
});

// PUT /drafts/:id - update draft
draftsRoute.put('/drafts/:id', async (c) => {
  const id = c.req.param('id');
  const body = await c.req.json();

  const [existing] = await db.select().from(emails)
    .where(and(eq(emails.id, id), eq(emails.role, 'draft')));
  if (!existing) return c.json({ error: 'Not found' }, 404);

  await db.update(emails).set({
    body: body.body_html !== undefined ? body.body_html : existing.body,
    subject: body.subject !== undefined ? body.subject : existing.subject,
    toAttendees: body.to !== undefined ? body.to : existing.toAttendees,
    updatedAt: new Date(),
  }).where(eq(emails.id, id));

  const [updated] = await db.select().from(emails).where(eq(emails.id, id));
  return c.json(updated);
});

// POST /drafts/:id/send - send draft
draftsRoute.post('/drafts/:id/send', async (c) => {
  const id = c.req.param('id');

  const [draft] = await db.select().from(emails)
    .where(and(eq(emails.id, id), eq(emails.role, 'draft')));
  if (!draft) return c.json({ error: 'Not found' }, 404);

  // Move from draft to sent
  await db.update(emails).set({
    role: 'sent',
    dateSent: new Date(),
    updatedAt: new Date(),
  }).where(eq(emails.id, id));

  const [sent] = await db.select().from(emails).where(eq(emails.id, id));
  return c.json(sent);
});

// DELETE /drafts/:id
draftsRoute.delete('/drafts/:id', async (c) => {
  const id = c.req.param('id');

  const [existing] = await db.select().from(emails)
    .where(and(eq(emails.id, id), eq(emails.role, 'draft')));
  if (!existing) return c.json({ error: 'Not found' }, 404);

  await db.delete(emails).where(eq(emails.id, id));
  return c.json({ deleted: true });
});

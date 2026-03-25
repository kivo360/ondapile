import { Hono } from 'hono';
import { db } from '../db/client';
import { emails, accounts } from '../db/schema';
import { eq, and, ilike, or } from 'drizzle-orm';
import { dispatchWebhook } from '../lib/webhook-dispatcher';

export const emailsRoute = new Hono();

// Helper: verify account belongs to org
async function verifyAccountOwnership(accountId: string, orgId: string): Promise<boolean> {
  const [acc] = await db.select({ id: accounts.id })
    .from(accounts)
    .where(and(eq(accounts.id, accountId), eq(accounts.organizationId, orgId)));
  return !!acc;
}

// GET /emails/folders - MUST be before /emails/:id
emailsRoute.get('/emails/folders', async (c) => {
  const orgId = c.get('orgId') as string;
  const accountId = c.req.query('account_id');
  if (!accountId) return c.json({ error: 'account_id required' }, 422);

  if (!(await verifyAccountOwnership(accountId, orgId))) {
    return c.json({ error: 'Account not found' }, 404);
  }

  const rows = await db.select({ folders: emails.folders })
    .from(emails).where(eq(emails.accountId, accountId));

  const folderSet = new Set<string>();
  for (const row of rows) {
    if (Array.isArray(row.folders)) {
      for (const f of row.folders) folderSet.add(f);
    }
  }

  return c.json({ items: Array.from(folderSet) });
});

// POST /emails - send email
emailsRoute.post('/emails', async (c) => {
  const orgId = c.get('orgId') as string;
  const body = await c.req.json();
  const accountId = body.account_id;
  if (!accountId) return c.json({ error: 'account_id required' }, 422);
  if (!(await verifyAccountOwnership(accountId, orgId))) return c.json({ error: 'Not found' }, 404);

  const [sent] = await db.insert(emails).values({
    accountId,
    provider: 'IMAP',
    subject: body.subject || '',
    body: body.body_html || '',
    toAttendees: body.to || [],
    role: 'sent',
    dateSent: new Date(),
    metadata: {},
  }).returning();

  // Dispatch email.sent webhook
  await dispatchWebhook(orgId, 'email.sent', { emailId: sent.id, accountId });

  return c.json(sent);
});

// GET /emails - list emails for account
emailsRoute.get('/emails', async (c) => {
  const orgId = c.get('orgId') as string;
  const accountId = c.req.query('account_id');

  if (!accountId) return c.json({ error: 'account_id required' }, 422);

  if (!(await verifyAccountOwnership(accountId, orgId))) {
    return c.json({ error: 'Account not found' }, 404);
  }

  const q = c.req.query('q');

  let rows;
  if (q) {
    rows = await db.select().from(emails)
      .where(and(
        eq(emails.accountId, accountId),
        or(
          ilike(emails.subject, `%${q}%`),
          ilike(emails.bodyPlain, `%${q}%`),
          ilike(emails.body, `%${q}%`),
        )
      ));
  } else {
    rows = await db.select().from(emails)
      .where(eq(emails.accountId, accountId));
  }

  return c.json({ items: rows });
});

// GET /emails/:id
emailsRoute.get('/emails/:id', async (c) => {
  const orgId = c.get('orgId') as string;
  const id = c.req.param('id');
  const accountId = c.req.query('account_id');

  if (!accountId) return c.json({ error: 'account_id required' }, 422);

  if (!(await verifyAccountOwnership(accountId, orgId))) {
    return c.json({ error: 'Account not found' }, 404);
  }

  const [email] = await db.select().from(emails)
    .where(and(eq(emails.id, id), eq(emails.accountId, accountId)));

  if (!email) return c.json({ error: 'Not found' }, 404);
  return c.json(email);
});

// PUT /emails/:id - update read/starred
emailsRoute.put('/emails/:id', async (c) => {
  const orgId = c.get('orgId') as string;
  const id = c.req.param('id');
  const body = await c.req.json();

  const [existing] = await db.select().from(emails).where(eq(emails.id, id));
  if (!existing) return c.json({ error: 'Not found' }, 404);

  if (!(await verifyAccountOwnership(existing.accountId, orgId))) {
    return c.json({ error: 'Not found' }, 404);
  }

  await db.update(emails).set({
    isRead: body.read !== undefined ? body.read : existing.isRead,
    readDate: body.read ? new Date() : existing.readDate,
    metadata: body.starred !== undefined
      ? { ...((existing.metadata || {}) as Record<string, unknown>), starred: body.starred }
      : existing.metadata,
    folders: body.folder ? [body.folder] : existing.folders,
    updatedAt: new Date(),
  }).where(eq(emails.id, id));

  const [updated] = await db.select().from(emails).where(eq(emails.id, id));

  // Dispatch email.moved webhook if folder changed
  if (body.folder) {
    await dispatchWebhook(orgId, 'email.moved', { emailId: id, folder: body.folder });
  }

  return c.json(updated);
});

// POST /emails/:id/reply
emailsRoute.post('/emails/:id/reply', async (c) => {
  const orgId = c.get('orgId') as string;
  const id = c.req.param('id');
  const body = await c.req.json();
  const accountId = body.account_id;
  if (!accountId) return c.json({ error: 'account_id required' }, 422);
  if (!(await verifyAccountOwnership(accountId, orgId))) return c.json({ error: 'Not found' }, 404);

  const [original] = await db.select().from(emails)
    .where(and(eq(emails.id, id), eq(emails.accountId, accountId)));
  if (!original) return c.json({ error: 'Not found' }, 404);

  const [reply] = await db.insert(emails).values({
    accountId,
    provider: original.provider,
    subject: `Re: ${original.subject || ''}`,
    body: body.body_html || '',
    role: 'sent',
    metadata: { inReplyTo: id },
  }).returning();

  // Dispatch email.sent webhook
  await dispatchWebhook(orgId, 'email.sent', { emailId: reply.id, accountId });

  return c.json(reply);
});

// POST /emails/:id/forward
emailsRoute.post('/emails/:id/forward', async (c) => {
  const orgId = c.get('orgId') as string;
  const id = c.req.param('id');
  const body = await c.req.json();
  const accountId = body.account_id;
  if (!accountId) return c.json({ error: 'account_id required' }, 422);
  if (!(await verifyAccountOwnership(accountId, orgId))) return c.json({ error: 'Not found' }, 404);

  const [original] = await db.select().from(emails)
    .where(and(eq(emails.id, id), eq(emails.accountId, accountId)));
  if (!original) return c.json({ error: 'Not found' }, 404);

  const [forwarded] = await db.insert(emails).values({
    accountId,
    provider: original.provider,
    subject: `Fwd: ${original.subject || ''}`,
    body: body.body_html || '',
    toAttendees: body.to || [],
    role: 'sent',
    metadata: { forwardOf: id },
  }).returning();

  // Dispatch email.sent webhook
  await dispatchWebhook(orgId, 'email.sent', { emailId: forwarded.id, accountId });

  return c.json(forwarded);
});

// DELETE /emails/:id
emailsRoute.delete('/emails/:id', async (c) => {
  const orgId = c.get('orgId') as string;
  const id = c.req.param('id');

  const [existing] = await db.select({ accountId: emails.accountId })
    .from(emails).where(eq(emails.id, id));
  if (!existing) return c.json({ error: 'Not found' }, 404);

  if (!(await verifyAccountOwnership(existing.accountId, orgId))) {
    return c.json({ error: 'Not found' }, 404);
  }

  await db.delete(emails).where(eq(emails.id, id));
  return c.json({ deleted: true });
});

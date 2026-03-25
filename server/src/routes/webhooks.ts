import { Hono } from 'hono';
import { db } from '../db/client';
import { webhooks } from '../db/schema';
import { eq, and } from 'drizzle-orm';
import crypto from 'crypto';

export const webhooksRoute = new Hono();

// POST /webhooks
webhooksRoute.post('/webhooks', async (c) => {
  const orgId = c.get('orgId') as string;
  const body = await c.req.json();

  const id = 'whk_' + crypto.randomUUID().replace(/-/g, '');
  const secret = crypto.randomBytes(32).toString('hex');

  const [created] = await db.insert(webhooks).values({
    id,
    url: body.url,
    events: body.events || [],
    secret,
    organizationId: orgId,
  }).returning();

  return c.json(created, 201);
});

// GET /webhooks
webhooksRoute.get('/webhooks', async (c) => {
  const orgId = c.get('orgId') as string;

  const rows = await db.select().from(webhooks)
    .where(eq(webhooks.organizationId, orgId));

  return c.json({ items: rows });
});

// DELETE /webhooks/:id
webhooksRoute.delete('/webhooks/:id', async (c) => {
  const orgId = c.get('orgId') as string;
  const id = c.req.param('id');

  const [deleted] = await db.delete(webhooks)
    .where(and(eq(webhooks.id, id), eq(webhooks.organizationId, orgId)))
    .returning();

  if (!deleted) return c.json({ error: 'Not found' }, 404);
  return c.json({ deleted: true });
});

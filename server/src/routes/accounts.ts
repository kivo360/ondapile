import { Hono } from 'hono';
import { db } from '../db/client';
import { accounts } from '../db/schema';
import { eq, and } from 'drizzle-orm';
import { encrypt } from '../lib/crypto';
import crypto from 'crypto';

export const accountsRoute = new Hono();

// GET /accounts - list accounts scoped to org
accountsRoute.get('/accounts', async (c) => {
  const orgId = c.get('orgId') as string;

  const rows = await db.select().from(accounts)
    .where(eq(accounts.organizationId, orgId));

  // Strip credentials from response
  const items = rows.map(({ credentialsEnc, ...rest }) => rest);

  return c.json({ items });
});

// POST /accounts - create new account
accountsRoute.post('/accounts', async (c) => {
  const orgId = c.get('orgId') as string;
  const body = await c.req.json();

  const { provider, identifier, name, credentials, metadata } = body;

  // Encrypt credentials if provided
  let credentialsEnc: Buffer | null = null;
  if (credentials && Object.keys(credentials).length > 0) {
    credentialsEnc = encrypt(JSON.stringify(credentials));
  }

  const id = 'acc_' + crypto.randomUUID().replace(/-/g, '');

  const [created] = await db.insert(accounts).values({
    id,
    provider,
    identifier,
    name,
    status: 'CONNECTING',
    credentialsEnc,
    metadata: metadata || {},
    organizationId: orgId,
  }).returning();

  // Return without credentials
  const { credentialsEnc: _, ...result } = created;
  return c.json(result, 201);
});

// GET /accounts/:id - get single account
accountsRoute.get('/accounts/:id', async (c) => {
  const orgId = c.get('orgId') as string;
  const id = c.req.param('id');

  const [row] = await db.select().from(accounts)
    .where(and(eq(accounts.id, id), eq(accounts.organizationId, orgId)));

  if (!row) {
    return c.json({ error: 'Not found' }, 404);
  }

  const { credentialsEnc, ...result } = row;
  return c.json(result);
});

// DELETE /accounts/:id - delete account
accountsRoute.delete('/accounts/:id', async (c) => {
  const orgId = c.get('orgId') as string;
  const id = c.req.param('id');

  const [deleted] = await db.delete(accounts)
    .where(and(eq(accounts.id, id), eq(accounts.organizationId, orgId)))
    .returning();

  if (!deleted) {
    return c.json({ error: 'Not found' }, 404);
  }

  return c.json({ deleted: true });
});

// POST /accounts/hosted-auth - generate hosted auth URL
accountsRoute.post('/accounts/hosted-auth', async (c) => {
  const body = await c.req.json();
  const token = crypto.randomBytes(32).toString('hex');
  const baseUrl = process.env.BETTER_AUTH_URL || 'http://localhost:3000';
  const redirectUrl = body.redirect_url || `${baseUrl}/oauth/success`;
  const url = `${baseUrl}/connect/${token}?provider=${body.provider || 'gmail'}&redirect=${encodeURIComponent(redirectUrl)}`;
  return c.json({ url, token });
});

// POST /accounts/:id/reconnect
accountsRoute.post('/accounts/:id/reconnect', async (c) => {
  const orgId = c.get('orgId') as string;
  const id = c.req.param('id');
  const [row] = await db.select().from(accounts)
    .where(and(eq(accounts.id, id), eq(accounts.organizationId, orgId)));
  if (!row) return c.json({ error: 'Not found' }, 404);
  // Update status to CONNECTING
  await db.update(accounts).set({ status: 'CONNECTING', updatedAt: new Date() }).where(eq(accounts.id, id));
  const [updated] = await db.select().from(accounts).where(eq(accounts.id, id));
  const { credentialsEnc: _, ...result } = updated;
  return c.json(result);
});

// GET /accounts/:id/qr - QR code for WhatsApp
accountsRoute.get('/accounts/:id/qr', async (c) => {
  const orgId = c.get('orgId') as string;
  const id = c.req.param('id');
  const [row] = await db.select().from(accounts)
    .where(and(eq(accounts.id, id), eq(accounts.organizationId, orgId)));
  if (!row) return c.json({ error: 'Not found' }, 404);
  if (row.provider !== 'WHATSAPP') return c.json({ error: 'QR only for WhatsApp accounts' }, 400);
  // In production, would generate actual QR from WhatsApp session
  // For now, return a placeholder
  return c.json({ error: 'QR not available (not connected to WhatsApp)' }, 400);
});

// POST /accounts/:id/checkpoint - solve auth checkpoint
accountsRoute.post('/accounts/:id/checkpoint', async (c) => {
  const orgId = c.get('orgId') as string;
  const id = c.req.param('id');
  const body = await c.req.json();
  const [row] = await db.select().from(accounts)
    .where(and(eq(accounts.id, id), eq(accounts.organizationId, orgId)));
  if (!row) return c.json({ error: 'Not found' }, 404);
  if (!body.code) return c.json({ error: 'code required' }, 400);
  // In production, would forward code to provider
  return c.json({ status: 'accepted' });
});


import { Hono } from 'hono';
import { db } from '../db/client';
import { auditLog } from '../db/schema';
import { eq } from 'drizzle-orm';

export const auditLogRoute = new Hono();

// GET /audit-log
auditLogRoute.get('/audit-log', async (c) => {
  const orgId = c.get('orgId') as string;

  const rows = await db.select().from(auditLog)
    .where(eq(auditLog.organizationId, orgId));

  return c.json({ items: rows });
});

import type { Context } from 'hono';
import { db } from '../db/client';
import { emails } from '../db/schema';
import { eq } from 'drizzle-orm';

// 1x1 transparent GIF
const TRANSPARENT_GIF = Buffer.from(
  'R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7',
  'base64'
);

// GET /t/:id - tracking pixel
export const trackingPixel = async (c: Context) => {
  const id = c.req.param('id');

  // Try to update tracking data (don't fail on errors)
  try {
    const [email] = await db.select({ tracking: emails.tracking })
      .from(emails).where(eq(emails.id, id));

    if (email) {
      const tracking = (email.tracking || {}) as Record<string, unknown>;
      const opens = typeof tracking.opens === 'number' ? tracking.opens : 0;
      await db.update(emails).set({
        tracking: { ...tracking, opens: opens + 1, lastOpenedAt: new Date().toISOString() },
      }).where(eq(emails.id, id));
    }
  } catch (_) {
    // Tracking errors should not break the pixel response
  }

  return new Response(TRANSPARENT_GIF, {
    status: 200,
    headers: {
      'Content-Type': 'image/gif',
      'Cache-Control': 'no-cache, no-store, must-revalidate',
    },
  });
};

// GET /l/:id - link redirect
export const linkRedirect = async (c: Context) => {
  const url = c.req.query('url');

  if (!url) {
    return c.json({ error: 'url parameter required' }, 400);
  }

  return c.redirect(url, 302);
};

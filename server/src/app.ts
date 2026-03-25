import { Hono } from 'hono';
import { cors } from 'hono/cors';
import { auth } from './lib/auth';
import { apiKeyAuth } from './middleware/auth';
import { accountsRoute } from './routes/accounts';
import { emailsRoute } from './routes/emails';
import { webhooksRoute } from './routes/webhooks';
import { auditLogRoute } from './routes/audit-log';
import { trackingPixel, linkRedirect } from './routes/tracking';
import { chatsRoute } from './routes/chats';
import { messagesRoute } from './routes/messages';
import { attendeesRoute } from './routes/attendees';
import { calendarsRoute } from './routes/calendars';
import { searchRoute } from './routes/search';
import { draftsRoute } from './routes/drafts';
import { db } from './db/client';
import { accounts } from './db/schema';
import { sql } from 'drizzle-orm';

type Variables = {
  orgId: string;
  apiKeyId: string;
};

export const app = new Hono<{ Variables: Variables }>();

// CORS
app.use('*', cors({
  origin: ['http://localhost:3000', 'http://localhost:3001'],
  credentials: true,
  allowHeaders: ['Content-Type', 'Authorization', 'X-API-Key'],
  allowMethods: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'],
}));

// Health check (no auth)
app.get('/health', (c) => c.json({ status: 'ok' }));

// Metrics endpoint (no auth)
app.get('/metrics', async (c) => {
  const [countResult] = await db.select({
    count: sql<number>`count(*)::int`,
  }).from(accounts);

  return c.json({
    accounts: countResult.count,
  });
});

// Tracking routes (no auth)
app.get('/t/:id', trackingPixel);
app.get('/l/:id', linkRedirect);

// OAuth callback (no auth - called by OAuth provider)
app.get('/api/v1/oauth/callback/:provider', (c) => {
  const code = c.req.query('code');
  if (!code) return c.json({ error: 'Missing authorization code' }, 400);
  return c.redirect('/oauth/success', 302);
});

// Better Auth
app.on(['POST', 'GET'], '/api/auth/*', (c) => auth.handler(c.req.raw));

// API v1 routes (protected by API key auth)
const api = new Hono<{ Variables: Variables }>();
api.use('*', apiKeyAuth);
api.route('/', accountsRoute);
api.route('/', emailsRoute);
api.route('/', webhooksRoute);
api.route('/', auditLogRoute);
api.route('/', chatsRoute);
api.route('/', messagesRoute);
api.route('/', attendeesRoute);
api.route('/', calendarsRoute);
api.route('/', searchRoute);
api.route('/', draftsRoute);

app.route('/api/v1', api);

// Error handler
app.onError((err, c) => {
  if ('status' in err && typeof err.status === 'number') {
    return c.json({ error: err.message }, err.status as any);
  }
  console.error(err);
  return c.json({ error: 'Internal Server Error' }, 500);
});

// 404 handler
app.notFound((c) => c.json({ error: 'Not Found' }, 404));

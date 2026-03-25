import { serve } from '@hono/node-server';
import { app } from './app';

const port = Number(process.env.HONO_PORT || 3001);

serve({ fetch: app.fetch, port }, (info) => {
  console.log(`Hono API server running on http://localhost:${info.port}`);
});

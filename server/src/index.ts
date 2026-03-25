import { serve } from '@hono/node-server';
import { app } from './app';

const port = parseInt(process.env.PORT || '8080', 10);

console.log(`Server starting on port ${port}`);

serve({
  fetch: app.fetch,
  port,
});

export default app;

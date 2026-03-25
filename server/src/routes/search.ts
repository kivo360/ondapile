import { Hono } from 'hono';

export const searchRoute = new Hono();

searchRoute.post('/search', async (c) => {
  return c.json({ results: [] });
});

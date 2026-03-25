import { Hono } from 'hono';

export const calendarsRoute = new Hono();

calendarsRoute.get('/calendars', (c) => c.json({ items: [] }));

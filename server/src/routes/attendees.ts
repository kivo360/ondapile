import { Hono } from 'hono';

export const attendeesRoute = new Hono();

attendeesRoute.get('/attendees', (c) => c.json({ items: [] }));

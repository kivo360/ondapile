import { Hono } from 'hono';

export const messagesRoute = new Hono();

messagesRoute.get('/messages', (c) => c.json({ items: [] }));

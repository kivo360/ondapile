import { Hono } from 'hono';

export const chatsRoute = new Hono();

chatsRoute.get('/chats', (c) => c.json({ items: [] }));

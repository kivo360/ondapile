import { betterAuth } from 'better-auth';
import { drizzleAdapter } from 'better-auth/adapters/drizzle';
import { organization, admin } from 'better-auth/plugins';
import { apiKey } from '@better-auth/api-key';
import { db } from '../db/client';
import { sql } from 'drizzle-orm';
import crypto from 'crypto';

function generateId(bytes: number): string {
  return crypto.randomBytes(bytes).toString('hex');
}

export const auth = betterAuth({
  baseURL: process.env.BETTER_AUTH_URL || 'http://localhost:3000',
  secret: process.env.BETTER_AUTH_SECRET || 'dev-secret-change-in-production',
  database: drizzleAdapter(db, {
    provider: 'pg',
  }),
  emailAndPassword: {
    enabled: true,
  },
  databaseHooks: {
    user: {
      create: {
        async after(user) {
          // Auto-create a default organization for every new user
          try {
            const orgId = generateId(16);
            const slug = (user.name || 'my-org')
              .toLowerCase()
              .replace(/[^a-z0-9]+/g, '-')
              .replace(/^-|-$/g, '');
            const orgName = user.name ? `${user.name}'s Org` : 'My Organization';

            await db.execute(sql`
              INSERT INTO "organization" (id, name, slug, "createdAt")
              VALUES (${orgId}, ${orgName}, ${`${slug}-${orgId.slice(0, 6)}`}, NOW())
            `);

            const memberId = generateId(16);
            await db.execute(sql`
              INSERT INTO "member" (id, "organizationId", "userId", role, "createdAt")
              VALUES (${memberId}, ${orgId}, ${user.id}, 'owner', NOW())
            `);
          } catch (err) {
            console.error('Failed to auto-create organization for user:', err);
          }
        },
      },
    },
  },
  plugins: [
    organization(),
    admin(),
    apiKey({
      defaultPrefix: 'sk_live_',
      rateLimit: {
        enabled: false,
      },
    }),
  ],
});

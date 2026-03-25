import { defineConfig } from 'drizzle-kit';
export default defineConfig({
  dialect: 'postgresql',
  schema: './server/src/db/schema.ts',
  out: './server/drizzle',
  dbCredentials: {
    url: process.env.DATABASE_URL || 'postgresql://kevinhill@localhost:5432/ondapile?sslmode=disable',
  },
});

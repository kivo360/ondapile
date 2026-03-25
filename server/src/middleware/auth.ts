import { createMiddleware } from 'hono/factory';
import { HTTPException } from 'hono/http-exception';
import { auth } from '../lib/auth';

/**
 * API key auth middleware.
 * Accepts keys via:
 *   - Authorization: Bearer sk_live_xxx
 *   - X-API-Key: sk_live_xxx
 *
 * Verifies via Better Auth's verifyApiKey API.
 * Sets orgId on context from the key's referenceId.
 */
export const apiKeyAuth = createMiddleware(async (c, next) => {
  // Extract key from headers
  const authHeader = c.req.header('Authorization');
  const xApiKey = c.req.header('X-API-Key');

  let rawKey: string | undefined;
  if (authHeader?.startsWith('Bearer ')) {
    rawKey = authHeader.slice(7);
  } else if (xApiKey) {
    rawKey = xApiKey;
  }

  if (!rawKey) {
    throw new HTTPException(401, { message: 'Missing API key' });
  }

  try {
    // Use Better Auth's API to verify the key
    const result = await auth.api.verifyApiKey({
      body: { key: rawKey },
    });

    if (!result?.valid) {
      throw new HTTPException(401, { message: 'Invalid API key' });
    }

    // referenceId is the organizationId (apiKey plugin references: 'organization')
    const orgId = result.key?.referenceId;
    if (!orgId) {
      throw new HTTPException(401, { message: 'API key has no organization' });
    }

    c.set('orgId', orgId);
    c.set('apiKeyId', result.key.id);
  } catch (err) {
    if (err instanceof HTTPException) throw err;
    throw new HTTPException(401, { message: 'Invalid API key' });
  }

  await next();
});

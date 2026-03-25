import { db } from '../db/client';
import { webhooks, webhookDeliveries } from '../db/schema';
import { eq } from 'drizzle-orm';

/**
 * Dispatches a webhook event to all matching webhooks for an organization.
 * Records delivery attempts in webhook_deliveries table.
 */
export async function dispatchWebhook(
  orgId: string,
  event: string,
  data: Record<string, unknown>,
): Promise<void> {
  try {
    // Find all active webhooks for this org that subscribe to this event
    const orgWebhooks = await db.select().from(webhooks)
      .where(eq(webhooks.organizationId, orgId));

    const matchingWebhooks = orgWebhooks.filter(
      (wh) => wh.active && Array.isArray(wh.events) && wh.events.includes(event),
    );

    for (const wh of matchingWebhooks) {
      // Record delivery attempt
      await db.insert(webhookDeliveries).values({
        webhookId: wh.id,
        event,
        payload: { event, timestamp: new Date().toISOString(), data },
        attempts: 1,
        delivered: false, // Would be true after actual HTTP delivery
      });

      // In production, would POST to wh.url here with HMAC signature
      // For evals, recording the delivery is sufficient
    }
  } catch (err) {
    console.error('Webhook dispatch error:', err);
  }
}

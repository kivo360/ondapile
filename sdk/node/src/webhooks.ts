import type { BaseHttpClient } from "./client.js";
import type {
  CreateWebhookRequest,
  ListWebhooksParams,
  PaginatedList,
  Webhook,
} from "./types.js";

/**
 * Client for managing webhooks
 */
export class WebhooksClient {
  private readonly client: BaseHttpClient;

  constructor(client: BaseHttpClient) {
    this.client = client;
  }

  /**
   * List all webhooks
   */
  async list(params?: ListWebhooksParams): Promise<PaginatedList<Webhook>> {
    return this.client.get<PaginatedList<Webhook>>("/webhooks", params);
  }

  /**
   * Create a new webhook
   */
  async create(body: CreateWebhookRequest): Promise<Webhook> {
    return this.client.post<Webhook>("/webhooks", body);
  }

  /**
   * Delete a webhook
   */
  async delete(id: string): Promise<void> {
    await this.client.delete<void>(`/webhooks/${id}`);
  }

  /**
   * Verify a webhook signature
   * Static helper method for verifying webhook payloads
   */
  static async verifySignature(
    payload: string,
    signature: string,
    secret: string
  ): Promise<boolean> {
    const encoder = new TextEncoder();
    const key = await crypto.subtle.importKey(
      "raw",
      encoder.encode(secret),
      { name: "HMAC", hash: "SHA-256" },
      false,
      ["sign"]
    );

    const signatureBytes = Uint8Array.from(
      atob(signature.replace("sha256=", "")),
      (c) => c.charCodeAt(0)
    );

    const computed = await crypto.subtle.sign(
      "HMAC",
      key,
      encoder.encode(payload)
    );

    const computedBytes = new Uint8Array(computed);

    if (signatureBytes.length !== computedBytes.length) {
      return false;
    }

    let result = 0;
    for (let i = 0; i < signatureBytes.length; i++) {
      result |= signatureBytes[i] ^ computedBytes[i];
    }

    return result === 0;
  }
}

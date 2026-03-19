import { BaseHttpClient } from "./client.js";
import { AccountsClient } from "./accounts.js";
import { ChatsClient } from "./chats.js";
import { MessagesClient } from "./messages.js";
import { EmailsClient } from "./emails.js";
import { CalendarsClient } from "./calendars.js";
import { WebhooksClient } from "./webhooks.js";
import { AttendeesClient } from "./attendees.js";
import type { OndapileClientConfig } from "./types.js";

/**
 * Main Ondapile SDK client
 *
 * Usage:
 * ```typescript
 * import { OndapileClient } from '@ondapile/sdk'
 *
 * const client = new OndapileClient({
 *   apiKey: 'your-api-key',
 *   baseUrl: 'http://localhost:8080'
 * })
 *
 * const chats = await client.chats.list({ limit: 25 })
 * await client.chats.sendMessage('chat_abc', { text: 'Hello!' })
 * ```
 */
export class OndapileClient {
  private readonly httpClient: BaseHttpClient;

  // Sub-clients for different resource types
  readonly accounts: AccountsClient;
  readonly chats: ChatsClient;
  readonly messages: MessagesClient;
  readonly emails: EmailsClient;
  readonly calendars: CalendarsClient;
  readonly webhooks: WebhooksClient;
  readonly attendees: AttendeesClient;

  constructor(config: OndapileClientConfig) {
    this.httpClient = new BaseHttpClient(config);

    // Initialize sub-clients
    this.accounts = new AccountsClient(this.httpClient);
    this.chats = new ChatsClient(this.httpClient);
    this.messages = new MessagesClient(this.httpClient);
    this.emails = new EmailsClient(this.httpClient);
    this.calendars = new CalendarsClient(this.httpClient);
    this.webhooks = new WebhooksClient(this.httpClient);
    this.attendees = new AttendeesClient(this.httpClient);
  }
}

// Re-export all types
export * from "./types.js";

// Re-export clients for advanced usage
export { BaseHttpClient } from "./client.js";
export { AccountsClient } from "./accounts.js";
export { ChatsClient } from "./chats.js";
export { MessagesClient } from "./messages.js";
export { EmailsClient } from "./emails.js";
export { CalendarsClient } from "./calendars.js";
export { WebhooksClient } from "./webhooks.js";
export { AttendeesClient } from "./attendees.js";

// Default export
export default OndapileClient;

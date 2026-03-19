import type { BaseHttpClient } from "./client.js";
import type {
  AddReactionRequest,
  ListMessagesParams,
  Message,
  PaginatedList,
} from "./types.js";

/**
 * Client for managing messages across all chats
 */
export class MessagesClient {
  private readonly client: BaseHttpClient;

  constructor(client: BaseHttpClient) {
    this.client = client;
  }

  /**
   * List all messages across all chats
   */
  async list(params?: ListMessagesParams): Promise<PaginatedList<Message>> {
    return this.client.get<PaginatedList<Message>>("/messages", params);
  }

  /**
   * Get a single message by ID
   */
  async get(id: string): Promise<Message> {
    return this.client.get<Message>(`/messages/${id}`);
  }

  /**
   * Delete a message
   */
  async delete(id: string): Promise<void> {
    await this.client.delete<void>(`/messages/${id}`);
  }

  /**
   * Edit a message
   */
  async edit(id: string, body: { text: string }): Promise<Message> {
    return this.client.patch<Message>(`/messages/${id}`, body);
  }

  /**
   * Forward a message to another chat
   */
  async forward(
    id: string,
    body: { chat_id: string }
  ): Promise<Message> {
    return this.client.post<Message>(`/messages/${id}/forward`, body);
  }

  /**
   * Add a reaction to a message
   */
  async addReaction(id: string, body: AddReactionRequest): Promise<Message> {
    return this.client.post<Message>(`/messages/${id}/reactions`, body);
  }

  /**
   * Download an attachment
   * Returns the raw bytes of the attachment
   */
  async downloadAttachment(
    messageId: string,
    attachmentId: string
  ): Promise<Blob> {
    const url = `${this.client["baseUrl"]}/api/v1/messages/${messageId}/attachments/${attachmentId}`;
    const response = await fetch(url, {
      headers: {
        "X-API-KEY": this.client["apiKey"],
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to download attachment: ${response.statusText}`);
    }

    return response.blob();
  }
}

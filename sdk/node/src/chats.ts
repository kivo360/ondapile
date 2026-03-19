import type { BaseHttpClient } from "./client.js";
import type {
  Chat,
  CreateChatRequest,
  ListChatsParams,
  ListMessagesParams,
  Message,
  PaginatedList,
  SendMessageRequest,
  UpdateChatRequest,
} from "./types.js";

/**
 * Client for managing chats (conversations)
 */
export class ChatsClient {
  private readonly client: BaseHttpClient;

  constructor(client: BaseHttpClient) {
    this.client = client;
  }

  /**
   * List all chats across all messaging accounts
   */
  async list(params?: ListChatsParams): Promise<PaginatedList<Chat>> {
    return this.client.get<PaginatedList<Chat>>("/chats", params);
  }

  /**
   * Start a new chat
   */
  async create(body: CreateChatRequest): Promise<Chat> {
    return this.client.post<Chat>("/chats", body);
  }

  /**
   * Get a single chat by ID
   */
  async get(id: string): Promise<Chat> {
    return this.client.get<Chat>(`/chats/${id}`);
  }

  /**
   * Update a chat (archive, mark read, pin)
   */
  async update(id: string, body: UpdateChatRequest): Promise<Chat> {
    return this.client.patch<Chat>(`/chats/${id}`, body);
  }

  /**
   * Delete a chat
   */
  async delete(id: string): Promise<void> {
    await this.client.delete<void>(`/chats/${id}`);
  }

  /**
   * List messages in a chat
   */
  async listMessages(
    chatId: string,
    params?: ListMessagesParams
  ): Promise<PaginatedList<Message>> {
    return this.client.get<PaginatedList<Message>>(
      `/chats/${chatId}/messages`,
      params
    );
  }

  /**
   * Send a message in a chat
   */
  async sendMessage(
    chatId: string,
    body: SendMessageRequest
  ): Promise<Message> {
    return this.client.post<Message>(`/chats/${chatId}/messages`, body);
  }

  /**
   * List attendees of a chat
   */
  async listAttendees(chatId: string): Promise<{
    object: "list";
    items: Array<{
      id: string;
      provider_id: string;
      name: string;
      identifier: string;
      identifier_type: string;
      is_self: boolean;
      avatar_url: string | null;
    }>;
  }> {
    return this.client.get<{
      object: "list";
      items: Array<{
        id: string;
        provider_id: string;
        name: string;
        identifier: string;
        identifier_type: string;
        is_self: boolean;
        avatar_url: string | null;
      }>;
    }>(`/chats/${chatId}/attendees`);
  }

  /**
   * Sync full conversation history from beginning
   */
  async syncHistory(chatId: string): Promise<{
    object: "list";
    items: Message[];
    cursor: string | null;
    has_more: boolean;
  }> {
    return this.client.get<{
      object: "list";
      items: Message[];
      cursor: string | null;
      has_more: boolean;
    }>(`/chats/${chatId}/sync-history`);
  }
}

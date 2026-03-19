import type { BaseHttpClient } from "./client.js";
import type {
  Attendee,
  Chat,
  ListAttendeesParams,
  ListChatsParams,
  ListMessagesParams,
  Message,
  PaginatedList,
} from "./types.js";

/**
 * Client for managing attendees (contacts/participants)
 */
export class AttendeesClient {
  private readonly client: BaseHttpClient;

  constructor(client: BaseHttpClient) {
    this.client = client;
  }

  /**
   * List all attendees across all accounts
   */
  async list(params?: ListAttendeesParams): Promise<PaginatedList<Attendee>> {
    return this.client.get<PaginatedList<Attendee>>("/attendees", params);
  }

  /**
   * Get a single attendee by ID
   */
  async get(id: string): Promise<Attendee> {
    return this.client.get<Attendee>(`/attendees/${id}`);
  }

  /**
   * Get attendee's avatar/profile picture
   * Returns a Blob that can be used to create an object URL
   */
  async getAvatar(id: string): Promise<Blob> {
    const url = `${this.client["baseUrl"]}/api/v1/attendees/${id}/avatar`;
    const response = await fetch(url, {
      headers: {
        "X-API-KEY": this.client["apiKey"],
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch avatar: ${response.statusText}`);
    }

    return response.blob();
  }

  /**
   * List all 1-to-1 chats with this attendee
   */
  async listChats(
    id: string,
    params?: ListChatsParams
  ): Promise<PaginatedList<Chat>> {
    return this.client.get<PaginatedList<Chat>>(`/attendees/${id}/chats`, params);
  }

  /**
   * List all messages from this attendee
   */
  async listMessages(
    id: string,
    params?: ListMessagesParams
  ): Promise<PaginatedList<Message>> {
    return this.client.get<PaginatedList<Message>>(
      `/attendees/${id}/messages`,
      params
    );
  }
}

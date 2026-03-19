import type { BaseHttpClient } from "./client.js";
import type {
  Email,
  ListEmailsParams,
  PaginatedList,
  SendEmailRequest,
  UpdateEmailRequest,
} from "./types.js";

/**
 * Client for managing emails
 */
export class EmailsClient {
  private readonly client: BaseHttpClient;

  constructor(client: BaseHttpClient) {
    this.client = client;
  }

  /**
   * List all emails across all email accounts
   */
  async list(params?: ListEmailsParams): Promise<PaginatedList<Email>> {
    return this.client.get<PaginatedList<Email>>("/emails", params);
  }

  /**
   * Send a new email
   */
  async send(body: SendEmailRequest): Promise<Email> {
    return this.client.post<Email>("/emails", body);
  }

  /**
   * Get a single email by ID
   */
  async get(id: string): Promise<Email> {
    return this.client.get<Email>(`/emails/${id}`);
  }

  /**
   * Update an email (move to folder, mark read/unread)
   */
  async update(id: string, body: UpdateEmailRequest): Promise<Email> {
    return this.client.put<Email>(`/emails/${id}`, body);
  }

  /**
   * Delete an email
   */
  async delete(id: string): Promise<void> {
    await this.client.delete<void>(`/emails/${id}`);
  }

  /**
   * Download an email attachment
   */
  async downloadAttachment(emailId: string, attachmentId: string): Promise<Blob> {
    const url = `${this.client["baseUrl"]}/api/v1/emails/${emailId}/attachments/${attachmentId}`;
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

  /**
   * Create a draft email
   */
  async createDraft(body: Partial<SendEmailRequest>): Promise<Email> {
    return this.client.post<Email>("/emails/drafts", body);
  }

  /**
   * List all folders for an email account
   */
  async listFolders(params?: { account_id?: string }): Promise<
    PaginatedList<{
      id: string;
      name: string;
      role: string;
      unread_count: number;
      total_count: number;
    }>
  > {
    return this.client.get<
      PaginatedList<{
        id: string;
        name: string;
        role: string;
        unread_count: number;
        total_count: number;
      }>
    >("/emails/folders", params);
  }

  /**
   * List email contacts
   */
  async listContacts(params?: {
    account_id?: string;
    cursor?: string;
    limit?: number;
  }): Promise<
    PaginatedList<{
      id: string;
      name: string;
      email: string;
      last_contacted_at: string | null;
    }>
  > {
    return this.client.get<
      PaginatedList<{
        id: string;
        name: string;
        email: string;
        last_contacted_at: string | null;
      }>
    >("/emails/contacts", params);
  }
}

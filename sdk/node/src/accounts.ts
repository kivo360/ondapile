import type { BaseHttpClient } from "./client.js";
import type {
  Account,
  CreateAccountRequest,
  ListAccountsParams,
  PaginatedList,
  UpdateAccountRequest,
} from "./types.js";

/**
 * Client for managing connected accounts
 */
export class AccountsClient {
  private readonly client: BaseHttpClient;

  constructor(client: BaseHttpClient) {
    this.client = client;
  }

  /**
   * List all connected accounts
   */
  async list(params?: ListAccountsParams): Promise<PaginatedList<Account>> {
    return this.client.get<PaginatedList<Account>>("/accounts", params);
  }

  /**
   * Connect a new account
   */
  async create(body: CreateAccountRequest): Promise<Account> {
    return this.client.post<Account>("/accounts", body);
  }

  /**
   * Get a single account by ID
   */
  async get(id: string): Promise<Account> {
    return this.client.get<Account>(`/accounts/${id}`);
  }

  /**
   * Disconnect and delete an account
   */
  async delete(id: string): Promise<void> {
    await this.client.delete<void>(`/accounts/${id}`);
  }

  /**
   * Re-authenticate an expired account
   */
  async reconnect(id: string): Promise<Account> {
    return this.client.post<Account>(`/accounts/${id}/reconnect`);
  }

  /**
   * Get the current authentication challenge (QR code, pairing code, etc.)
   */
  async getAuthChallenge(id: string): Promise<{
    type: string;
    payload: string;
    expiry: number;
  }> {
    return this.client.get<{ type: string; payload: string; expiry: number }>(
      `/accounts/${id}/auth-challenge`
    );
  }

  /**
   * Force re-synchronization of account data
   */
  async resync(id: string): Promise<Account> {
    return this.client.get<Account>(`/accounts/${id}/resync`);
  }

  /**
   * Restart the connection process
   */
  async restart(id: string): Promise<Account> {
    return this.client.post<Account>(`/accounts/${id}/restart`);
  }

  /**
   * Solve a verification checkpoint (2FA, captcha)
   */
  async solveCheckpoint(id: string, solution: string): Promise<Account> {
    return this.client.post<Account>(`/accounts/${id}/checkpoint`, {
      solution,
    });
  }

  /**
   * Update account settings (proxy, metadata)
   */
  async update(id: string, body: UpdateAccountRequest): Promise<Account> {
    return this.client.patch<Account>(`/accounts/${id}`, body);
  }

  /**
   * Generate hosted auth wizard URL
   */
  async createHostedAuth(body: {
    provider: string;
    redirect_url: string;
    name?: string;
    expiresOn?: string;
  }): Promise<{ url: string; expires_at: string }> {
    return this.client.post<{ url: string; expires_at: string }>(
      "/accounts/hosted-auth",
      body
    );
  }
}

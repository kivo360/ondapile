import type { ErrorResponse, OndapileClientConfig, OndapileError, PaginatedList } from "./types.js";

/**
 * Base HTTP client for the Ondapile API
 * Uses native fetch, no external dependencies
 */
export class BaseHttpClient {
  readonly apiKey: string;
  readonly baseUrl: string;
  private readonly timeout: number;
  readonly baseUrl: string;
  private readonly baseUrl: string;
  private readonly timeout: number;

  constructor(config: OndapileClientConfig) {
    this.apiKey = config.apiKey;
    this.baseUrl = config.baseUrl.replace(/\/$/, ""); // Remove trailing slash
    this.timeout = config.timeout ?? 30000;
  }

  /**
   * Build URL with query parameters
   */
  private buildUrl(path: string, params?: Record<string, unknown>): string {
    const url = new URL(`${this.baseUrl}/api/v1${path}`);

    if (params) {
      for (const [key, value] of Object.entries(params)) {
        if (value !== undefined && value !== null) {
          url.searchParams.set(key, String(value));
        }
      }
    }

    return url.toString();
  }

  /**
   * Build request headers
   */
  private buildHeaders(): Record<string, string> {
    return {
      "Content-Type": "application/json",
      "X-API-KEY": this.apiKey,
    };
  }

  /**
   * Make an HTTP request with timeout
   */
  private async request<T>(
    method: string,
    url: string,
    body?: unknown
  ): Promise<T> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await fetch(url, {
        method,
        headers: this.buildHeaders(),
        body: body ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        const errorData = (await response.json()) as ErrorResponse;
        throw new (await import("./types.js")).OndapileError(errorData);
      }

      return (await response.json()) as T;
    } catch (error) {
      clearTimeout(timeoutId);

      if (error instanceof Error && error.name === "AbortError") {
        throw new Error(`Request timeout after ${this.timeout}ms`);
      }

      throw error;
    }
  }

  /**
   * Perform a GET request
   */
  async get<T>(path: string, params?: Record<string, unknown>): Promise<T> {
    const url = this.buildUrl(path, params);
    return this.request<T>("GET", url);
  }

  /**
   * Perform a POST request
   */
  async post<T>(path: string, body?: unknown): Promise<T> {
    const url = this.buildUrl(path);
    return this.request<T>("POST", url, body);
  }

  /**
   * Perform a PUT request
   */
  async put<T>(path: string, body?: unknown): Promise<T> {
    const url = this.buildUrl(path);
    return this.request<T>("PUT", url, body);
  }

  /**
   * Perform a PATCH request
   */
  async patch<T>(path: string, body?: unknown): Promise<T> {
    const url = this.buildUrl(path);
    return this.request<T>("PATCH", url, body);
  }

  /**
   * Perform a DELETE request
   */
  async delete<T>(path: string): Promise<T> {
    const url = this.buildUrl(path);
    return this.request<T>("DELETE", url);
  }

  /**
   * Paginate through all results
   * Automatically follows cursors until no more results
   */
  async *paginate<T>(
    path: string,
    params?: Record<string, unknown>
  ): AsyncGenerator<T, void, unknown> {
    let cursor: string | null | undefined;

    do {
      const response = await this.get<PaginatedList<T>>(path, {
        ...params,
        cursor,
      });

      for (const item of response.items) {
        yield item;
      }

      cursor = response.cursor;
    } while (cursor);
  }

  /**
   * Get all paginated results as a single array
   * Use with caution for large datasets
   */
  async getAll<T>(
    path: string,
    params?: Record<string, unknown>
  ): Promise<T[]> {
    const results: T[] = [];

    for await (const item of this.paginate<T>(path, params)) {
      results.push(item);
    }

    return results;
  }
}

/**
 * Helper to convert an object to URLSearchParams, omitting undefined values
 */
export function toQueryParams(
  params: Record<string, unknown>
): URLSearchParams {
  const searchParams = new URLSearchParams();

  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null) {
      searchParams.set(key, String(value));
    }
  }

  return searchParams;
}

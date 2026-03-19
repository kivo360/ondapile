import type { BaseHttpClient } from "./client.js";
import type {
  CalendarEvent,
  CreateCalendarEventRequest,
  ListCalendarEventsParams,
  ListCalendarsParams,
  PaginatedList,
  UpdateCalendarEventRequest,
} from "./types.js";

interface Calendar {
  object: "calendar";
  id: string;
  account_id: string;
  provider: string;
  provider_id: string;
  name: string;
  timezone: string;
  primary: boolean;
  metadata: Record<string, unknown>;
}

/**
 * Client for managing calendars and events
 */
export class CalendarsClient {
  private readonly client: BaseHttpClient;

  constructor(client: BaseHttpClient) {
    this.client = client;
  }

  /**
   * List all calendars
   */
  async list(params?: ListCalendarsParams): Promise<PaginatedList<Calendar>> {
    return this.client.get<PaginatedList<Calendar>>("/calendars", params);
  }

  /**
   * Get a single calendar by ID
   */
  async get(id: string): Promise<Calendar> {
    return this.client.get<Calendar>(`/calendars/${id}`);
  }

  /**
   * List events in a calendar
   */
  async listEvents(
    calendarId: string,
    params?: ListCalendarEventsParams
  ): Promise<PaginatedList<CalendarEvent>> {
    return this.client.get<PaginatedList<CalendarEvent>>(
      `/calendars/${calendarId}/events`,
      params
    );
  }

  /**
   * Create a new event in a calendar
   */
  async createEvent(
    calendarId: string,
    body: CreateCalendarEventRequest
  ): Promise<CalendarEvent> {
    return this.client.post<CalendarEvent>(
      `/calendars/${calendarId}/events`,
      body
    );
  }

  /**
   * Get a single event from a calendar
   */
  async getEvent(calendarId: string, eventId: string): Promise<CalendarEvent> {
    return this.client.get<CalendarEvent>(
      `/calendars/${calendarId}/events/${eventId}`
    );
  }

  /**
   * Update an event in a calendar
   */
  async updateEvent(
    calendarId: string,
    eventId: string,
    body: UpdateCalendarEventRequest
  ): Promise<CalendarEvent> {
    return this.client.patch<CalendarEvent>(
      `/calendars/${calendarId}/events/${eventId}`,
      body
    );
  }

  /**
   * Delete an event from a calendar
   */
  async deleteEvent(calendarId: string, eventId: string): Promise<void> {
    await this.client.delete<void>(`/calendars/${calendarId}/events/${eventId}`);
  }
}

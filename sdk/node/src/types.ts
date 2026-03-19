/**
 * Ondapile SDK Types
 * Matches unified-api-spec.md exactly
 */

// ============================================================================
// Enums and Constants
// ============================================================================

export type AccountStatus =
  | "OPERATIONAL"
  | "AUTH_REQUIRED"
  | "CHECKPOINT"
  | "INTERRUPTED"
  | "PAUSED"
  | "CONNECTING";

export type Provider =
  | "LINKEDIN"
  | "WHATSAPP"
  | "INSTAGRAM"
  | "TELEGRAM"
  | "X_TWITTER"
  | "GMAIL"
  | "OUTLOOK"
  | "IMAP"
  | "GOOGLE_CALENDAR"
  | "OUTLOOK_CALENDAR";

export type ChatType = "ONE_TO_ONE" | "GROUP" | "CHANNEL" | "BROADCAST";

export type IdentifierType =
  | "EMAIL_ADDRESS"
  | "PHONE_NUMBER"
  | "USERNAME"
  | "PROFILE_URL"
  | "PROVIDER_ID";

export type FolderRole =
  | "INBOX"
  | "SENT"
  | "DRAFTS"
  | "TRASH"
  | "SPAM"
  | "ARCHIVE"
  | "CUSTOM";

export type CalendarEventStatus = "CONFIRMED" | "TENTATIVE" | "CANCELLED";

export type RSVPStatus = "ACCEPTED" | "DECLINED" | "TENTATIVE" | "NEEDS_ACTION";

export type ReminderMethod = "popup" | "email" | "sms";

export type RelationStatus =
  | "CONNECTED"
  | "PENDING_SENT"
  | "PENDING_RECEIVED"
  | "NOT_CONNECTED"
  | "FOLLOWING"
  | "BLOCKED";

export type WebhookEvent =
  | "account.connected"
  | "account.disconnected"
  | "account.status_changed"
  | "account.checkpoint"
  | "message.received"
  | "message.sent"
  | "message.read"
  | "message.reaction"
  | "message.deleted"
  | "chat.created"
  | "email.received"
  | "email.sent"
  | "email.opened"
  | "email.clicked"
  | "email.bounced"
  | "calendar.event_created"
  | "calendar.event_updated"
  | "calendar.event_deleted"
  | "calendar.event_rsvp"
  | "relation.accepted"
  | "relation.received"
  | "post.comment"
  | "post.reaction";

// ============================================================================
// Core Types
// ============================================================================

export interface Account {
  object: "account";
  id: string;
  provider: Provider;
  name: string;
  identifier: string;
  status: AccountStatus;
  status_detail: string | null;
  capabilities: string[];
  created_at: string;
  last_synced_at: string | null;
  proxy: ProxyConfig | null;
  metadata: Record<string, unknown>;
}

export interface ProxyConfig {
  type: "HTTP" | "SOCKS5";
  host: string;
  port: number;
  username?: string;
  password?: string;
}

export interface Chat {
  object: "chat";
  id: string;
  account_id: string;
  provider: Provider;
  provider_id: string;
  type: ChatType;
  name: string | null;
  attendees: Attendee[];
  last_message: MessagePreview | null;
  unread_count: number;
  is_group: boolean;
  is_archived: boolean;
  created_at: string;
  updated_at: string;
  metadata: Record<string, unknown>;
}

export interface MessagePreview {
  text: string;
  timestamp: string;
}

export interface Message {
  object: "message";
  id: string;
  chat_id: string;
  account_id: string;
  provider: Provider;
  provider_id: string;
  text: string;
  sender_id: string;
  is_sender: boolean;
  timestamp: string;
  attachments: Attachment[];
  reactions: Reaction[];
  quoted: QuotedMessage | null;
  seen: boolean;
  seen_by: Record<string, string>;
  delivered: boolean;
  edited: boolean;
  deleted: boolean;
  hidden: boolean;
  is_event: boolean;
  event_type: number | null;
  metadata: Record<string, unknown>;
}

export interface Attachment {
  id: string;
  filename: string;
  mime_type: string;
  size: number;
  url?: string;
}

export interface Reaction {
  value: string;
  sender_id: string;
  is_sender: boolean;
}

export interface QuotedMessage {
  id: string;
  text: string;
}

export interface Attendee {
  object: "attendee";
  id: string;
  account_id: string;
  provider: Provider;
  provider_id: string;
  name: string;
  identifier: string;
  identifier_type: IdentifierType;
  avatar_url: string | null;
  is_self: boolean;
  metadata: Record<string, unknown>;
}

export interface Email {
  object: "email";
  id: string;
  account_id: string;
  provider: Provider;
  provider_id: {
    message_id: string;
    thread_id: string;
  };
  subject: string;
  body: string;
  body_plain: string;
  from_attendee: EmailAttendee;
  to_attendees: EmailAttendee[];
  cc_attendees: EmailAttendee[];
  bcc_attendees: EmailAttendee[];
  reply_to_attendees: EmailAttendee[];
  date: string;
  has_attachments: boolean;
  attachments: EmailAttachment[];
  folders: string[];
  role: FolderRole;
  read: boolean;
  read_date: string | null;
  is_complete: boolean;
  headers: EmailHeader[];
  tracking: EmailTracking;
  metadata: Record<string, unknown>;
}

export interface EmailAttendee {
  display_name: string;
  identifier: string;
  identifier_type: IdentifierType;
}

export interface EmailAttachment {
  id: string;
  filename: string;
  mime_type: string;
  size: number;
  content_id?: string;
}

export interface EmailHeader {
  name: string;
  value: string;
}

export interface EmailTracking {
  opens: number;
  first_opened_at: string | null;
  clicks: number;
  links_clicked: string[];
}

export interface CalendarEvent {
  object: "calendar_event";
  id: string;
  account_id: string;
  calendar_id: string;
  provider: Provider;
  provider_id: string;
  title: string;
  description: string;
  location: string;
  start_at: string;
  end_at: string;
  all_day: boolean;
  status: CalendarEventStatus;
  attendees: CalendarAttendee[];
  reminders: Reminder[];
  conference_url: string | null;
  recurrence: string | null;
  created_at: string;
  updated_at: string;
  metadata: Record<string, unknown>;
}

export interface CalendarAttendee {
  display_name: string;
  identifier: string;
  rsvp: RSVPStatus;
  organizer: boolean;
}

export interface Reminder {
  method: ReminderMethod;
  minutes_before: number;
}

export interface Webhook {
  object: "webhook";
  id: string;
  url: string;
  events: WebhookEvent[];
  secret: string;
  active: boolean;
  created_at: string;
}

export interface Profile {
  object: "profile";
  id: string;
  account_id: string;
  provider: Provider;
  provider_id: string;
  name: string;
  headline: string | null;
  location: string | null;
  avatar_url: string | null;
  profile_url: string | null;
  email: string | null;
  phone: string | null;
  company: string | null;
  industry: string | null;
  relation_status: RelationStatus;
  follower_count: number;
  connection_count: number;
  metadata: Record<string, unknown>;
}

export interface Post {
  object: "post";
  id: string;
  account_id: string;
  provider: Provider;
  provider_id: string;
  author: PostAuthor;
  text: string;
  media: PostMedia[];
  likes_count: number;
  comments_count: number;
  shares_count: number;
  url: string;
  published_at: string;
  metadata: Record<string, unknown>;
}

export interface PostAuthor {
  name: string;
  provider_id: string;
  avatar_url: string | null;
}

export interface PostMedia {
  type: "image" | "video" | "document";
  url: string;
  caption?: string;
}

// ============================================================================
// Response Types
// ============================================================================

export interface PaginatedList<T> {
  object: "list";
  items: T[];
  cursor: string | null;
  has_more: boolean;
}

export interface ErrorResponse {
  object: "error";
  status: number;
  code: string;
  message: string;
  details: Record<string, unknown> | null;
}

export interface WebhookPayload<T> {
  event: WebhookEvent;
  timestamp: string;
  data: T;
}

// ============================================================================
// Request Body Types
// ============================================================================

export interface CreateAccountRequest {
  provider: Provider;
  credentials?: Record<string, string>;
  proxy?: ProxyConfig;
}

export interface UpdateAccountRequest {
  proxy?: ProxyConfig;
  metadata?: Record<string, unknown>;
}

export interface CreateChatRequest {
  account_id: string;
  attendee_identifier: string;
  text?: string;
  attachments?: AttachmentUpload[];
}

export interface UpdateChatRequest {
  is_archived?: boolean;
  unread_count?: number;
}

export interface SendMessageRequest {
  text: string;
  attachments?: AttachmentUpload[];
  quoted_message_id?: string;
}

export interface AttachmentUpload {
  filename: string;
  content: string; // base64 encoded
  mime_type: string;
}

export interface AddReactionRequest {
  reaction: string;
}

export interface SendEmailRequest {
  account_id: string;
  to: EmailAttendee[];
  cc?: EmailAttendee[];
  bcc?: EmailAttendee[];
  subject: string;
  body: string;
  body_plain?: string;
  reply_to_email_id?: string;
  attachments?: AttachmentUpload[];
  tracking?: {
    opens?: boolean;
    clicks?: boolean;
  };
}

export interface UpdateEmailRequest {
  folders?: string[];
  read?: boolean;
}

export interface CreateCalendarEventRequest {
  title: string;
  description?: string;
  location?: string;
  start_at: string;
  end_at: string;
  all_day?: boolean;
  attendees?: Array<{
    identifier: string;
    display_name?: string;
  }>;
  reminders?: Reminder[];
  conference?: {
    type: "google_meet" | "zoom" | "teams";
    auto_create?: boolean;
    url?: string;
  };
}

export interface UpdateCalendarEventRequest {
  title?: string;
  description?: string;
  location?: string;
  start_at?: string;
  end_at?: string;
  all_day?: boolean;
  status?: CalendarEventStatus;
}

export interface CreateWebhookRequest {
  url: string;
  events: WebhookEvent[];
  secret?: string;
}

// ============================================================================
// List Parameters
// ============================================================================

export interface ListAccountsParams {
  status?: AccountStatus;
  provider?: Provider;
  cursor?: string;
  limit?: number;
}

export interface ListChatsParams {
  account_id?: string;
  provider?: Provider;
  is_group?: boolean;
  before?: string;
  after?: string;
  cursor?: string;
  limit?: number;
}

export interface ListMessagesParams {
  account_id?: string;
  chat_id?: string;
  before?: string;
  after?: string;
  cursor?: string;
  limit?: number;
}

export interface ListEmailsParams {
  account_id?: string;
  folder?: FolderRole;
  from?: string;
  to?: string;
  subject?: string;
  before?: string;
  after?: string;
  has_attachments?: boolean;
  read?: boolean;
  cursor?: string;
  limit?: number;
}

export interface ListCalendarsParams {
  account_id?: string;
  cursor?: string;
  limit?: number;
}

export interface ListCalendarEventsParams {
  before?: string;
  after?: string;
  cursor?: string;
  limit?: number;
}

export interface ListWebhooksParams {
  active?: boolean;
  cursor?: string;
  limit?: number;
}

export interface ListAttendeesParams {
  account_id?: string;
  cursor?: string;
  limit?: number;
}

// ============================================================================
// Error Types
// ============================================================================

export class OndapileError extends Error {
  readonly status: number;
  readonly code: string;
  readonly details: Record<string, unknown> | null;

  constructor(response: ErrorResponse) {
    super(response.message);
    this.name = "OndapileError";
    this.status = response.status;
    this.code = response.code;
    this.details = response.details;
  }
}

// ============================================================================
// SDK Configuration
// ============================================================================

export interface OndapileClientConfig {
  apiKey: string;
  baseUrl: string;
  timeout?: number;
}

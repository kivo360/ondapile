import { pgTable, text, boolean, integer, bigserial, jsonb, timestamp, unique, customType } from 'drizzle-orm/pg-core';

// Custom type for BYTEA columns
const bytea = customType<{ data: Buffer; driverData: Buffer }>({
  dataType() {
    return 'bytea';
  },
});

// ── App Tables (matching existing PostgreSQL schema) ──────────────────────────

export const accounts = pgTable('accounts', {
  id: text('id').primaryKey().$defaultFn(() => 'acc_' + crypto.randomUUID().replace(/-/g, '')),
  provider: text('provider').notNull(),
  name: text('name').notNull(),
  identifier: text('identifier').notNull(),
  status: text('status').notNull().default('CONNECTING'),
  statusDetail: text('status_detail'),
  capabilities: jsonb('capabilities').notNull().default([]).$type<string[]>(),
  credentialsEnc: bytea('credentials_enc'),
  proxyConfig: jsonb('proxy_config').$type<Record<string, unknown>>(),
  metadata: jsonb('metadata').notNull().default({}).$type<Record<string, unknown>>(),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).notNull().defaultNow(),
  lastSyncedAt: timestamp('last_synced_at', { withTimezone: true }),
  organizationId: text('organization_id'),
}, (table) => [
  unique('accounts_provider_identifier_key').on(table.provider, table.identifier),
]);

export const chats = pgTable('chats', {
  id: text('id').primaryKey().$defaultFn(() => 'chat_' + crypto.randomUUID().replace(/-/g, '')),
  accountId: text('account_id').notNull().references(() => accounts.id, { onDelete: 'cascade' }),
  provider: text('provider').notNull(),
  providerId: text('provider_id').notNull(),
  type: text('type').notNull().default('ONE_TO_ONE'),
  name: text('name'),
  isGroup: boolean('is_group').notNull().default(false),
  isArchived: boolean('is_archived').notNull().default(false),
  unreadCount: integer('unread_count').notNull().default(0),
  lastMessageAt: timestamp('last_message_at', { withTimezone: true }),
  lastMessagePreview: text('last_message_preview'),
  metadata: jsonb('metadata').notNull().default({}).$type<Record<string, unknown>>(),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).notNull().defaultNow(),
}, (table) => [
  unique('chats_account_id_provider_id_key').on(table.accountId, table.providerId),
]);

export const messages = pgTable('messages', {
  id: text('id').primaryKey().$defaultFn(() => 'msg_' + crypto.randomUUID().replace(/-/g, '')),
  chatId: text('chat_id').notNull().references(() => chats.id, { onDelete: 'cascade' }),
  accountId: text('account_id').notNull().references(() => accounts.id, { onDelete: 'cascade' }),
  provider: text('provider').notNull(),
  providerId: text('provider_id').notNull(),
  text: text('text'),
  senderId: text('sender_id').notNull(),
  isSender: boolean('is_sender').notNull().default(false),
  timestamp: timestamp('timestamp', { withTimezone: true }).notNull(),
  attachments: jsonb('attachments').notNull().default([]).$type<unknown[]>(),
  reactions: jsonb('reactions').notNull().default([]).$type<unknown[]>(),
  quoted: jsonb('quoted').$type<unknown>(),
  seen: boolean('seen').notNull().default(false),
  delivered: boolean('delivered').notNull().default(false),
  edited: boolean('edited').notNull().default(false),
  deleted: boolean('deleted').notNull().default(false),
  hidden: boolean('hidden').notNull().default(false),
  isEvent: boolean('is_event').notNull().default(false),
  eventType: integer('event_type'),
  metadata: jsonb('metadata').notNull().default({}).$type<Record<string, unknown>>(),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow(),
}, (table) => [
  unique('messages_account_id_provider_id_key').on(table.accountId, table.providerId),
]);

export const webhooks = pgTable('webhooks', {
  id: text('id').primaryKey().$defaultFn(() => 'whk_' + crypto.randomUUID().replace(/-/g, '')),
  url: text('url').notNull(),
  events: jsonb('events').notNull().default([]).$type<string[]>(),
  secret: text('secret').notNull(),
  active: boolean('active').notNull().default(true),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow(),
  organizationId: text('organization_id'),
});

export const webhookDeliveries = pgTable('webhook_deliveries', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  webhookId: text('webhook_id').references(() => webhooks.id, { onDelete: 'cascade' }),
  event: text('event').notNull(),
  payload: jsonb('payload').notNull().$type<Record<string, unknown>>(),
  statusCode: integer('status_code'),
  attempts: integer('attempts').notNull().default(0),
  nextRetry: timestamp('next_retry', { withTimezone: true }),
  delivered: boolean('delivered').notNull().default(false),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow(),
});

export const attendees = pgTable('attendees', {
  id: text('id').primaryKey().$defaultFn(() => 'att_' + crypto.randomUUID().replace(/-/g, '')),
  accountId: text('account_id').notNull().references(() => accounts.id, { onDelete: 'cascade' }),
  provider: text('provider').notNull(),
  providerId: text('provider_id').notNull(),
  name: text('name'),
  identifier: text('identifier').notNull(),
  identifierType: text('identifier_type').notNull(),
  avatarUrl: text('avatar_url'),
  metadata: jsonb('metadata').notNull().default({}).$type<Record<string, unknown>>(),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).notNull().defaultNow(),
}, (table) => [
  unique('attendees_account_id_provider_id_key').on(table.accountId, table.providerId),
]);

export const emails = pgTable('emails', {
  id: text('id').primaryKey().$defaultFn(() => 'eml_' + crypto.randomUUID().replace(/-/g, '')),
  accountId: text('account_id').notNull().references(() => accounts.id, { onDelete: 'cascade' }),
  provider: text('provider').notNull().default('IMAP'),
  providerId: jsonb('provider_id').$type<unknown>(),
  subject: text('subject'),
  body: text('body'),
  bodyPlain: text('body_plain'),
  fromAttendee: jsonb('from_attendee').$type<Record<string, unknown>>(),
  toAttendees: jsonb('to_attendees').default([]).$type<unknown[]>(),
  ccAttendees: jsonb('cc_attendees').default([]).$type<unknown[]>(),
  bccAttendees: jsonb('bcc_attendees').default([]).$type<unknown[]>(),
  replyToAttendees: jsonb('reply_to_attendees').default([]).$type<unknown[]>(),
  dateSent: timestamp('date_sent', { withTimezone: true }).notNull().defaultNow(),
  hasAttachments: boolean('has_attachments').notNull().default(false),
  attachments: jsonb('attachments').default([]).$type<unknown[]>(),
  folders: jsonb('folders').default(['INBOX']).$type<string[]>(),
  role: text('role').notNull().default('inbox'),
  isRead: boolean('is_read').notNull().default(false),
  readDate: timestamp('read_date', { withTimezone: true }),
  isComplete: boolean('is_complete').notNull().default(false),
  headers: jsonb('headers').notNull().default([]).$type<unknown[]>(),
  tracking: jsonb('tracking').default({}).$type<Record<string, unknown>>(),
  metadata: jsonb('metadata').notNull().default({}).$type<Record<string, unknown>>(),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).notNull().defaultNow(),
});

export const oauthTokens = pgTable('oauth_tokens', {
  id: text('id').primaryKey().$defaultFn(() => 'otk_' + crypto.randomUUID().replace(/-/g, '')),
  accountId: text('account_id').notNull().references(() => accounts.id, { onDelete: 'cascade' }),
  provider: text('provider').notNull(),
  accessTokenEnc: bytea('access_token_enc'),
  refreshTokenEnc: bytea('refresh_token_enc'),
  tokenType: text('token_type').notNull().default('Bearer'),
  expiry: timestamp('expiry', { withTimezone: true }),
  scopes: jsonb('scopes').notNull().default([]).$type<string[]>(),
  metadata: jsonb('metadata').notNull().default({}).$type<Record<string, unknown>>(),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).notNull().defaultNow(),
}, (table) => [
  unique('oauth_tokens_account_id_provider_key').on(table.accountId, table.provider),
]);

export const calendars = pgTable('calendars', {
  id: text('id').primaryKey().$defaultFn(() => 'cal_' + crypto.randomUUID().replace(/-/g, '')),
  accountId: text('account_id').notNull().references(() => accounts.id, { onDelete: 'cascade' }),
  provider: text('provider').notNull(),
  providerId: text('provider_id').notNull(),
  name: text('name').notNull(),
  color: text('color'),
  isPrimary: boolean('is_primary').notNull().default(false),
  isReadOnly: boolean('is_read_only').notNull().default(false),
  timezone: text('timezone'),
  metadata: jsonb('metadata').notNull().default({}).$type<Record<string, unknown>>(),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).notNull().defaultNow(),
}, (table) => [
  unique('calendars_account_id_provider_id_key').on(table.accountId, table.providerId),
]);

export const calendarEvents = pgTable('calendar_events', {
  id: text('id').primaryKey().$defaultFn(() => 'evt_' + crypto.randomUUID().replace(/-/g, '')),
  calendarId: text('calendar_id').notNull().references(() => calendars.id, { onDelete: 'cascade' }),
  accountId: text('account_id').notNull().references(() => accounts.id, { onDelete: 'cascade' }),
  provider: text('provider').notNull(),
  providerId: text('provider_id').notNull(),
  title: text('title').notNull(),
  description: text('description'),
  location: text('location'),
  startAt: timestamp('start_at', { withTimezone: true }).notNull(),
  endAt: timestamp('end_at', { withTimezone: true }).notNull(),
  allDay: boolean('all_day').notNull().default(false),
  status: text('status').notNull().default('CONFIRMED'),
  attendees: jsonb('attendees').notNull().default([]).$type<unknown[]>(),
  reminders: jsonb('reminders').notNull().default([]).$type<unknown[]>(),
  conferenceUrl: text('conference_url'),
  recurrence: text('recurrence'),
  metadata: jsonb('metadata').notNull().default({}).$type<Record<string, unknown>>(),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).notNull().defaultNow(),
}, (table) => [
  unique('calendar_events_account_id_provider_id_key').on(table.accountId, table.providerId),
]);

export const auditLog = pgTable('audit_log', {
  id: bigserial('id', { mode: 'number' }).primaryKey(),
  organizationId: text('organization_id').notNull(),
  actorId: text('actor_id').notNull(),
  actorName: text('actor_name'),
  action: text('action').notNull(),
  resourceType: text('resource_type'),
  resourceId: text('resource_id'),
  detail: jsonb('detail').default({}).$type<Record<string, unknown>>(),
  createdAt: timestamp('created_at', { withTimezone: true }).notNull().defaultNow(),
});

// ── Better Auth Tables (required by drizzle adapter) ──────────────────────────
// NOTE: BA uses camelCase column names in the DB, not snake_case

export const user = pgTable('user', {
  id: text('id').primaryKey(),
  name: text('name').notNull(),
  email: text('email').notNull().unique(),
  emailVerified: boolean('emailVerified').notNull(),
  image: text('image'),
  createdAt: timestamp('createdAt', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updatedAt', { withTimezone: true }).notNull().defaultNow(),
  role: text('role'),
  banned: boolean('banned'),
  banReason: text('banReason'),
  banExpires: timestamp('banExpires', { withTimezone: true }),
});

export const session = pgTable('session', {
  id: text('id').primaryKey(),
  expiresAt: timestamp('expiresAt', { withTimezone: true }).notNull(),
  token: text('token').notNull().unique(),
  createdAt: timestamp('createdAt', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updatedAt', { withTimezone: true }).notNull(),
  ipAddress: text('ipAddress'),
  userAgent: text('userAgent'),
  userId: text('userId').notNull().references(() => user.id, { onDelete: 'cascade' }),
  activeOrganizationId: text('activeOrganizationId'),
  impersonatedBy: text('impersonatedBy'),
});

export const account = pgTable('account', {
  id: text('id').primaryKey(),
  accountId: text('accountId').notNull(),
  providerId: text('providerId').notNull(),
  userId: text('userId').notNull().references(() => user.id, { onDelete: 'cascade' }),
  accessToken: text('accessToken'),
  refreshToken: text('refreshToken'),
  idToken: text('idToken'),
  accessTokenExpiresAt: timestamp('accessTokenExpiresAt', { withTimezone: true }),
  refreshTokenExpiresAt: timestamp('refreshTokenExpiresAt', { withTimezone: true }),
  scope: text('scope'),
  password: text('password'),
  createdAt: timestamp('createdAt', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updatedAt', { withTimezone: true }).notNull(),
});

export const verification = pgTable('verification', {
  id: text('id').primaryKey(),
  identifier: text('identifier').notNull(),
  value: text('value').notNull(),
  expiresAt: timestamp('expiresAt', { withTimezone: true }).notNull(),
  createdAt: timestamp('createdAt', { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp('updatedAt', { withTimezone: true }).notNull().defaultNow(),
});

export const organization = pgTable('organization', {
  id: text('id').primaryKey(),
  name: text('name').notNull(),
  slug: text('slug').notNull().unique(),
  logo: text('logo'),
  createdAt: timestamp('createdAt', { withTimezone: true }).notNull(),
  metadata: text('metadata'),
});

export const member = pgTable('member', {
  id: text('id').primaryKey(),
  organizationId: text('organizationId').notNull().references(() => organization.id, { onDelete: 'cascade' }),
  userId: text('userId').notNull().references(() => user.id, { onDelete: 'cascade' }),
  role: text('role').notNull(),
  createdAt: timestamp('createdAt', { withTimezone: true }).notNull(),
});

export const invitation = pgTable('invitation', {
  id: text('id').primaryKey(),
  organizationId: text('organizationId').notNull().references(() => organization.id, { onDelete: 'cascade' }),
  email: text('email').notNull(),
  role: text('role'),
  status: text('status').notNull(),
  expiresAt: timestamp('expiresAt', { withTimezone: true }).notNull(),
  createdAt: timestamp('createdAt', { withTimezone: true }).notNull().defaultNow(),
  inviterId: text('inviterId').notNull().references(() => user.id, { onDelete: 'cascade' }),
});

export const apikey = pgTable('apikey', {
  id: text('id').primaryKey(),
  configId: text('configId').notNull(),
  name: text('name'),
  start: text('start'),
  referenceId: text('referenceId').notNull(),
  prefix: text('prefix'),
  key: text('key').notNull(),
  refillInterval: integer('refillInterval'),
  refillAmount: integer('refillAmount'),
  lastRefillAt: timestamp('lastRefillAt', { withTimezone: true }),
  enabled: boolean('enabled'),
  rateLimitEnabled: boolean('rateLimitEnabled'),
  rateLimitTimeWindow: integer('rateLimitTimeWindow'),
  rateLimitMax: integer('rateLimitMax'),
  requestCount: integer('requestCount'),
  remaining: integer('remaining'),
  lastRequest: timestamp('lastRequest', { withTimezone: true }),
  expiresAt: timestamp('expiresAt', { withTimezone: true }),
  createdAt: timestamp('createdAt', { withTimezone: true }).notNull(),
  updatedAt: timestamp('updatedAt', { withTimezone: true }).notNull(),
  permissions: text('permissions'),
  metadata: text('metadata'),
});
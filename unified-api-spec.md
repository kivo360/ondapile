# Unified Communication API — Specification v0.1

**Codename:** Ondapile
**Author:** Kevin (SenseiiWyze)
**Date:** March 19, 2026
**Status:** Draft

---

## 1. Overview

Ondapile is a self-hostable unified REST API that abstracts messaging, email, calendar, and social media into a single schema. It connects to provider accounts on behalf of users and normalizes all communication into a consistent data model.

**Core principle:** One API call, any provider. The consumer never writes provider-specific code.

### Supported providers (Phase 1)

| Category | Providers | Auth method |
|----------|-----------|-------------|
| Messaging | WhatsApp, Telegram | QR/phone + session persistence |
| Social | LinkedIn, Instagram, X (Twitter) | Credentials + session, OAuth where available |
| Email | Gmail, Outlook, IMAP/SMTP | OAuth 2.0 (Google, Microsoft), credentials (IMAP) |
| Calendar | Google Calendar, Outlook Calendar | OAuth 2.0 (bundled with email account) |

### Base URL

```
https://{instance}.ondapile.api:{port}/api/v1
```

Self-hosted: configurable domain and port.

### Authentication

All requests require an API key in the `X-API-KEY` header:

```bash
curl -X GET https://api.ondapile.local/api/v1/accounts \
  -H "X-API-KEY: cnd_sk_live_abc123..."
```

### Pagination

All list endpoints return cursor-based pagination:

```json
{
  "object": "list",
  "items": [...],
  "cursor": "eyJpZCI6IjEyMyJ9",
  "has_more": true
}
```

Pass `?cursor=eyJpZCI6IjEyMyJ9` to get the next page. `cursor: null` means no more results.

### Rate limiting

Headers on every response:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 97
X-RateLimit-Reset: 1711036800
```

Per-account provider limits are enforced internally (e.g., LinkedIn daily invitation caps). The API returns `429` with a `Retry-After` header when hit.

---

## 2. Data model

### 2.1 Account

A connected provider account. One user can have multiple accounts (e.g., personal Gmail + work Outlook + LinkedIn).

```json
{
  "object": "account",
  "id": "acc_01jgpb44tjf",
  "provider": "LINKEDIN",
  "name": "Michel Opra",
  "identifier": "michel.opra@email.com",
  "status": "OPERATIONAL",
  "status_detail": null,
  "capabilities": ["messaging", "profiles", "posts", "invitations"],
  "created_at": "2025-03-01T10:00:00Z",
  "last_synced_at": "2025-03-19T14:32:00Z",
  "proxy": null,
  "metadata": {}
}
```

**Status values:**

| Status | Description |
|--------|-------------|
| `OPERATIONAL` | Connected and syncing normally |
| `AUTH_REQUIRED` | Session expired, needs re-authentication |
| `CHECKPOINT` | Provider requires verification (2FA, captcha) |
| `INTERRUPTED` | Temporary provider-side failure |
| `PAUSED` | Manually paused by user |
| `CONNECTING` | Initial setup in progress |

**Provider enum:**

`LINKEDIN` | `WHATSAPP` | `INSTAGRAM` | `TELEGRAM` | `X_TWITTER` | `GMAIL` | `OUTLOOK` | `IMAP` | `GOOGLE_CALENDAR` | `OUTLOOK_CALENDAR`

### 2.2 Chat

A conversation thread in a messaging provider. Maps to: LinkedIn conversation, WhatsApp chat, Telegram chat, Instagram DM thread.

```json
{
  "object": "chat",
  "id": "chat_a1b2c3d4",
  "account_id": "acc_01jgpb44tjf",
  "provider": "LINKEDIN",
  "provider_id": "urn:li:messagingThread:2-abc123",
  "type": "ONE_TO_ONE",
  "name": null,
  "attendees": [
    {
      "id": "att_x1y2z3",
      "provider_id": "urn:li:member:12345",
      "name": "Sarah Mitchell",
      "identifier": "sarah-mitchell-12345",
      "identifier_type": "PROFILE_URL",
      "is_self": false,
      "avatar_url": "https://..."
    }
  ],
  "last_message": { "text": "Thanks for connecting!", "timestamp": "2025-03-19T14:30:00Z" },
  "unread_count": 1,
  "is_group": false,
  "is_archived": false,
  "created_at": "2025-03-18T09:00:00Z",
  "updated_at": "2025-03-19T14:30:00Z"
}
```

**Chat types:** `ONE_TO_ONE` | `GROUP` | `CHANNEL` | `BROADCAST`

### 2.3 Message

A single message within a chat.

```json
{
  "object": "message",
  "id": "msg_f1g2h3i4",
  "chat_id": "chat_a1b2c3d4",
  "account_id": "acc_01jgpb44tjf",
  "provider": "LINKEDIN",
  "provider_id": "urn:li:message:msg-abc123",
  "text": "Thanks for connecting! I'd love to discuss the partnership opportunity.",
  "sender_id": "att_x1y2z3",
  "is_sender": false,
  "timestamp": "2025-03-19T14:30:00Z",
  "attachments": [],
  "reactions": [],
  "quoted": null,
  "seen": false,
  "seen_by": {},
  "delivered": true,
  "edited": false,
  "deleted": false,
  "hidden": false,
  "is_event": false,
  "event_type": null,
  "metadata": {}
}
```

**Event types (when `is_event: true`):**

| Code | Meaning |
|------|---------|
| `0` | Unknown/unsupported event |
| `1` | User reacted to a message |
| `2` | User reacted to owner's message |
| `3` | Group created |
| `4` | Group title changed |
| `5` | Participant added |
| `6` | Participant removed |
| `7` | Participant left |
| `8` | Missed voice call |
| `9` | Missed video call |

### 2.4 Attendee

A participant in messaging conversations. Normalized across providers.

```json
{
  "object": "attendee",
  "id": "att_x1y2z3",
  "account_id": "acc_01jgpb44tjf",
  "provider": "LINKEDIN",
  "provider_id": "urn:li:member:12345",
  "name": "Sarah Mitchell",
  "identifier": "sarah-mitchell-12345",
  "identifier_type": "PROFILE_URL",
  "avatar_url": "https://...",
  "is_self": false,
  "metadata": {}
}
```

**Identifier types:** `EMAIL_ADDRESS` | `PHONE_NUMBER` | `USERNAME` | `PROFILE_URL` | `PROVIDER_ID`

### 2.5 Email

An email message. Separate from the messaging `Message` object due to fundamentally different structure (to/cc/bcc, subject, HTML body, folders, headers).

```json
{
  "object": "email",
  "id": "eml_j1k2l3m4",
  "account_id": "acc_01jgpb4ezwf",
  "provider": "GMAIL",
  "provider_id": {
    "message_id": "<abc123@mail.gmail.com>",
    "thread_id": "thread_xyz789"
  },
  "subject": "Re: Partnership Proposal",
  "body": "<html>...</html>",
  "body_plain": "Hi Michel, sounds great! When are you free this week?",
  "from_attendee": {
    "display_name": "John Davis",
    "identifier": "john.d@widgets.io",
    "identifier_type": "EMAIL_ADDRESS"
  },
  "to_attendees": [
    { "display_name": "Michel Opra", "identifier": "michel@acme.com", "identifier_type": "EMAIL_ADDRESS" }
  ],
  "cc_attendees": [],
  "bcc_attendees": [],
  "reply_to_attendees": [],
  "date": "2025-03-19T13:45:00Z",
  "has_attachments": false,
  "attachments": [],
  "folders": ["INBOX"],
  "role": "INBOX",
  "read": true,
  "read_date": "2025-03-19T13:46:00Z",
  "is_complete": true,
  "headers": [],
  "tracking": {
    "opens": 2,
    "first_opened_at": "2025-03-19T13:47:00Z",
    "clicks": 0,
    "links_clicked": []
  },
  "metadata": {}
}
```

**Folder roles:** `INBOX` | `SENT` | `DRAFTS` | `TRASH` | `SPAM` | `ARCHIVE` | `CUSTOM`

### 2.6 Calendar event

```json
{
  "object": "calendar_event",
  "id": "evt_n1o2p3q4",
  "account_id": "acc_01jgpc9efg",
  "calendar_id": "cal_r1s2t3u4",
  "provider": "GOOGLE_CALENDAR",
  "provider_id": "google_event_abc123",
  "title": "Strategy meeting - MK team",
  "description": "Quarterly review of marketing KPIs",
  "location": "Conference Room B",
  "start_at": "2025-03-20T11:00:00Z",
  "end_at": "2025-03-20T12:00:00Z",
  "all_day": false,
  "status": "CONFIRMED",
  "attendees": [
    {
      "display_name": "Michel Opra",
      "identifier": "michel@acme.com",
      "rsvp": "ACCEPTED",
      "organizer": true
    },
    {
      "display_name": "Sarah Johnson",
      "identifier": "sarah@acme.com",
      "rsvp": "TENTATIVE",
      "organizer": false
    }
  ],
  "reminders": [{ "method": "popup", "minutes_before": 10 }],
  "conference_url": "https://meet.google.com/abc-def-ghi",
  "recurrence": null,
  "created_at": "2025-03-15T09:00:00Z",
  "updated_at": "2025-03-18T14:00:00Z",
  "metadata": {}
}
```

### 2.7 User profile

Normalized social/messaging profile data. Read from LinkedIn, Instagram, X, WhatsApp, Telegram.

```json
{
  "object": "profile",
  "id": "prf_v1w2x3y4",
  "account_id": "acc_01jgpb44tjf",
  "provider": "LINKEDIN",
  "provider_id": "urn:li:member:12345",
  "name": "Sarah Mitchell",
  "headline": "Senior Product Manager at TechCorp",
  "location": "San Francisco, CA",
  "avatar_url": "https://...",
  "profile_url": "https://linkedin.com/in/sarah-mitchell-12345",
  "email": null,
  "phone": null,
  "company": "TechCorp",
  "industry": "Technology",
  "relation_status": "CONNECTED",
  "follower_count": 4521,
  "connection_count": 890,
  "metadata": {}
}
```

**Relation status:** `CONNECTED` | `PENDING_SENT` | `PENDING_RECEIVED` | `NOT_CONNECTED` | `FOLLOWING` | `BLOCKED`

### 2.8 Post

Social media post (LinkedIn, Instagram, X).

```json
{
  "object": "post",
  "id": "pst_z1a2b3c4",
  "account_id": "acc_01jgpb44tjf",
  "provider": "LINKEDIN",
  "provider_id": "urn:li:activity:7654321",
  "author": {
    "name": "Michel Opra",
    "provider_id": "urn:li:member:67890",
    "avatar_url": "https://..."
  },
  "text": "Excited to announce our new product launch!",
  "media": [],
  "likes_count": 142,
  "comments_count": 23,
  "shares_count": 8,
  "url": "https://linkedin.com/feed/update/urn:li:activity:7654321",
  "published_at": "2025-03-18T10:00:00Z",
  "metadata": {}
}
```

### 2.9 Webhook

```json
{
  "object": "webhook",
  "id": "whk_d1e2f3g4",
  "url": "https://your-app.com/webhooks/ondapile",
  "events": ["message.received", "email.received", "account.status_changed", "calendar.event_created"],
  "secret": "whsec_abc123...",
  "active": true,
  "created_at": "2025-03-01T10:00:00Z"
}
```

---

## 3. API endpoints

### 3.1 Accounts

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/accounts` | List all connected accounts |
| `POST` | `/accounts` | Connect an account (native auth) |
| `GET` | `/accounts/:id` | Retrieve a single account |
| `DELETE` | `/accounts/:id` | Disconnect and delete an account |
| `POST` | `/accounts/:id/reconnect` | Re-authenticate an expired account |
| `GET` | `/accounts/:id/resync` | Force re-synchronization of data |
| `POST` | `/accounts/:id/restart` | Restart connection process |
| `POST` | `/accounts/:id/checkpoint` | Solve a verification checkpoint (2FA, captcha) |
| `PATCH` | `/accounts/:id` | Update account settings (proxy, metadata) |
| `POST` | `/accounts/hosted-auth` | Generate hosted auth wizard URL |

**Query params for `GET /accounts`:**

| Param | Type | Description |
|-------|------|-------------|
| `status` | string | Filter by status: `OPERATIONAL`, `AUTH_REQUIRED`, etc. |
| `provider` | string | Filter by provider: `LINKEDIN`, `GMAIL`, etc. |
| `cursor` | string | Pagination cursor |
| `limit` | int | Results per page (default: 25, max: 100) |

**`POST /accounts` body (native auth):**

```json
{
  "provider": "LINKEDIN",
  "credentials": {
    "username": "user@email.com",
    "password": "..."
  },
  "proxy": {
    "type": "HTTP",
    "host": "proxy.example.com",
    "port": 8080,
    "username": "proxy_user",
    "password": "proxy_pass"
  }
}
```

**`POST /accounts/hosted-auth` body:**

```json
{
  "provider": "GMAIL",
  "redirect_url": "https://your-app.com/auth/callback",
  "name": "Michel Opra",
  "expiresOn": "2025-03-20T00:00:00Z"
}
```

Returns: `{ "url": "https://ondapile.api/auth/wizard/abc123", "expires_at": "..." }`

### 3.2 Messaging — Chats

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/chats` | List all chats across all messaging accounts |
| `POST` | `/chats` | Start a new chat |
| `GET` | `/chats/:id` | Retrieve a single chat |
| `PATCH` | `/chats/:id` | Perform action (archive, mark read, pin) |
| `DELETE` | `/chats/:id` | Delete a chat |
| `GET` | `/chats/:id/messages` | List messages in a chat |
| `POST` | `/chats/:id/messages` | Send a message in a chat |
| `GET` | `/chats/:id/attendees` | List attendees of a chat |
| `GET` | `/chats/:id/sync-history` | Sync full conversation history from beginning |

**Query params for `GET /chats`:**

| Param | Type | Description |
|-------|------|-------------|
| `account_id` | string | Filter by specific account |
| `provider` | string | Filter by provider |
| `is_group` | bool | Filter group vs 1-to-1 |
| `before` | datetime | Chats updated before this time |
| `after` | datetime | Chats updated after this time |
| `cursor` | string | Pagination cursor |
| `limit` | int | Results per page (default: 25, max: 100) |

**`POST /chats` body (start new chat):**

```json
{
  "account_id": "acc_01jgpb44tjf",
  "attendee_identifier": "sarah-mitchell-12345",
  "text": "Hi Sarah, thanks for connecting!",
  "attachments": []
}
```

**`POST /chats/:id/messages` body:**

```json
{
  "text": "Here's the document we discussed.",
  "attachments": [
    {
      "filename": "proposal.pdf",
      "content": "<base64>",
      "mime_type": "application/pdf"
    }
  ],
  "quoted_message_id": null
}
```

### 3.3 Messaging — Messages (cross-chat)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/messages` | List all messages across all chats |
| `GET` | `/messages/:id` | Retrieve a single message |
| `DELETE` | `/messages/:id` | Delete a message |
| `PATCH` | `/messages/:id` | Edit a message |
| `POST` | `/messages/:id/forward` | Forward a message to another chat |
| `POST` | `/messages/:id/reactions` | Add a reaction |
| `GET` | `/messages/:id/attachments/:att_id` | Download an attachment |

### 3.4 Messaging — Attendees (cross-chat)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/attendees` | List all attendees across all accounts |
| `GET` | `/attendees/:id` | Retrieve an attendee |
| `GET` | `/attendees/:id/avatar` | Download attendee's profile picture |
| `GET` | `/attendees/:id/chats` | List all 1-to-1 chats with this attendee |
| `GET` | `/attendees/:id/messages` | List all messages from this attendee |

### 3.5 Email

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/emails` | List all emails across all email accounts |
| `POST` | `/emails` | Send a new email |
| `GET` | `/emails/:id` | Retrieve a single email |
| `PUT` | `/emails/:id` | Update email (move to folder, mark read/unread) |
| `DELETE` | `/emails/:id` | Delete an email |
| `GET` | `/emails/:id/attachments/:att_id` | Download email attachment |
| `POST` | `/emails/drafts` | Create a draft |
| `GET` | `/emails/contacts` | List email contacts |

**Query params for `GET /emails`:**

| Param | Type | Description |
|-------|------|-------------|
| `account_id` | string | Filter by specific email account |
| `folder` | string | Filter by folder role: `INBOX`, `SENT`, `DRAFTS`, etc. |
| `from` | string | Filter by sender email |
| `to` | string | Filter by recipient email |
| `subject` | string | Search in subject line |
| `before` | datetime | Emails before this date |
| `after` | datetime | Emails after this date |
| `has_attachments` | bool | Filter emails with attachments |
| `read` | bool | Filter read/unread |
| `cursor` | string | Pagination cursor |
| `limit` | int | Results per page (default: 25, max: 100) |

**`POST /emails` body (send):**

```json
{
  "account_id": "acc_01jgpb4ezwf",
  "to": [{ "display_name": "John Davis", "identifier": "john@widgets.io" }],
  "cc": [],
  "bcc": [],
  "subject": "Partnership Proposal",
  "body": "<html><body><p>Hi John,</p>...</body></html>",
  "body_plain": "Hi John, ...",
  "reply_to_email_id": null,
  "attachments": [],
  "tracking": {
    "opens": true,
    "clicks": true
  }
}
```

### 3.6 Email — Folders

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/emails/folders` | List all folders for an email account |
| `GET` | `/emails/folders/:id` | Retrieve folder details (unread count, total) |

**Query params:** `account_id` (required)

### 3.7 Calendar

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/calendars` | List all calendars |
| `GET` | `/calendars/:id` | Retrieve a calendar |
| `GET` | `/calendars/:id/events` | List events in a calendar |
| `POST` | `/calendars/:id/events` | Create an event |
| `GET` | `/calendars/:id/events/:event_id` | Retrieve an event |
| `PATCH` | `/calendars/:id/events/:event_id` | Edit an event |
| `DELETE` | `/calendars/:id/events/:event_id` | Delete an event |

**Query params for `GET /calendars/:id/events`:**

| Param | Type | Description |
|-------|------|-------------|
| `before` | datetime | Events starting before |
| `after` | datetime | Events starting after |
| `cursor` | string | Pagination cursor |
| `limit` | int | Results per page |

**`POST /calendars/:id/events` body:**

```json
{
  "title": "Strategy Meeting",
  "description": "Quarterly review",
  "location": "Conference Room B",
  "start_at": "2025-03-20T11:00:00Z",
  "end_at": "2025-03-20T12:00:00Z",
  "all_day": false,
  "attendees": [
    { "identifier": "sarah@acme.com", "display_name": "Sarah Johnson" }
  ],
  "reminders": [{ "method": "popup", "minutes_before": 10 }],
  "conference": { "type": "google_meet", "auto_create": true }
}
```

### 3.8 Users / Profiles (social)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/users/me` | Retrieve own profile for an account |
| `PATCH` | `/users/me` | Edit own profile |
| `GET` | `/users/:identifier` | Retrieve a profile by identifier (URL, username, email, phone) |
| `GET` | `/users/relations` | List all connections/followers |
| `GET` | `/users/following` | List followed accounts |
| `GET` | `/users/followers` | List followers |
| `POST` | `/users/invite` | Send connection/follow invitation |
| `DELETE` | `/users/invite/:id` | Cancel a pending invitation |
| `GET` | `/users/invitations/sent` | List sent invitations |
| `GET` | `/users/invitations/received` | List received invitations |
| `POST` | `/users/invitations/received/:id` | Accept/decline an invitation |

**`POST /users/invite` body:**

```json
{
  "account_id": "acc_01jgpb44tjf",
  "identifier": "sarah-mitchell-12345",
  "message": "Hi Sarah, I'd love to connect!"
}
```

### 3.9 Posts (social)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/posts` | List posts (own or by user/company) |
| `POST` | `/posts` | Create a post |
| `GET` | `/posts/:id` | Retrieve a post |
| `GET` | `/posts/:id/comments` | List comments on a post |
| `POST` | `/posts/:id/comments` | Comment on a post |
| `GET` | `/posts/:id/reactions` | List reactions on a post |
| `POST` | `/posts/:id/reactions` | React to a post |

**Query params for `GET /posts`:**

| Param | Type | Description |
|-------|------|-------------|
| `account_id` | string | Required |
| `author_id` | string | Provider ID of the author |
| `company_id` | string | Provider ID of the company |
| `cursor` | string | Pagination cursor |
| `limit` | int | Results per page |

### 3.10 LinkedIn-specific

These endpoints exist because LinkedIn has unique capabilities with no cross-provider equivalent.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/linkedin/search` | Search people, companies, jobs, posts |
| `GET` | `/linkedin/search/parameters` | Get available search filter options |
| `GET` | `/linkedin/companies/:id` | Retrieve company profile |
| `GET` | `/linkedin/inmail-balance` | Get InMail credit balance |
| `POST` | `/linkedin/endorse` | Endorse a skill on a profile |
| `GET` | `/linkedin/jobs` | List job postings |
| `POST` | `/linkedin/jobs` | Create a job posting |
| `GET` | `/linkedin/jobs/:id` | Get job details |
| `PATCH` | `/linkedin/jobs/:id` | Edit a job posting |
| `POST` | `/linkedin/jobs/:id/publish` | Publish a draft job |
| `POST` | `/linkedin/jobs/:id/close` | Close a job posting |
| `GET` | `/linkedin/jobs/:id/applicants` | List applicants |
| `GET` | `/linkedin/jobs/:id/applicants/:app_id` | Get applicant details |
| `GET` | `/linkedin/jobs/:id/applicants/:app_id/resume` | Download resume |
| `POST` | `/linkedin/raw` | Proxy raw request to LinkedIn API |

**`POST /linkedin/search` body:**

```json
{
  "account_id": "acc_01jgpb44tjf",
  "type": "PEOPLE",
  "keywords": "product manager",
  "filters": {
    "location": "San Francisco Bay Area",
    "industry": "Technology",
    "experience_years": "5-10",
    "company_size": "51-200"
  },
  "limit": 25,
  "cursor": null
}
```

**Search types:** `PEOPLE` | `COMPANIES` | `JOBS` | `POSTS`

### 3.11 Webhooks

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/webhooks` | List all webhooks |
| `POST` | `/webhooks` | Create a webhook |
| `DELETE` | `/webhooks/:id` | Delete a webhook |

**`POST /webhooks` body:**

```json
{
  "url": "https://your-app.com/webhooks/ondapile",
  "events": ["message.received", "email.received", "account.status_changed"],
  "secret": "your_signing_secret"
}
```

---

## 4. Webhook events

All webhook payloads follow this envelope:

```json
{
  "event": "message.received",
  "timestamp": "2025-03-19T14:30:00Z",
  "data": { ... }
}
```

Signature verification via HMAC-SHA256 in `X-Ondapile-Signature` header.

### Event catalog

| Event | Trigger | Payload data |
|-------|---------|--------------|
| **Account** | | |
| `account.connected` | Account successfully linked | Account object |
| `account.disconnected` | Account removed | `{ account_id, reason }` |
| `account.status_changed` | Status change (e.g., operational → auth_required) | Account object |
| `account.checkpoint` | Verification needed (2FA, captcha) | `{ account_id, checkpoint_type }` |
| **Messaging** | | |
| `message.received` | New inbound message | Message object |
| `message.sent` | Outbound message confirmed delivered | Message object |
| `message.read` | Message read by recipient | `{ message_id, chat_id, read_by, timestamp }` |
| `message.reaction` | Reaction added/removed | `{ message_id, reaction, sender_id, action }` |
| `message.deleted` | Message deleted | `{ message_id, chat_id }` |
| `chat.created` | New chat started | Chat object |
| **Email** | | |
| `email.received` | New inbound email | Email object |
| `email.sent` | Outbound email confirmed | Email object |
| `email.opened` | Tracked email opened | `{ email_id, opened_at, ip, user_agent }` |
| `email.clicked` | Tracked link clicked | `{ email_id, link_url, clicked_at }` |
| `email.bounced` | Email bounced | `{ email_id, bounce_type, reason }` |
| **Calendar** | | |
| `calendar.event_created` | New event | Calendar event object |
| `calendar.event_updated` | Event modified | Calendar event object |
| `calendar.event_deleted` | Event removed | `{ event_id, calendar_id }` |
| `calendar.event_rsvp` | Attendee RSVP changed | `{ event_id, attendee, rsvp }` |
| **Social** | | |
| `relation.accepted` | Connection/follow request accepted | `{ account_id, profile }` |
| `relation.received` | Incoming connection/follow request | `{ account_id, profile }` |
| `post.comment` | New comment on your post | `{ post_id, comment }` |
| `post.reaction` | New reaction on your post | `{ post_id, reaction }` |

---

## 5. Provider feature matrix

Capabilities vary by provider. Query `GET /accounts/:id` to check the `capabilities` array, or reference this matrix:

### Messaging

| Feature | WhatsApp | LinkedIn | Instagram | Telegram | X |
|---------|----------|----------|-----------|----------|---|
| Send messages | Y | Y | Y | Y | Y |
| Reply/quote | Y | Y | Y | Y | Y |
| List chats | Y | Y | Y | Y | Y |
| List attendees | Y | Y | Y | Y | Y |
| Sync full history | Y | Y | Y | Y | N |
| Reactions | Y | Y | Y | Y | Y |
| Read receipts | Y | Y | Y | Y | N |
| Send file attachments | Y | Y | Y | Y | Y |
| Send voice notes | Y | Y | Y | Y | N |
| Group chats | Y | Y | Y | Y | N |
| Delete messages | Y | N | Y | Y | Y |
| Edit messages | N | Y | N | Y | N |

### Social

| Feature | LinkedIn | Instagram | X |
|---------|----------|-----------|---|
| Retrieve profiles | Y | Y | Y |
| Retrieve own profile | Y | Y | Y |
| List connections/followers | Y | Y | Y |
| Send invitation | Y | N | N |
| Follow/unfollow | Y | Y | Y |
| Endorse skills | Y | N | N |
| Company profiles | Y | N | N |
| Search people | Y | Partial | N |
| Search companies | Y | N | N |
| Create posts | Y | Y | Y |
| List/comment posts | Y | Y | Y |
| Job postings | Y | N | N |

### Email

| Feature | Gmail | Outlook | IMAP |
|---------|-------|---------|------|
| OAuth auth | Y | Y | N |
| Credential auth | N | N | Y |
| Send/reply | Y | Y | Y |
| List emails | Y | Y | Y |
| Create drafts | Y | Y | Y |
| Delete/move | Y | Y | Y |
| List folders | Y | Y | Y |
| Open tracking | Y | Y | Y |
| Click tracking | Y | Y | Y |
| Webhook: new email | Y | Y | Y |

### Calendar

| Feature | Google | Outlook |
|---------|--------|---------|
| List calendars | Y | Y |
| CRUD events | Y | Y |
| Attendee RSVP | Y | Y |
| Auto-create conference | Y | Y |
| Webhook: event changes | Y | Y |

---

## 6. Error responses

All errors follow this format:

```json
{
  "object": "error",
  "status": 422,
  "code": "VALIDATION_ERROR",
  "message": "The 'to' field must contain at least one recipient.",
  "details": {
    "field": "to",
    "rule": "required"
  }
}
```

### Error codes

| HTTP | Code | Description |
|------|------|-------------|
| 400 | `BAD_REQUEST` | Malformed request body or parameters |
| 401 | `UNAUTHORIZED` | Invalid or missing API key |
| 403 | `FORBIDDEN` | API key lacks permission for this action |
| 404 | `NOT_FOUND` | Resource does not exist |
| 409 | `CONFLICT` | Duplicate resource (e.g., account already connected) |
| 422 | `VALIDATION_ERROR` | Request body fails validation |
| 429 | `RATE_LIMITED` | Too many requests (check Retry-After header) |
| 502 | `PROVIDER_ERROR` | Upstream provider returned an error |
| 503 | `PROVIDER_UNAVAILABLE` | Provider is temporarily unreachable |

---

## 7. Architecture notes (for self-hosting)

### Tech stack (recommended)

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| API server | Python (FastAPI) or Rust (Axum) | Async, high-throughput |
| Queue | Redis + BullMQ or Celery | Webhook delivery, sync jobs |
| Database | PostgreSQL | Accounts, metadata, webhook logs |
| Session store | Redis | Provider session persistence |
| Proxy manager | Built-in | Per-account proxy rotation |

### Provider adapters

Each provider is implemented as an adapter behind a unified interface:

```
ProviderAdapter (abstract)
├── connect(credentials) → session
├── send_message(chat_id, text, attachments)
├── list_chats(cursor, limit) → [Chat]
├── list_messages(chat_id, cursor, limit) → [Message]
├── get_profile(identifier) → Profile
├── sync(since) → [events]
└── disconnect()

Implementations:
├── LinkedInAdapter
├── WhatsAppAdapter (via WhatsApp Web protocol)
├── InstagramAdapter
├── TelegramAdapter (via TDLib or MTProto)
├── XTwitterAdapter
├── GmailAdapter (Google API + OAuth)
├── OutlookAdapter (Microsoft Graph + OAuth)
└── IMAPAdapter (IMAP/SMTP direct)
```

### Sync architecture

```
Provider → Polling/WebSocket → Sync Worker → Normalize → DB + Webhook Dispatch
```

Each account runs a sync loop:
1. Poll provider for new data (or maintain persistent connection for WhatsApp/Telegram)
2. Normalize into unified schema
3. Store in PostgreSQL
4. Dispatch webhook events to registered URLs
5. Update `last_synced_at` on account

### Security

- API keys are scoped per organization with optional IP allowlisting
- Provider credentials are AES-256 encrypted at rest
- Webhook payloads are signed with HMAC-SHA256
- Provider sessions are isolated per account (no cross-contamination)
- Proxy support per account to prevent IP-based rate limiting

---

## 8. SDK (planned)

### Node.js

```typescript
import { Ondapile } from '@ondapile/sdk';

const client = new Ondapile({
  apiKey: 'cnd_sk_live_abc123',
  baseUrl: 'https://api.ondapile.local'
});

// List all chats across all providers
const chats = await client.chats.list({ limit: 25 });

// Send a LinkedIn message
await client.chats.sendMessage('chat_a1b2c3d4', {
  text: 'Hi Sarah, thanks for connecting!'
});

// Send an email
await client.emails.send({
  account_id: 'acc_01jgpb4ezwf',
  to: [{ identifier: 'john@widgets.io', display_name: 'John Davis' }],
  subject: 'Partnership Proposal',
  body: '<html>...</html>'
});

// Listen for webhooks
client.webhooks.on('message.received', (event) => {
  console.log(`New message from ${event.data.sender_id}`);
});
```

### Python

```python
from ondapile import Ondapile

client = Ondapile(
    api_key="cnd_sk_live_abc123",
    base_url="https://api.ondapile.local"
)

# List accounts
accounts = client.accounts.list(status="OPERATIONAL")

# Search LinkedIn
results = client.linkedin.search(
    account_id="acc_01jgpb44tjf",
    type="PEOPLE",
    keywords="product manager",
    filters={"location": "San Francisco"}
)
```

---

## Appendix A: OpenAPI spec

The full OpenAPI 3.1 spec is available at:

```
GET /api-json   → JSON format
GET /api-yaml   → YAML format
```

Import into Postman, Insomnia, or any OpenAPI-compatible tool.

## Appendix B: Comparison to Unipile

| Feature | Ondapile | Unipile |
|---------|---------|---------|
| Self-hostable | Yes | No (cloud only) |
| Open source | Yes (MIT) | No |
| Pricing | Infrastructure cost only | Per-account monthly fee |
| Provider coverage | Same scope | Same scope |
| Data residency | Your servers | EU (Scaleway, France) |
| Hosted auth wizard | Yes | Yes |
| Webhook delivery | Yes | Yes |
| Provider-specific endpoints | LinkedIn only | LinkedIn only |
| SDK | Node.js, Python | Node.js |
| MCP integration | Planned | Yes |
| n8n integration | Planned | Yes |

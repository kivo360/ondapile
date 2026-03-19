# ONDAPILE — Coding Agent Handoff Document

**Project:** Ondapile — Self-hosted unified communication API
**Owner:** Kevin (SenseiiWyze)
**Date:** March 19, 2026
**Status:** Greenfield — nothing built yet

---

## 1. What is this project?

Ondapile is a self-hostable REST API that unifies messaging (WhatsApp, Telegram, LinkedIn, Instagram), email (Gmail, Outlook, IMAP), and calendar (Google, Outlook) into a single normalized schema. Think of it as an open-source alternative to Unipile (https://www.unipile.com/).

The first provider to implement is **WhatsApp** using a Go sidecar service built on top of **wuzapi** (https://github.com/asternic/wuzapi), which wraps the **whatsmeow** library (https://github.com/tulir/whatsmeow) — a Go library that communicates directly with WhatsApp's WebSocket servers using the multidevice protocol.

## 2. Architecture overview

```
┌─────────────────┐
│  Consumer app    │  ← Your app, frontend, AI agent, etc.
│  (HTTP client)   │
└────────┬────────┘
         │ REST API (JSON)
         ▼
┌─────────────────────────────────────────────┐
│  Ondapile API Gateway                         │
│  (Go — single binary)                        │
│                                              │
│  ┌──────────┐ ┌───────────┐ ┌─────────────┐ │
│  │ REST     │ │ Webhook   │ │ Provider    │ │
│  │ handlers │ │ dispatch  │ │ router      │ │
│  └──────────┘ └───────────┘ └──────┬──────┘ │
│                                     │        │
│  ┌─────────────────────────────────┐│        │
│  │ Provider adapters (interfaces)  ││        │
│  │ ┌───────────┐ ┌──────────────┐ ││        │
│  │ │ WhatsApp  │ │ Email        │ ││        │
│  │ │ (wuzapi)  │ │ (future)     │ ││        │
│  │ └───────────┘ └──────────────┘ ││        │
│  └─────────────────────────────────┘│        │
└──────────────────────────────────────────────┘
         │
         ▼
┌─────────────────┐   ┌─────────────┐
│  PostgreSQL      │   │  Redis       │
│  (accounts,      │   │  (sessions,  │
│   messages,      │   │   queues)    │
│   webhooks)      │   │              │
└─────────────────┘   └─────────────┘
```

**Key decision: Everything in Go.** Since wuzapi and whatsmeow are Go, and Go is excellent for this type of long-running WebSocket service, the entire API is written in Go. No polyglot sidecar complexity.

The architecture embeds wuzapi's WhatsApp logic directly rather than running it as a separate process. wuzapi's code is the starting point — we fork it, restructure it into a clean adapter pattern, and wrap it with the unified Ondapile API layer.

## 3. Tech stack

| Component | Technology | Why |
|-----------|-----------|-----|
| Language | Go 1.22+ | whatsmeow is Go, wuzapi is Go, high concurrency |
| HTTP framework | Gin or Echo | Mature, fast, middleware ecosystem |
| WhatsApp protocol | whatsmeow (go.mau.fi/whatsmeow) | Only maintained Go WA multidevice library |
| Database | PostgreSQL 16 | JSONB for flexible metadata, pgvector later for AI features |
| Session store | SQLite per device (whatsmeow default) + PostgreSQL for API state | whatsmeow requires SQLite for Signal protocol state |
| Cache / Queue | Redis 7 | Webhook delivery queue, rate limit counters |
| Webhook delivery | Background goroutines + Redis queue | Retry with exponential backoff |
| Auth | API key in X-API-KEY header | Simple, stateless |
| Container | Docker + docker-compose | Easy self-hosting |

## 4. What to fork and how to restructure

### 4.1 Fork wuzapi

Fork https://github.com/asternic/wuzapi. The relevant source files:

| File | What it does | What to keep |
|------|-------------|--------------|
| `main.go` | Entry point, flag parsing, server startup | Restructure as Ondapile entry point |
| `clients.go` | WhatsApp client manager (connect, disconnect, QR) | Core of the WhatsApp adapter |
| `handlers.go` | HTTP route handlers (send message, get chats, etc.) | Refactor into adapter interface |
| `wmiau.go` | whatsmeow event handler (incoming messages, receipts) | Core event normalization logic |
| `db.go` | SQLite/PostgreSQL database operations | Merge with Ondapile's DB layer |
| `routes.go` | Gin router setup | Replace with Ondapile router |
| `helpers.go` | Utility functions | Keep relevant helpers |
| `rabbitmq.go` | RabbitMQ integration | Reference for webhook queue pattern |
| `s3manager.go` | S3 media storage | Keep for media attachment handling |
| `migrations.go` | DB schema migrations | Merge with Ondapile migrations |
| `constants.go` | Shared constants | Merge |

### 4.2 New directory structure

```
ondapile/
├── cmd/
│   └── ondapile/
│       └── main.go                  # Entry point
├── internal/
│   ├── api/                         # HTTP layer
│   │   ├── router.go                # Route definitions
│   │   ├── middleware.go            # Auth, rate limiting, logging
│   │   ├── accounts.go             # /accounts endpoints
│   │   ├── chats.go                # /chats endpoints
│   │   ├── messages.go             # /messages endpoints
│   │   ├── emails.go               # /emails endpoints (future)
│   │   ├── calendars.go            # /calendars endpoints (future)
│   │   ├── webhooks.go             # /webhooks endpoints
│   │   └── errors.go               # Unified error responses
│   ├── adapter/                     # Provider adapter interface
│   │   ├── adapter.go              # Interface definition
│   │   └── registry.go             # Adapter registry
│   ├── whatsapp/                    # WhatsApp adapter (from wuzapi)
│   │   ├── adapter.go              # Implements adapter.Provider interface
│   │   ├── client.go               # whatsmeow client lifecycle
│   │   ├── events.go               # Event handler (from wmiau.go)
│   │   ├── send.go                 # Send messages (from handlers.go)
│   │   ├── media.go                # Upload/download media
│   │   └── session.go              # QR code, pairing, reconnection
│   ├── model/                       # Unified data models
│   │   ├── account.go              # Account model
│   │   ├── chat.go                 # Chat model
│   │   ├── message.go              # Message model
│   │   ├── attendee.go             # Attendee model
│   │   ├── email.go                # Email model (future)
│   │   ├── calendar.go             # Calendar event model (future)
│   │   ├── webhook.go              # Webhook config model
│   │   └── pagination.go           # Cursor-based pagination
│   ├── store/                       # Database layer
│   │   ├── postgres.go             # PostgreSQL connection + migrations
│   │   ├── accounts.go             # Account CRUD
│   │   ├── chats.go                # Chat CRUD
│   │   ├── messages.go             # Message CRUD
│   │   └── webhooks.go             # Webhook CRUD
│   ├── webhook/                     # Webhook dispatch engine
│   │   ├── dispatcher.go           # Background worker, retry logic
│   │   ├── signer.go               # HMAC-SHA256 signing
│   │   └── events.go               # Event type definitions
│   └── config/                      # Configuration
│       └── config.go               # Env vars, defaults
├── migrations/                      # SQL migration files
│   ├── 001_create_accounts.sql
│   ├── 002_create_chats.sql
│   ├── 003_create_messages.sql
│   └── 004_create_webhooks.sql
├── docker-compose.yml
├── Dockerfile
├── .env.sample
├── go.mod
├── go.sum
└── README.md
```

## 5. Core interfaces

### 5.1 Provider adapter interface

This is the most important abstraction. Every provider (WhatsApp, Gmail, LinkedIn, etc.) implements this interface. The API layer never talks to whatsmeow directly — it talks through the adapter.

```go
// internal/adapter/adapter.go
package adapter

import (
    "context"
    "ondapile/internal/model"
)

type Provider interface {
    // Lifecycle
    Name() string                                          // "WHATSAPP", "GMAIL", etc.
    Connect(ctx context.Context, accountID string, creds map[string]string) (*model.Account, error)
    Disconnect(ctx context.Context, accountID string) error
    Reconnect(ctx context.Context, accountID string) (*model.Account, error)
    Status(ctx context.Context, accountID string) (model.AccountStatus, error)

    // Auth flow
    GetAuthChallenge(ctx context.Context, accountID string) (*AuthChallenge, error)  // QR code, OAuth URL, etc.
    SolveCheckpoint(ctx context.Context, accountID string, solution string) error

    // Messaging
    ListChats(ctx context.Context, accountID string, opts ListOpts) (*model.PaginatedList[model.Chat], error)
    GetChat(ctx context.Context, accountID string, chatID string) (*model.Chat, error)
    ListMessages(ctx context.Context, accountID string, chatID string, opts ListOpts) (*model.PaginatedList[model.Message], error)
    SendMessage(ctx context.Context, accountID string, chatID string, msg SendMessageRequest) (*model.Message, error)
    StartChat(ctx context.Context, accountID string, req StartChatRequest) (*model.Chat, error)

    // Attendees
    ListAttendees(ctx context.Context, accountID string, opts ListOpts) (*model.PaginatedList[model.Attendee], error)
    GetAttendee(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error)

    // Media
    DownloadAttachment(ctx context.Context, accountID string, messageID string, attachmentID string) ([]byte, string, error)  // bytes, mimeType, error
}

type AuthChallenge struct {
    Type    string `json:"type"`     // "QR_CODE", "PAIRING_CODE", "OAUTH_URL", "CREDENTIALS"
    Payload string `json:"payload"`  // QR data, URL, etc.
    Expiry  int64  `json:"expiry"`   // Unix timestamp
}

type ListOpts struct {
    Cursor string
    Limit  int
    Before *time.Time
    After  *time.Time
}

type SendMessageRequest struct {
    Text        string              `json:"text"`
    Attachments []AttachmentUpload  `json:"attachments,omitempty"`
    QuotedMsgID *string             `json:"quoted_message_id,omitempty"`
}

type AttachmentUpload struct {
    Filename string `json:"filename"`
    Content  []byte `json:"content"`  // base64 decoded
    MimeType string `json:"mime_type"`
}

type StartChatRequest struct {
    AttendeeIdentifier string `json:"attendee_identifier"`  // phone number, username, etc.
    Text               string `json:"text"`
}
```

### 5.2 Unified data models

```go
// internal/model/account.go
package model

type AccountStatus string

const (
    StatusOperational  AccountStatus = "OPERATIONAL"
    StatusAuthRequired AccountStatus = "AUTH_REQUIRED"
    StatusCheckpoint   AccountStatus = "CHECKPOINT"
    StatusInterrupted  AccountStatus = "INTERRUPTED"
    StatusPaused       AccountStatus = "PAUSED"
    StatusConnecting   AccountStatus = "CONNECTING"
)

type Account struct {
    Object       string            `json:"object"`        // always "account"
    ID           string            `json:"id"`            // ondapile internal ID: "acc_xxxx"
    Provider     string            `json:"provider"`      // "WHATSAPP", "GMAIL", etc.
    Name         string            `json:"name"`          // display name
    Identifier   string            `json:"identifier"`    // phone, email, username
    Status       AccountStatus     `json:"status"`
    StatusDetail *string           `json:"status_detail"` // error message if applicable
    Capabilities []string          `json:"capabilities"`  // ["messaging", "media", "groups"]
    CreatedAt    time.Time         `json:"created_at"`
    LastSyncedAt *time.Time        `json:"last_synced_at"`
    Metadata     map[string]any    `json:"metadata"`
}
```

```go
// internal/model/message.go
package model

type Message struct {
    Object      string          `json:"object"`       // always "message"
    ID          string          `json:"id"`           // "msg_xxxx"
    ChatID      string          `json:"chat_id"`
    AccountID   string          `json:"account_id"`
    Provider    string          `json:"provider"`
    ProviderID  string          `json:"provider_id"`  // whatsmeow message ID
    Text        string          `json:"text"`
    SenderID    string          `json:"sender_id"`    // attendee ID or phone JID
    IsSender    bool            `json:"is_sender"`
    Timestamp   time.Time       `json:"timestamp"`
    Attachments []Attachment    `json:"attachments"`
    Reactions   []Reaction      `json:"reactions"`
    Quoted      *QuotedMessage  `json:"quoted"`
    Seen        bool            `json:"seen"`
    Delivered   bool            `json:"delivered"`
    Edited      bool            `json:"edited"`
    Deleted     bool            `json:"deleted"`
    Hidden      bool            `json:"hidden"`
    IsEvent     bool            `json:"is_event"`
    EventType   *int            `json:"event_type"`
    Metadata    map[string]any  `json:"metadata"`
}

type Attachment struct {
    ID        string `json:"id"`
    Filename  string `json:"filename"`
    MimeType  string `json:"mime_type"`
    Size      int64  `json:"size"`
    URL       string `json:"url,omitempty"`  // if stored in S3
}

type Reaction struct {
    Value    string `json:"value"`     // emoji
    SenderID string `json:"sender_id"`
    IsSender bool   `json:"is_sender"`
}

type QuotedMessage struct {
    ID   string `json:"id"`
    Text string `json:"text"`
}
```

```go
// internal/model/chat.go
package model

type Chat struct {
    Object      string          `json:"object"`       // always "chat"
    ID          string          `json:"id"`           // "chat_xxxx"
    AccountID   string          `json:"account_id"`
    Provider    string          `json:"provider"`
    ProviderID  string          `json:"provider_id"`  // JID for WhatsApp
    Type        string          `json:"type"`         // ONE_TO_ONE, GROUP
    Name        *string         `json:"name"`         // group name or null
    Attendees   []Attendee      `json:"attendees"`
    LastMessage *MessagePreview `json:"last_message"`
    UnreadCount int             `json:"unread_count"`
    IsGroup     bool            `json:"is_group"`
    IsArchived  bool            `json:"is_archived"`
    CreatedAt   time.Time       `json:"created_at"`
    UpdatedAt   time.Time       `json:"updated_at"`
}
```

## 6. WhatsApp adapter implementation guide

### 6.1 Mapping wuzapi concepts to Ondapile

| wuzapi concept | Ondapile concept | Notes |
|---------------|-----------------|-------|
| User (wuzapi user with token) | Account (`provider: "WHATSAPP"`) | Each wuzapi user = one connected WhatsApp number |
| wuzapi admin token | Ondapile API key | Single auth layer, not per-user |
| JID (`628xxx@s.whatsapp.net`) | `chat.provider_id` and `attendee.provider_id` | JID is WhatsApp's universal identifier |
| `events.Message` | `model.Message` | Normalize all message types |
| QR code / pairing code | `AuthChallenge` | Part of account connection flow |
| Webhook URL per user | Ondapile webhook dispatch | Centralized, not per-account |
| SQLite device store | Keep as-is per device | whatsmeow Signal protocol state MUST stay in SQLite |

### 6.2 Critical whatsmeow patterns

**One whatsmeow.Client per connected WhatsApp account:**
```go
container, _ := sqlstore.New("sqlite3", "file:devices/acc_xxxx.db?_foreign_keys=on", nil)
deviceStore, _ := container.GetFirstDevice()
client := whatsmeow.NewClient(deviceStore, nil)
```

**Event handler receives all events for the account:**
```go
client.AddEventHandler(func(evt interface{}) {
    switch v := evt.(type) {
    case *events.Message:
        // Normalize to model.Message, store in PostgreSQL, dispatch webhook
    case *events.Receipt:
        // Update read/delivered status
    case *events.Connected:
        // Update account status to OPERATIONAL
    case *events.Disconnected:
        // Update account status to INTERRUPTED
    case *events.HistorySync:
        // Bulk import chat history
    }
})
```

**QR code flow:**
```go
if client.Store.ID == nil {
    qrChan, _ := client.GetQRChannel(context.Background())
    client.Connect()
    for evt := range qrChan {
        if evt.Event == "code" {
            // Return QR code data to API caller
        }
    }
} else {
    client.Connect()  // Resume existing session
}
```

**Sending a text message:**
```go
targetJID := types.NewJID("628123456789", types.DefaultUserServer)
msg := &waE2E.Message{
    Conversation: proto.String("Hello from Ondapile!"),
}
resp, err := client.SendMessage(ctx, targetJID, msg, whatsmeow.SendRequestExtra{
    ID: client.GenerateMessageID(),
})
```

**Sending media:**
```go
// 1. Upload to WhatsApp servers
uploaded, err := client.Upload(ctx, fileBytes, whatsmeow.MediaImage)

// 2. Build message with upload reference
msg := &waE2E.Message{
    ImageMessage: &waE2E.ImageMessage{
        URL:           proto.String(uploaded.URL),
        DirectPath:    proto.String(uploaded.DirectPath),
        MediaKey:      uploaded.MediaKey,
        FileEncSHA256: uploaded.FileEncSHA256,
        FileSHA256:    uploaded.FileSHA256,
        FileLength:    proto.Uint64(uint64(len(fileBytes))),
        Mimetype:      proto.String("image/jpeg"),
        Caption:       proto.String("Check this out"),
    },
}
```

### 6.3 Message normalization (whatsmeow → Ondapile)

This is the core translation logic. Map from `*events.Message` to `model.Message`:

```go
func normalizeMessage(evt *events.Message, accountID string) *model.Message {
    msg := &model.Message{
        Object:    "message",
        ID:        generateOndapileID("msg"),
        AccountID: accountID,
        Provider:  "WHATSAPP",
        ProviderID: evt.Info.ID,
        ChatID:    chatIDFromJID(evt.Info.Chat),
        SenderID:  evt.Info.Sender.String(),
        IsSender:  evt.Info.IsFromMe,
        Timestamp: evt.Info.Timestamp,
        Seen:      false,
        Delivered:  true,
        Metadata:  map[string]any{},
    }

    // Extract text from various message types
    waMsg := evt.Message
    switch {
    case waMsg.GetConversation() != "":
        msg.Text = waMsg.GetConversation()
    case waMsg.GetExtendedTextMessage() != nil:
        msg.Text = waMsg.GetExtendedTextMessage().GetText()
        if waMsg.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage() != nil {
            msg.Quoted = &model.QuotedMessage{
                ID:   waMsg.GetExtendedTextMessage().GetContextInfo().GetStanzaID(),
                Text: waMsg.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage().GetConversation(),
            }
        }
    case waMsg.GetImageMessage() != nil:
        msg.Text = waMsg.GetImageMessage().GetCaption()
        msg.Attachments = append(msg.Attachments, model.Attachment{
            ID:       generateOndapileID("att"),
            MimeType: waMsg.GetImageMessage().GetMimetype(),
            Size:     int64(waMsg.GetImageMessage().GetFileLength()),
        })
    case waMsg.GetDocumentMessage() != nil:
        msg.Text = waMsg.GetDocumentMessage().GetCaption()
        msg.Attachments = append(msg.Attachments, model.Attachment{
            ID:       generateOndapileID("att"),
            Filename: waMsg.GetDocumentMessage().GetFileName(),
            MimeType: waMsg.GetDocumentMessage().GetMimetype(),
            Size:     int64(waMsg.GetDocumentMessage().GetFileLength()),
        })
    case waMsg.GetAudioMessage() != nil:
        msg.Attachments = append(msg.Attachments, model.Attachment{
            ID:       generateOndapileID("att"),
            MimeType: waMsg.GetAudioMessage().GetMimetype(),
            Size:     int64(waMsg.GetAudioMessage().GetFileLength()),
        })
    case waMsg.GetVideoMessage() != nil:
        msg.Text = waMsg.GetVideoMessage().GetCaption()
        msg.Attachments = append(msg.Attachments, model.Attachment{
            ID:       generateOndapileID("att"),
            MimeType: waMsg.GetVideoMessage().GetMimetype(),
            Size:     int64(waMsg.GetVideoMessage().GetFileLength()),
        })
    case waMsg.GetReactionMessage() != nil:
        msg.IsEvent = true
        eventType := 1
        msg.EventType = &eventType
        msg.Text = waMsg.GetReactionMessage().GetText()
    case waMsg.GetProtocolMessage() != nil:
        msg.Hidden = true
    }

    return msg
}
```

## 7. Database schema (PostgreSQL)

### 001_create_accounts.sql
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE accounts (
    id              TEXT PRIMARY KEY DEFAULT 'acc_' || replace(uuid_generate_v4()::text, '-', ''),
    provider        TEXT NOT NULL,
    name            TEXT NOT NULL,
    identifier      TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'CONNECTING',
    status_detail   TEXT,
    capabilities    JSONB NOT NULL DEFAULT '[]',
    credentials_enc BYTEA,                          -- AES-256 encrypted
    proxy_config    JSONB,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_synced_at  TIMESTAMPTZ,
    UNIQUE(provider, identifier)
);

CREATE INDEX idx_accounts_provider ON accounts(provider);
CREATE INDEX idx_accounts_status ON accounts(status);
```

### 002_create_chats.sql
```sql
CREATE TABLE chats (
    id              TEXT PRIMARY KEY DEFAULT 'chat_' || replace(uuid_generate_v4()::text, '-', ''),
    account_id      TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL,
    provider_id     TEXT NOT NULL,                   -- JID for WhatsApp
    type            TEXT NOT NULL DEFAULT 'ONE_TO_ONE',
    name            TEXT,
    is_group        BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived     BOOLEAN NOT NULL DEFAULT FALSE,
    unread_count    INTEGER NOT NULL DEFAULT 0,
    last_message_at TIMESTAMPTZ,
    last_message_preview TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(account_id, provider_id)
);

CREATE INDEX idx_chats_account ON chats(account_id);
CREATE INDEX idx_chats_updated ON chats(updated_at DESC);
```

### 003_create_messages.sql
```sql
CREATE TABLE messages (
    id              TEXT PRIMARY KEY DEFAULT 'msg_' || replace(uuid_generate_v4()::text, '-', ''),
    chat_id         TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    account_id      TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL,
    provider_id     TEXT NOT NULL,
    text            TEXT,
    sender_id       TEXT NOT NULL,
    is_sender       BOOLEAN NOT NULL DEFAULT FALSE,
    timestamp       TIMESTAMPTZ NOT NULL,
    attachments     JSONB NOT NULL DEFAULT '[]',
    reactions       JSONB NOT NULL DEFAULT '[]',
    quoted          JSONB,
    seen            BOOLEAN NOT NULL DEFAULT FALSE,
    delivered       BOOLEAN NOT NULL DEFAULT FALSE,
    edited          BOOLEAN NOT NULL DEFAULT FALSE,
    deleted         BOOLEAN NOT NULL DEFAULT FALSE,
    hidden          BOOLEAN NOT NULL DEFAULT FALSE,
    is_event        BOOLEAN NOT NULL DEFAULT FALSE,
    event_type      INTEGER,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(account_id, provider_id)
);

CREATE INDEX idx_messages_chat ON messages(chat_id, timestamp DESC);
CREATE INDEX idx_messages_account ON messages(account_id);
CREATE INDEX idx_messages_sender ON messages(sender_id);
```

### 004_create_webhooks.sql
```sql
CREATE TABLE webhooks (
    id          TEXT PRIMARY KEY DEFAULT 'whk_' || replace(uuid_generate_v4()::text, '-', ''),
    url         TEXT NOT NULL,
    events      JSONB NOT NULL DEFAULT '[]',
    secret      TEXT NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE webhook_deliveries (
    id          BIGSERIAL PRIMARY KEY,
    webhook_id  TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event       TEXT NOT NULL,
    payload     JSONB NOT NULL,
    status_code INTEGER,
    attempts    INTEGER NOT NULL DEFAULT 0,
    next_retry  TIMESTAMPTZ,
    delivered   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_deliveries_pending ON webhook_deliveries(next_retry) WHERE delivered = FALSE;
```

### 005_create_attendees.sql
```sql
CREATE TABLE attendees (
    id              TEXT PRIMARY KEY DEFAULT 'att_' || replace(uuid_generate_v4()::text, '-', ''),
    account_id      TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL,
    provider_id     TEXT NOT NULL,                   -- JID, email, username
    name            TEXT,
    identifier      TEXT NOT NULL,
    identifier_type TEXT NOT NULL,                   -- PHONE_NUMBER, EMAIL, USERNAME, etc.
    avatar_url      TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(account_id, provider_id)
);

CREATE INDEX idx_attendees_account ON attendees(account_id);
```

## 8. API endpoints (Phase 1 — WhatsApp only)

### Authentication
All requests require `X-API-KEY` header. Single key for the instance, stored in env var `ONDAPILE_API_KEY`.

### Endpoints to implement

```
# Accounts
GET    /api/v1/accounts                          List all connected accounts
POST   /api/v1/accounts                          Connect a new WhatsApp account (returns QR/pairing challenge)
GET    /api/v1/accounts/:id                       Get account details + status
DELETE /api/v1/accounts/:id                       Disconnect and delete
POST   /api/v1/accounts/:id/reconnect            Re-connect expired session
GET    /api/v1/accounts/:id/auth-challenge        Get current QR code / pairing code
POST   /api/v1/accounts/:id/checkpoint            Solve 2FA checkpoint

# Chats
GET    /api/v1/chats                              List all chats (cross-account, filterable)
POST   /api/v1/chats                              Start a new chat
GET    /api/v1/chats/:id                          Get chat details
GET    /api/v1/chats/:id/messages                 List messages in chat
POST   /api/v1/chats/:id/messages                 Send a message

# Messages (cross-chat)
GET    /api/v1/messages                           List all messages (cross-account)
GET    /api/v1/messages/:id                       Get a single message
GET    /api/v1/messages/:id/attachments/:att_id   Download attachment
POST   /api/v1/messages/:id/reactions             Add reaction

# Attendees
GET    /api/v1/attendees                          List all attendees
GET    /api/v1/attendees/:id                      Get attendee details

# Webhooks
GET    /api/v1/webhooks                           List webhooks
POST   /api/v1/webhooks                           Create webhook
DELETE /api/v1/webhooks/:id                       Delete webhook
```

### Query parameters for list endpoints

| Param | Type | Used on | Description |
|-------|------|---------|-------------|
| `cursor` | string | all lists | Pagination cursor from previous response |
| `limit` | int | all lists | Items per page (default 25, max 100) |
| `account_id` | string | chats, messages, attendees | Filter by account |
| `before` | ISO datetime | chats, messages | Items before this timestamp |
| `after` | ISO datetime | chats, messages | Items after this timestamp |
| `is_group` | bool | chats | Filter group vs 1-to-1 |

### Response format

All responses follow this envelope:

**Success (single resource):**
```json
{
    "object": "account",
    "id": "acc_xxxx",
    ...fields
}
```

**Success (list):**
```json
{
    "object": "list",
    "items": [...],
    "cursor": "base64_encoded_cursor_or_null",
    "has_more": true
}
```

**Error:**
```json
{
    "object": "error",
    "status": 404,
    "code": "NOT_FOUND",
    "message": "Account not found",
    "details": null
}
```

## 9. Webhook events (Phase 1)

```json
{
    "event": "message.received",
    "timestamp": "2026-03-19T14:30:00Z",
    "data": { ...message object }
}
```

Signed with HMAC-SHA256, signature in `X-Ondapile-Signature` header.

**Events to implement:**
- `account.connected`
- `account.disconnected`
- `account.status_changed`
- `account.checkpoint`
- `message.received`
- `message.sent`
- `message.read` (from receipt events)
- `message.reaction`
- `chat.created`

**Delivery:** POST to registered webhook URL. Retry 3 times with exponential backoff (10s, 60s, 300s). Store failed deliveries in `webhook_deliveries` table.

## 10. Implementation order

### Sprint 1: Foundation (3-5 days)
1. Scaffold Go project with directory structure from Section 4.2
2. Set up PostgreSQL with migrations from Section 7
3. Implement config loading from env vars
4. Implement API key auth middleware
5. Implement error response helpers
6. Implement cursor-based pagination helpers

### Sprint 2: Account management (3-5 days)
1. Port wuzapi's client manager (`clients.go`) into `internal/whatsapp/client.go`
2. Implement `Provider` interface for WhatsApp adapter
3. Build `POST /accounts` → starts whatsmeow client, returns QR code
4. Build `GET /accounts/:id/auth-challenge` → returns current QR
5. Build `GET /accounts/:id` → returns account with live status from whatsmeow
6. Build `DELETE /accounts/:id` → disconnects and cleans up
7. Build `GET /accounts` → list all from PostgreSQL
8. Handle account reconnection on server restart (iterate all OPERATIONAL accounts, call `client.Connect()`)

### Sprint 3: Messaging (3-5 days)
1. Port wuzapi's event handler (`wmiau.go`) into `internal/whatsapp/events.go`
2. Implement message normalization (Section 6.3)
3. On incoming message event: normalize → store in PostgreSQL → create/update chat record → dispatch webhook
4. Build `GET /chats` and `GET /chats/:id`
5. Build `GET /chats/:id/messages`
6. Build `POST /chats/:id/messages` (text + media)
7. Build `POST /chats` (start new chat by phone number)
8. Handle receipt events → update `seen`/`delivered` flags → dispatch `message.read` webhook

### Sprint 4: Webhooks + polish (2-3 days)
1. Build webhook CRUD endpoints
2. Implement webhook dispatcher with HMAC signing
3. Implement retry queue with Redis (or in-memory for MVP)
4. Build `GET /messages` (cross-chat, cross-account)
5. Build attachment download endpoint
6. Add Docker support (Dockerfile + docker-compose.yml)
7. Write README with setup instructions

### Sprint 5: Hardening (2-3 days)
1. Rate limiting middleware (per API key, using Redis)
2. Graceful shutdown (disconnect all whatsmeow clients cleanly)
3. Health check endpoint (`GET /health`)
4. Structured logging (JSON format)
5. Basic metrics (connected accounts, messages/sec, webhook delivery rate)

## 11. Reference materials

| Resource | URL | What to use it for |
|----------|-----|-------------------|
| wuzapi source code | https://github.com/asternic/wuzapi | Fork base, reference implementation |
| whatsmeow godoc | https://pkg.go.dev/go.mau.fi/whatsmeow | All client methods, event types |
| whatsmeow example | https://github.com/tulir/whatsmeow (mdtest) | CLI reference implementation |
| GOWA (alternative reference) | https://github.com/aldinokemal/go-whatsapp-web-multidevice | Multi-device patterns, MCP integration |
| Ondapile API spec | unified-api-spec.md (companion file) | Full API design with all data models |
| Unipile API docs | https://developer.unipile.com/reference | Reference for API design decisions |
| Unipile message object | https://developer.unipile.com/docs/message-payload | Field-level reference for message normalization |

## 12. Key decisions log

| Decision | Chosen | Rationale |
|----------|--------|-----------|
| Language | Go | whatsmeow is Go, wuzapi is Go, no polyglot overhead |
| Fork wuzapi vs build from scratch | Fork | 80% of WhatsApp logic already works, just needs restructuring |
| Sidecar vs embedded | Embedded | Single binary, no IPC overhead, simpler deployment |
| SQLite for whatsmeow sessions | Keep SQLite | whatsmeow's Signal protocol state requires it, can't be moved to PostgreSQL |
| PostgreSQL for API state | PostgreSQL | JSONB for flexible metadata, proper indexing, future pgvector for AI |
| Webhook delivery | Redis queue | Reliable retry, doesn't block request handling |
| Auth | API key (X-API-KEY) | Simple, stateless, sufficient for self-hosted |
| ID format | `{prefix}_{uuid_no_hyphens}` | Human-readable prefix (acc_, msg_, chat_) + collision-free UUID |

## 13. What NOT to build in Phase 1

- Email provider (Gmail, Outlook, IMAP) — Phase 2
- Calendar provider — Phase 2
- LinkedIn/Instagram/Telegram providers — Phase 3
- Hosted auth wizard (white-label OAuth page) — Phase 2
- SDK (Node.js, Python) — Phase 2
- Dashboard UI — Phase 3
- Multi-tenant / organization support — Phase 3
- End-to-end encryption for stored messages — Phase 2
- pgvector embeddings for message search — Phase 3

## 14. Success criteria

Phase 1 is done when you can:

1. `POST /accounts` with a WhatsApp number → get a QR code
2. Scan the QR code with your phone → account status becomes `OPERATIONAL`
3. Send a WhatsApp message from your phone → webhook fires with normalized `message.received` event
4. `POST /chats/:id/messages` with text → message appears on recipient's phone
5. `GET /chats` → returns all WhatsApp conversations with last message previews
6. `GET /chats/:id/messages` → returns full message history for a conversation
7. Restart the server → all accounts automatically reconnect
8. All of the above works from a single `docker-compose up`

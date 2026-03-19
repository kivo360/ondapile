# ondapile

Self-hosted unified communication API

[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## What is ondapile?

Ondapile is a self-hostable REST API that unifies messaging and email into a single normalized schema. Connect your WhatsApp account (QR code) and email accounts (IMAP/SMTP), then send and receive through one API.

Currently supported providers:

- WhatsApp (via whatsmeow multidevice protocol)
- Email (IMAP/SMTP — any provider)

Planned: Telegram, LinkedIn, Instagram, Gmail OAuth, Outlook OAuth, Calendar

## Quick Start

### Prerequisites

- Go 1.25+
- PostgreSQL
- Redis

### Setup

1. Clone the repository:

```bash
git clone <repo>
cd ondapile
cp .env.sample .env
# Edit .env with your settings
```

2. Create the database:

```bash
createdb ondapile
```

3. Start Redis:

```bash
redis-server --port 6380 --daemonize yes
```

4. Build and run:

```bash
CGO_ENABLED=1 go build -o ondapile ./cmd/ondapile
./ondapile
```

## Docker

```bash
docker-compose up
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ONDAPILE_API_KEY` | (required) | API key for authentication |
| `ONDAPILE_PORT` | 8080 | Server port |
| `ONDAPILE_HOST` | 0.0.0.0 | Server host |
| `DB_HOST` | localhost | PostgreSQL host |
| `DB_PORT` | 5432 | PostgreSQL port |
| `DB_USER` | postgres | PostgreSQL user |
| `DB_PASSWORD` | | PostgreSQL password |
| `DB_NAME` | ondapile | Database name |
| `DB_SSLMODE` | disable | SSL mode |
| `REDIS_HOST` | localhost | Redis host |
| `REDIS_PORT` | 6379 | Redis port |
| `WA_DEVICE_STORE_PATH` | ./devices | WhatsApp SQLite device store directory |
| `ONDAPILE_ENCRYPTION_KEY` | (auto) | Passphrase for credential encryption (defaults to API key) |
| `LOG_LEVEL` | info | Log level: debug, info, warn, error |

## API Overview

All requests require the `X-API-KEY` header for authentication.

### Accounts

```
GET    /api/v1/accounts                   List accounts
POST   /api/v1/accounts                   Connect account
GET    /api/v1/accounts/:id               Get account
DELETE /api/v1/accounts/:id               Disconnect
POST   /api/v1/accounts/:id/reconnect     Reconnect
GET    /api/v1/accounts/:id/qr            QR code as PNG
```

### Chats

```
GET    /api/v1/chats                      List chats
POST   /api/v1/chats                      Start new chat
GET    /api/v1/chats/:id                  Get chat
PATCH  /api/v1/chats/:id                  Archive/read/pin
DELETE /api/v1/chats/:id                  Delete
GET    /api/v1/chats/:id/messages         List messages
POST   /api/v1/chats/:id/messages         Send message
```

### Messages

```
GET    /api/v1/messages                   List all messages
GET    /api/v1/messages/:id               Get message
DELETE /api/v1/messages/:id               Delete message
POST   /api/v1/messages/:id/reactions     Add reaction
```

### Emails

```
GET    /api/v1/emails                     List emails
POST   /api/v1/emails                     Send email
GET    /api/v1/emails/:id                 Get email
PUT    /api/v1/emails/:id                 Update (folder, read)
DELETE /api/v1/emails/:id                 Delete
GET    /api/v1/emails/folders             List folders
```

### Webhooks

```
GET    /api/v1/webhooks                   List webhooks
POST   /api/v1/webhooks                   Create webhook
DELETE /api/v1/webhooks/:id               Delete webhook
```

### Other

```
GET    /health                            Health check
GET    /metrics                           System metrics
GET    /wa/qr/:id?key=API_KEY             WhatsApp QR page (browser)
```

## Connecting WhatsApp

```bash
# 1. Create account
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "X-API-KEY: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"provider": "WHATSAPP", "identifier": "my-phone"}'

# 2. Open QR page in browser
open "http://localhost:8080/wa/qr/ACCOUNT_ID?key=your-api-key"

# 3. Scan with WhatsApp → Linked Devices → Link a Device
```

## Connecting Email (IMAP)

```bash
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "X-API-KEY: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "IMAP",
    "identifier": "user@example.com",
    "credentials": {
      "imap_host": "imap.example.com",
      "imap_port": "993",
      "imap_username": "user@example.com",
      "imap_password": "password",
      "smtp_host": "smtp.example.com",
      "smtp_port": "587",
      "smtp_username": "user@example.com",
      "smtp_password": "password"
    }
  }'
```

## Webhooks

Register a URL to receive real-time events:

```bash
curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "X-API-KEY: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-app.com/webhook",
    "events": ["message.received", "email.received"]
  }'
```

### Events

- `account.connected`
- `account.disconnected`
- `account.status_changed`
- `account.checkpoint`
- `message.received`
- `message.sent`
- `message.read`
- `message.reaction`
- `message.deleted`
- `chat.created`
- `email.received`
- `email.sent`

## Running Tests

```bash
# Create test database
createdb ondapile_test

# Run integration tests
CGO_ENABLED=1 go test ./tests/integration/... -v -count=1
```

## Architecture

- **Go/Gin**: HTTP framework and routing
- **PostgreSQL**: Persistent storage for accounts, messages, and metadata
- **Redis**: Caching and session management
- **whatsmeow**: WhatsApp Web multidevice protocol
- **go-imap/go-mail**: Email protocol handling
- **Provider adapter interface**: Unified abstraction across all communication channels
- **AES-256-GCM**: Credential encryption at rest
- **HMAC-SHA256**: Webhook signature verification

## License

MIT

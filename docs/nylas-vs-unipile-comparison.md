# Nylas vs Unipile: Actors, User Flows & SDK Patterns

> Reference doc for ondapile development. Shows how both platforms model actors, handle user flows, and what their SDKs look/feel like — so we can cherry-pick the best patterns.
>
> Created: 2026-03-25

---

## At a Glance

| Dimension | Nylas | Unipile |
|-----------|-------|---------|
| **Focus** | Email + Calendar + Contacts | Messaging + Email + Calendar |
| **Providers** | Gmail, Outlook, IMAP, Yahoo, iCloud | LinkedIn, WhatsApp, Instagram, Telegram, Gmail, Outlook, IMAP |
| **Auth model** | OAuth → `grant_id` | Hosted Auth Wizard → `account_id` |
| **API version** | v3 | v1 |
| **Base URL** | `https://api.us.nylas.com/v3/` | `https://{YOUR_DSN}/api/v1/` |
| **Auth header** | `Authorization: Bearer <API_KEY>` | `X-API-KEY: <API_KEY>` |
| **SDK languages** | Python, Node, Ruby, Java, Kotlin | Node.js (official) |
| **Pricing** | Per-connected-account | €5/account/month |
| **Self-host** | No (cloud only) | No (cloud only) |
| **Open-source clone** | — | **ondapile** ← that's us |

---

## 1. Actors

### Side-by-Side

| # | Actor | Nylas | Unipile | Ondapile |
|---|-------|-------|---------|----------|
| 1 | **Platform Operator** | Nylas (the company) | Unipile (the company) | Whoever deploys the instance |
| 2 | **Developer** | Builds with SDK/API, manages app in Dashboard | Builds with SDK/API, manages via Dashboard | Builds with SDK/API, manages via API keys |
| 3 | **End User** | Email account holder who authorizes via OAuth | Account owner who authenticates via hosted auth wizard | Account connector who authenticates via hosted auth |
| 4 | **Admin** | Manages connectors, webhooks, grants in Dashboard | Manages accounts, webhooks in Dashboard | Org Admin — team mgmt, billing (mocked) |
| 5 | **Webhook Consumer** | Developer's server receiving events | Developer's server receiving events | Publisher's server — dispatcher + SDK exist |

### Key Difference

**Nylas** has a simpler actor model — it's email-first, so there's really just Developer ↔ End User ↔ Provider. The Developer registers a "connector" (Google/Microsoft OAuth app credentials) and end users authenticate through Nylas's hosted OAuth.

**Unipile** has a richer actor model because it spans messaging platforms where auth is more complex — QR codes (WhatsApp/Telegram), session cookies (LinkedIn/Instagram), and OAuth (Gmail/Outlook). The "hosted auth wizard" is a single UI that handles ALL these auth methods.

**Ondapile** inherits Unipile's actor model but adds a Platform Operator actor (because it's self-hosted).

### Actor Interaction Maps

#### Nylas: Who talks to whom

```
                    ┌──────────────────┐
                    │   Nylas (Cloud)  │
                    │  Platform Oper.  │
                    │  manages tokens, │
                    │  connectors,     │
                    │  webhook routing │
                    └────────┬─────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
     ┌──────────────┐  ┌──────────┐  ┌──────────────┐
     │  Developer A │  │ Dev B    │  │  Developer C │
     │  (SaaS app)  │  │ (CRM)    │  │  (AI tool)   │
     └──────┬───────┘  └────┬─────┘  └──────┬───────┘
            │               │               │
   API calls with           │        API calls with
   grant_id ─────────────► ALL ◄──────── grant_id
            │               │               │
            ▼               ▼               ▼
     ┌──────────────────────────────────────────────┐
     │              End Users                       │
     │  Authenticate via OAuth (Google/Microsoft)   │
     │  Each user = 1 grant_id                      │
     └──────────────────┬─────────────────────────┘
                         │
            ┌────────────┼────────────┐
            ▼            ▼            ▼
     ┌──────────┐ ┌──────────┐ ┌──────────┐
     │  Gmail   │ │ Outlook  │ │   IMAP   │
     │  (OAuth) │ │ (OAuth)  │ │  (creds) │
     └──────────┘ └──────────┘ └──────────┘

Interactions:
  Developer ──► Nylas API ──► Provider    (read/send email)
  Developer ──► Nylas Dashboard           (manage connectors, view grants)
  End User  ──► Nylas OAuth ──► Provider  (one-time consent)
  Nylas     ──► Developer's webhook URL   (event notifications)
  Developer  ✗  End User credentials      (never touches tokens)
```

#### Unipile: Who talks to whom

```
                    ┌──────────────────┐
                    │ Unipile (Cloud)  │
                    │  Platform Oper.  │
                    │  manages sessions│
                    │  proxies, tokens │
                    │  hosted auth UI  │
                    └────────┬─────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
     ┌──────────────┐  ┌──────────┐  ┌──────────────┐
     │ Publisher A  │  │ Pub B    │  │ Publisher C  │
     │ (Outreach)   │  │ (ATS)    │  │ (AI Agent)   │
     └──────┬───────┘  └────┬─────┘  └──────┬───────┘
            │               │               │
   API calls with           │        API calls with
   account_id ───────────► ALL ◄────── account_id
            │               │               │
            ▼               ▼               ▼
     ┌──────────────────────────────────────────────┐
     │              End Users                       │
     │  Authenticate via Hosted Auth Wizard         │
     │  (OAuth, QR code, credentials, 2FA)          │
     │  Each user = 1 account_id per provider       │
     └──────────────────┬─────────────────────────┘
                         │
       ┌────────┬────────┼────────┬────────┐
       ▼        ▼        ▼        ▼        ▼
  ┌────────┐┌────────┐┌──────┐┌────────┐┌──────────┐
  │LinkedIn││WhatsApp││Gmail ││Telegram││Instagram │
  │(cookie)││ (QR)   ││(OAuth││ (QR)   ││ (cookie) │
  └────────┘└────────┘└──────┘└────────┘└──────────┘

Interactions:
  Publisher ──► Unipile API ──► Provider  (send msg/email, read inbox)
  Publisher ──► Unipile Dashboard         (manage accounts, view logs)
  End User  ──► Hosted Auth Wizard        (multi-provider auth UI)
  Unipile   ──► Publisher's webhook URL   (event notifications)
  Unipile   ──► Publisher's notify_url    (account connection status)
  Publisher  ✗  End User credentials      (never touches tokens/cookies)
  Actions appear AS the End User          ("auth on behalf", not bot)
```

#### Ondapile: Who talks to whom (self-hosted)

```
┌──────────────────────────────────────────────────────────┐
│              PLATFORM OPERATOR (you)                      │
│  Deploys instance, configures OAuth creds, sets billing   │
│  ┌─────────┐ ┌──────────┐ ┌────────┐ ┌──────────────┐    │
│  │ Deploy  │ │ Google/  │ │ Health │ │ Billing      │    │
│  │ Docker  │ │ MS OAuth │ │ Monitor│ │ (Stripe)     │    │
│  └─────────┘ └──────────┘ └────────┘ └──────────────┘    │
└──────────────────────┬───────────────────────────────────┘
                       │ self-hosted API
         ┌─────────────┼─────────────┐
         ▼             ▼             ▼
┌──────────────┐ ┌──────────┐ ┌──────────────┐
│ SaaS         │ │ SaaS     │ │ SaaS         │
│ Publisher A  │ │ Pub B    │ │ Publisher C  │
│ (ATS)        │ │ (CRM)    │ │ (AI Agent)   │
│              │ │          │ │              │
│ Org Admin ◄──manages team, billing───►     │
└──────┬───────┘ └────┬─────┘ └──────┬───────┘
       │              │              │
       ▼              ▼              ▼
┌──────────────┐              ┌──────────────┐
│  Developer   │              │  Developer   │
│  (SDK/API)   │              │  (SDK/API)   │
└──────┬───────┘              └──────┬───────┘
       │                             │
       ▼                             ▼
┌──────────────┐              ┌──────────────┐
│  End Users   │              │  End Users   │
│  (connect    │              │  (connect    │
│   accounts)  │              │   accounts)  │
└──────┬───────┘              └──────┬───────┘
       │                             │
  ┌────┼─────┬──────┐          ┌─────┼────┐
  ▼    ▼     ▼      ▼          ▼     ▼    ▼
Gmail IMAP LinkedIn WhatsApp  Outlook Telegram
(OAuth)(creds)(cookie) (QR)   (OAuth)  (QR)

       ┌──────────────────────────┐
       │   Webhook Consumer       │
       │  (Publisher's server)    │
       │  Receives 24+ event      │
       │  types via HMAC-signed   │
       │  POST to their URL       │
       └──────────────────────────┘

Interactions:
  Operator   ──► Deploy + configure instance       (one-time setup)
  Operator   ──► Monitor /health, /metrics          (ongoing)
  Publisher  ──► Ondapile API with X-API-Key        (CRUD operations)
  Org Admin  ──► Manage team members, billing       (dashboard)
  Developer  ──► SDK/API integration                (build time)
  End User   ──► Hosted Auth (OAuth/QR/creds)       (one-time per provider)
  Ondapile   ──► Webhook Consumer (HMAC-signed)     (real-time events)
  Publisher   ✗  End User credentials                (encrypted at rest, AES-256-GCM)
  Operator has FULL data sovereignty                 (self-hosted advantage)
```

### 6 Actors × Interaction Matrix (Ondapile)

| | Platform Operator | Publisher | Developer | End User | Org Admin | Webhook Consumer |
|-|-------------------|-----------|-----------|----------|-----------|-----------------|
| **Platform Operator** | — | Provides API platform | Provides SDK/docs | Provides hosted auth | Provides billing config | Provides webhook infra |
| **Publisher** | Pays/subscribes | — | Employs | Serves (embeds auth) | Manages | Owns the server |
| **Developer** | Uses deploy docs | Builds for | — | Implements auth flow | Reports to | Implements handler |
| **End User** | — | Uses their app | — | — | Belongs to org | Triggers events |
| **Org Admin** | — | Part of | Manages access | Manages team | — | Configures URLs |
| **Webhook Consumer** | — | Receives events for | Code by developer | Triggered by actions | — | — |

---

## 2. User Flows

### Flow 1: Connect an Account

#### Nylas (OAuth-first)

```
Developer                    Nylas                       Provider (Google/Microsoft)
    │                          │                              │
    │  1. Create connector     │                              │
    │  (register OAuth creds)  │                              │
    │─────────────────────────►│                              │
    │                          │                              │
    │  2. Generate OAuth URL   │                              │
    │─────────────────────────►│                              │
    │  ◄─────────────────────  │                              │
    │  (auth URL returned)     │                              │
    │                          │                              │
    │  3. Redirect End User ──────────────────────────────────►
    │                          │     4. User consents          │
    │                          │  ◄────────────────────────────│
    │                          │     (auth code)               │
    │                          │                              │
    │  5. Callback with        │                              │
    │     grant_id             │                              │
    │  ◄───────────────────────│                              │
    │                          │                              │
    │  All future calls use grant_id                          │
```

**Nylas Python:**
```python
from nylas import Client

nylas = Client(api_key="nylas_api_key_here")

# Generate auth URL
auth_url = nylas.auth.url_for_oauth2(
    client_id="your_client_id",
    provider="google",
    redirect_uri="https://yourapp.com/oauth/callback",
    login_hint="user@gmail.com",
    scope=["email"]
)
# → redirect user to auth_url

# Handle callback
response = nylas.auth.exchange_code_for_token(
    client_id="your_client_id",
    client_secret="your_client_secret",
    code=request.args["code"],
    redirect_uri="https://yourapp.com/oauth/callback"
)
grant_id = response.grant_id  # ← store this
```

**Nylas Node:**
```javascript
import Nylas from "nylas";

const nylas = new Nylas({ apiKey: "nylas_api_key_here" });

// Generate auth URL
const authUrl = nylas.auth.urlForOAuth2({
    clientId: "your_client_id",
    provider: "google",
    redirectUri: "https://yourapp.com/oauth/callback",
    loginHint: "user@gmail.com",
    scope: ["email"],
});
// → res.redirect(authUrl)

// Handle callback
const { grantId } = await nylas.auth.exchangeCodeForToken({
    clientId: "your_client_id",
    clientSecret: "your_client_secret",
    code: req.query.code,
    redirectUri: "https://yourapp.com/oauth/callback",
});
// → store grantId
```

---

#### Unipile (Hosted Auth Wizard)

```
Developer                    Unipile                     Provider (any)
    │                          │                              │
    │  1. POST /hosted/        │                              │
    │     accounts/link        │                              │
    │  { type: "create",       │                              │
    │    providers: ["*"],      │                              │
    │    notify_url: "..." }   │                              │
    │─────────────────────────►│                              │
    │  ◄─────────────────────  │                              │
    │  { url: "https://        │                              │
    │    account.unipile..." } │                              │
    │                          │                              │
    │  2. Redirect End User ──►│  3. Hosted Auth Wizard       │
    │                          │     shows provider picker    │
    │                          │     handles OAuth/QR/creds   │
    │                          │─────────────────────────────►│
    │                          │  ◄───────────────────────────│
    │                          │                              │
    │  4. Webhook to           │                              │
    │     notify_url           │                              │
    │  { status: "CREATION_    │                              │
    │    SUCCESS",             │                              │
    │    account_id: "e54m.."} │                              │
    │  ◄───────────────────────│                              │
    │                          │                              │
    │  All future calls use account_id                        │
```

**Unipile Node:**
```javascript
import { UnipileClient } from "unipile-node-sdk";

const client = new UnipileClient("https://your-dsn", "your_api_key");

// Generate hosted auth link (handles ALL providers in one UI)
const { url } = await client.account.createHostedAuthLink({
    type: "create",
    api_url: "https://your-dsn",
    providers: "*",  // or ["LINKEDIN", "WHATSAPP", "GOOGLE"]
    notify_url: "https://yourapp.com/callback",
    name: "user_123",
    expiresOn: "2026-12-31T00:00:00Z",
});
// → redirect user to url

// Webhook handler receives:
// { status: "CREATION_SUCCESS", account_id: "e54m8LR22bA7G5qsAc8w", name: "user_123" }
```

**Unipile curl:**
```bash
curl -X POST https://your-dsn/api/v1/hosted/accounts/link \
  -H 'X-API-KEY: your_api_key' \
  -H 'Content-Type: application/json' \
  -d '{
    "type": "create",
    "providers": ["LINKEDIN", "GOOGLE"],
    "api_url": "https://your-dsn",
    "notify_url": "https://yourapp.com/callback",
    "name": "user_123"
  }'
```

#### Pattern Comparison

| Aspect | Nylas | Unipile |
|--------|-------|---------|
| Auth UI | You build the redirect yourself | Pre-built hosted wizard (white-label) |
| Provider selection | One connector per provider type | One link handles all providers |
| Token management | Nylas manages refresh tokens | Unipile manages sessions/tokens |
| Result delivery | Sync (callback URL returns grant_id) | Async (webhook to notify_url) |
| Reconnection | New OAuth flow | `type: "reconnect"` with same endpoint |
| User identifier | `grant_id` | `account_id` |

---

### Flow 2: Read Email

#### Nylas Python
```python
# List messages
messages, _, _ = nylas.messages.list(
    grant_id,
    query_params={"limit": 10, "in": "INBOX"}
)

for msg in messages:
    print(f"From: {msg.from_[0].email}")
    print(f"Subject: {msg.subject}")
    print(f"Snippet: {msg.snippet}")
    print(f"Unread: {msg.unread}")

# Get single message with full body
message, _ = nylas.messages.find(grant_id, message_id)
print(message.body)  # full HTML body

# List threads (grouped conversations)
threads, _, _ = nylas.threads.list(
    grant_id,
    query_params={"limit": 5}
)
```

#### Nylas Node
```javascript
const messages = await nylas.messages.list({
    identifier: grantId,
    queryParams: { limit: 10, in: "INBOX" },
});

for (const msg of messages.data) {
    console.log(`From: ${msg.from[0].email}`);
    console.log(`Subject: ${msg.subject}`);
    console.log(`Unread: ${msg.unread}`);
}

// Get full message
const message = await nylas.messages.find({
    identifier: grantId,
    messageId: "msg_abc123",
});
```

#### Unipile Node
```javascript
// List emails
const emails = await client.email.getAllEmails({ account_id: "acc_xyz" });

// Get single email
const email = await client.email.getEmail("email_id_here");

// List folders
const folders = await client.email.getAllFolders({ account_id: "acc_xyz" });
```

#### Unipile curl
```bash
# List emails
curl -X GET "https://your-dsn/api/v1/emails?account_id=acc_xyz" \
  -H 'X-API-KEY: your_api_key'

# Get single email
curl -X GET "https://your-dsn/api/v1/emails/email_id_here" \
  -H 'X-API-KEY: your_api_key'
```

#### Ondapile (what we have now)
```bash
# List emails
curl -X GET "http://localhost:8080/api/v1/emails?account_id=acc_xxx" \
  -H 'X-API-Key: your_api_key'

# Get single email
curl -X GET "http://localhost:8080/api/v1/emails/email_id" \
  -H 'X-API-Key: your_api_key'
```

---

### Flow 3: Send Email

#### Nylas Python
```python
message = nylas.messages.send(
    grant_id,
    request_body={
        "to": [{"name": "Alice", "email": "alice@example.com"}],
        "cc": [{"name": "Bob", "email": "bob@example.com"}],
        "subject": "Quarterly Report",
        "body": "<h1>Report</h1><p>See attached.</p>",
        "reply_to": [{"email": "noreply@company.com"}],
    }
)
print(f"Sent: {message.id}")
```

#### Nylas Node
```javascript
const sentMessage = await nylas.messages.send({
    identifier: grantId,
    requestBody: {
        to: [{ name: "Alice", email: "alice@example.com" }],
        subject: "Quarterly Report",
        body: "<h1>Report</h1><p>See attached.</p>",
    },
});
```

#### Unipile Node
```javascript
await client.email.sendEmail({
    account_id: "acc_xyz",
    to: [{ email: "alice@example.com", display_name: "Alice" }],
    subject: "Quarterly Report",
    body: "<h1>Report</h1><p>See attached.</p>",
});
```

#### Unipile curl
```bash
curl -X POST "https://your-dsn/api/v1/emails" \
  -H 'X-API-KEY: your_api_key' \
  -H 'Content-Type: application/json' \
  -d '{
    "account_id": "acc_xyz",
    "to": [{"email": "alice@example.com", "display_name": "Alice"}],
    "subject": "Quarterly Report",
    "body": "<h1>Report</h1>"
  }'
```

#### Ondapile (what we have now)
```bash
curl -X POST "http://localhost:8080/api/v1/emails" \
  -H 'X-API-Key: your_api_key' \
  -H 'Content-Type: application/json' \
  -d '{
    "account_id": "acc_xxx",
    "to": "alice@example.com",
    "subject": "Quarterly Report",
    "body_html": "<h1>Report</h1>"
  }'
```

---

### Flow 4: Webhooks

#### Nylas — Create Subscription
```javascript
const webhook = await nylas.webhooks.create({
    requestBody: {
        triggerTypes: ["message.created", "message.updated"],
        webhookUrl: "https://yourapp.com/webhooks/nylas",
        description: "Email notifications",
        notificationEmailAddress: "alerts@yourapp.com",
    },
});
```

#### Nylas — Incoming Payload
```json
{
    "specversion": "1.0",
    "type": "message.created",
    "source": "/google/emails/realtime",
    "id": "webhook_event_id",
    "time": 1723821985,
    "data": {
        "application_id": "app_id",
        "object": {
            "id": "msg_abc",
            "grant_id": "grant_xyz",
            "from": [{"email": "sender@example.com", "name": "Sender"}],
            "to": [{"email": "you@example.com"}],
            "subject": "Hello",
            "snippet": "Preview text...",
            "unread": true,
            "folders": ["INBOX"]
        }
    }
}
```

#### Unipile — Create Subscription
```bash
curl -X POST "https://your-dsn/api/v1/webhooks" \
  -H 'X-API-KEY: your_api_key' \
  -H 'Content-Type: application/json' \
  -d '{
    "request_url": "https://yourapp.com/webhooks/unipile",
    "source": "messaging",
    "events": ["message.created", "mail_received"]
  }'
```

#### Unipile — Incoming Payload
```json
{
    "event": "message.created",
    "timestamp": "2025-01-09T14:32:00Z",
    "account_id": "acc_xyz",
    "data": {
        "id": "msg_xyz123",
        "provider": "linkedin",
        "sender": { "name": "Sarah", "profile_url": "https://..." },
        "content": "Thanks for connecting!",
        "thread_id": "thread_abc"
    }
}
```

#### Webhook Event Comparison

| Event Category | Nylas | Unipile |
|----------------|-------|---------|
| New message | `message.created` | `message_received` / `mail_received` |
| Message updated | `message.updated` | `message_read`, `message_edited` |
| Message sent | `message.send_success` | `mail_sent` |
| Message failed | `message.send_failed` | — |
| Bounce | `message.bounce_detected` | — |
| Account connected | `grant.created` | `CREATION_SUCCESS` (via notify_url) |
| Account expired | `grant.expired` | `account.status: CREDENTIALS` |
| Email opened | — | `email_opened` |
| Link clicked | — | `link_clicked` |
| Calendar event | `event.created/updated/deleted` | `event_created/updated/deleted` |

---

### Flow 5: Send a Message (Messaging — Unipile only)

Nylas doesn't do messaging. This is Unipile-only territory (and ondapile's differentiator).

```javascript
// Send to existing chat
await client.messaging.sendMessage({
    chat_id: "chat_abc",
    text: "Hello! Following up on our conversation.",
});

// Start new chat (find/create 1:1 with a user)
await client.messaging.startNewChat({
    account_id: "acc_xyz",
    text: "Hi! I'd love to connect.",
    attendees_ids: ["linkedin_profile_id_here"],
});
```

```bash
# Start new chat via curl
curl -X POST "https://your-dsn/api/v1/chats" \
  -H 'X-API-KEY: your_api_key' \
  -F account_id=acc_xyz \
  -F 'text=Hi! I would love to connect.' \
  -F attendees_ids=linkedin_profile_id_here
```

---

## 3. API Structure Comparison

### URL Patterns

| Resource | Nylas | Unipile | Ondapile |
|----------|-------|---------|----------|
| List emails | `GET /v3/grants/{grant_id}/messages` | `GET /api/v1/emails?account_id=X` | `GET /api/v1/emails?account_id=X` |
| Get email | `GET /v3/grants/{grant_id}/messages/{id}` | `GET /api/v1/emails/{id}` | `GET /api/v1/emails/{id}` |
| Send email | `POST /v3/grants/{grant_id}/messages/send` | `POST /api/v1/emails` | `POST /api/v1/emails` |
| Delete email | `DELETE /v3/grants/{grant_id}/messages/{id}` | `DELETE /api/v1/emails/{id}` | `DELETE /api/v1/emails/{id}` |
| List folders | `GET /v3/grants/{grant_id}/folders` | `GET /api/v1/folders?account_id=X` | `GET /api/v1/folders?account_id=X` |
| Drafts | `POST /v3/grants/{grant_id}/drafts` | `POST /api/v1/emails/drafts` | — (not yet) |
| List chats | — | `GET /api/v1/chats` | `GET /api/v1/chats` |
| Send message | — | `POST /api/v1/chats/{id}/messages` | `POST /api/v1/messages` |
| Webhooks | `POST /v3/webhooks/` | `POST /api/v1/webhooks` | `POST /api/v1/webhooks` |
| Accounts | `GET /v3/grants/` | `GET /api/v1/accounts` | `GET /api/v1/accounts` |

### Key Design Difference

**Nylas** puts `grant_id` in the URL path: `/v3/grants/{grant_id}/messages`
- Pro: RESTful, clear ownership
- Con: Every endpoint needs the grant_id in the path

**Unipile** puts `account_id` as a query param: `/api/v1/emails?account_id=X`
- Pro: Simpler endpoint structure, can omit for "all accounts" queries
- Con: Less RESTful

**Ondapile** follows Unipile's pattern (query param).

---

## 4. Message/Email Object Comparison

### Nylas Message Object
```json
{
    "id": "msg_abc123",
    "object": "message",
    "grant_id": "grant_xyz",
    "thread_id": "thread_001",
    "subject": "Hello",
    "from": [{"name": "Alice", "email": "alice@example.com"}],
    "to": [{"name": "Bob", "email": "bob@example.com"}],
    "cc": [],
    "bcc": [],
    "reply_to": [],
    "date": 1706811644,
    "unread": true,
    "starred": false,
    "folders": ["INBOX", "UNREAD"],
    "snippet": "Preview of message body...",
    "body": "<html>Full HTML body</html>",
    "attachments": [
        {"id": "att_1", "filename": "report.pdf", "size": 25000, "content_type": "application/pdf"}
    ],
    "created_at": 1706811644
}
```

### Unipile Email Object
```json
{
    "id": "email_abc123",
    "account_id": "acc_xyz",
    "provider": "gmail",
    "from": {"email": "alice@example.com", "display_name": "Alice"},
    "to": [{"email": "bob@example.com", "display_name": "Bob"}],
    "cc": [],
    "bcc": [],
    "subject": "Hello",
    "body": "<html>Full HTML body</html>",
    "body_plain": "Plain text version",
    "date": "2026-03-25T10:00:00Z",
    "read": false,
    "folder": "INBOX",
    "attachments": [
        {"id": "att_1", "filename": "report.pdf", "size": 25000, "mime_type": "application/pdf"}
    ],
    "tracking": {
        "opens": 0,
        "clicks": 0
    }
}
```

### Notable Differences

| Field | Nylas | Unipile |
|-------|-------|---------|
| User identifier | `grant_id` | `account_id` |
| From field | Array of `{name, email}` | Single `{email, display_name}` |
| Read status | `unread: true/false` | `read: true/false` (inverted) |
| Folders | Array: `["INBOX", "UNREAD"]` | Single string: `"INBOX"` |
| Date format | Unix timestamp | ISO 8601 string |
| Provider field | Not included | `"gmail"`, `"outlook"`, etc. |
| Tracking | Not included | `{opens, clicks}` built-in |
| Threads | `thread_id` field + `/threads` endpoint | Thread via conversation grouping |

---

## 5. SDK Initialization Patterns

### Nylas — Minimal Setup
```python
# Python
from nylas import Client
nylas = Client(api_key="nyk_v0_xxx", api_uri="https://api.us.nylas.com")
```
```javascript
// Node
import Nylas from "nylas";
const nylas = new Nylas({ apiKey: "nyk_v0_xxx", apiUri: "https://api.us.nylas.com" });
```

### Unipile — Minimal Setup
```javascript
// Node (only official SDK)
import { UnipileClient } from "unipile-node-sdk";
const client = new UnipileClient("https://your-dsn.unipile.com", "your_api_key");
```

### Ondapile — Minimal Setup
```python
# Python (raw requests — no official SDK yet)
import requests
BASE = "http://localhost:8080/api/v1"
HEADERS = {"X-API-Key": "your_key"}
requests.get(f"{BASE}/emails", params={"account_id": "acc_xxx"}, headers=HEADERS)
```
```javascript
// Node SDK exists
import { OndapileClient } from "@ondapile/sdk";
const client = new OndapileClient({ apiKey: "your_key", baseUrl: "http://localhost:8080" });
```

---

## 6. What Ondapile Should Steal

### From Nylas
1. **Thread-first email model** — `thread_id` on every message + dedicated `/threads` endpoint. Much better for "inbox" UIs
2. **Python SDK** — Nylas has a polished Python SDK; ondapile should have one too (most AI/automation devs use Python)
3. **Cursor-based pagination** — `next_cursor` / `page_token` pattern is cleaner than offset-based
4. **Webhook CloudEvents format** — `specversion`, `type`, `source` fields follow the CloudEvents spec
5. **Regional endpoints** — US and EU base URLs for data residency

### From Unipile
1. **Hosted auth wizard** — Single endpoint, one UI for ALL providers (ondapile already has this concept)
2. **`account_id` as query param** — Simpler than Nylas's path-based `grant_id` (ondapile already does this)
3. **Email tracking built-in** — `opens` and `clicks` on the email object, pixel/link tracking
4. **Multi-provider messaging** — LinkedIn, WhatsApp, Instagram, Telegram in unified API (ondapile's core differentiator)
5. **Reconnection flow** — `type: "reconnect"` in hosted auth, webhook-driven session management
6. **White-label auth** — CNAME support for `auth.yourapp.com`

### Ondapile's Unique Advantage
- **Self-hosted** — Neither Nylas nor Unipile offers this. Full data sovereignty.
- **Open-source** — Community-driven, auditable, extensible.
- **No per-account billing** — Deploy once, connect unlimited accounts.

---

## 7. Quick Reference: "I want to ___"

| Task | Nylas | Unipile | Ondapile |
|------|-------|---------|----------|
| Connect Gmail | Create Google connector → OAuth redirect | Hosted auth link → wizard | `POST /accounts` with Gmail credentials |
| Read inbox | `nylas.messages.list(grant_id)` | `GET /emails?account_id=X` | `GET /emails?account_id=X` |
| Send email | `nylas.messages.send(grant_id, {...})` | `POST /emails` | `POST /emails` |
| Reply to email | Include `reply_to_message_id` | Include `in_reply_to` | Include `in_reply_to` |
| Send WhatsApp | ❌ Not supported | `POST /chats` with WhatsApp account | `POST /messages` with WhatsApp account |
| Send LinkedIn DM | ❌ Not supported | `POST /chats` with LinkedIn account | `POST /messages` with LinkedIn account |
| Get webhooks | `nylas.webhooks.create({...})` | `POST /webhooks` | `POST /webhooks` |
| Track opens | ❌ Not built-in | ✅ Built-in tracking pixels | ✅ Tracking pixel support |
| Search emails | Provider-native query syntax | Query params filtering | Query params filtering |

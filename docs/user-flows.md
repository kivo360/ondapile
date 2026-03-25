# Ondapile User Flows

> How each actor interacts with ondapile, shown through SDK code as if everything works.
> Comments mark what's not built yet.
>
> Created: 2026-03-25

---

## Actor 1: Platform Operator

The person who deploys and runs the ondapile instance. They never use the SDK — they use Docker, env vars, and CLI tools.

```bash
# ── 1. Deploy ──────────────────────────────────────────────

# Option A: Docker (recommended)
docker-compose up -d

# Option B: Binary
export DB_HOST=localhost DB_USER=ondapile DB_PASSWORD=secret DB_NAME=ondapile
export ONDAPILE_API_KEY=your-master-key
export ONDAPILE_ENCRYPTION_KEY=32-byte-hex-key
export GOOGLE_CLIENT_ID=xxx.apps.googleusercontent.com
export GOOGLE_CLIENT_SECRET=xxx
export MICROSOFT_CLIENT_ID=xxx
export MICROSOFT_CLIENT_SECRET=xxx
export BASE_URL=https://api.yourcompany.com  # for tracking pixels + OAuth callbacks

./ondapile serve


# ── 2. Verify it's running ─────────────────────────────────

curl http://localhost:8080/health
# → { "status": "ok" }

curl http://localhost:8080/metrics
# → { "db_pool": { "total": 10, "idle": 8 }, "goroutines": 12, "uptime": "2h30m" }


# ── 3. Configure OAuth providers ───────────────────────────

# Google: console.cloud.google.com → APIs & Services → Credentials
#   Authorized redirect URI: https://api.yourcompany.com/api/v1/oauth/callback/gmail
#
# Microsoft: portal.azure.com → App registrations → Authentication
#   Redirect URI: https://api.yourcompany.com/api/v1/oauth/callback/outlook


# ── 4. Monitor (ongoing) ──────────────────────────────────

# Health check (for uptime monitors like UptimeRobot)
curl http://localhost:8080/health

# Database + webhook queue depth
# [NOT YET BUILT] ondapile admin dashboard at /admin
# [NOT YET BUILT] ondapile admin CLI: ondapile status, ondapile orgs list
# For now: direct PostgreSQL queries
psql -c "SELECT COUNT(*) FROM webhook_deliveries WHERE status = 'pending'"
```

---

## Actor 2: SaaS Publisher

A company (ATS, CRM, AI agent tool) that integrates ondapile into their product. They sign up on the dashboard, get API keys, and hand those keys to their developers.

```python
# ── 1. Sign up ─────────────────────────────────────────────

# Publisher visits https://yourondapile.com/signup
# Creates account with email+password or GitHub OAuth
# → Auto-creates organization
# → Lands on /dashboard

# From the dashboard:
#   /dashboard/api-keys/  → Create API key (permissions: full)
#   /dashboard/webhooks/  → Register webhook URL
#   /dashboard/accounts/  → View connected end-user accounts
#   /dashboard/logs/      → Audit trail


# ── 2. Create API key (dashboard UI) ──────────────────────

# Click "Create Key" → choose permissions:
#   full     = all operations
#   read     = read-only
#   email    = email operations only
#   calendar = calendar operations only
#
# → Shows key once: sk_live_abc123...
# → SHA-256 hashed in database (can't be recovered)


# ── 3. Configure webhooks ─────────────────────────────────

from ondapile import Ondapile

client = Ondapile(
    api_key="sk_live_abc123",
    base_url="https://api.yourcompany.com"
)

webhook = client.webhooks.create(
    url="https://myapp.com/webhooks/ondapile",
    events=[
        "email_received",
        "email_sent",
        "email_opened",
        "link_clicked",
        "account_connected",
        "account_credentials",  # session expired, needs reconnect
    ]
)
print(webhook.secret)  # hmac_xxx — store this for signature verification
# [NOT YET BUILT] Python SDK — currently Node.js only + raw HTTP


# ── 4. Generate hosted auth links for end users ───────────

# When a recruiter in your ATS app clicks "Connect Gmail":
auth_link = client.accounts.create_hosted_auth(
    provider="gmail",
    redirect_url="https://myapp.com/settings/email?connected=true",
    # [NOT YET BUILT] name parameter for white-label
    # [NOT YET BUILT] custom_domain="auth.myapp.com"
)
print(auth_link.url)
# → https://api.yourcompany.com/api/v1/oauth/callback/gmail?state=xxx
# Embed this URL in your app — end user clicks it, connects their Gmail


# ── 5. Monitor connected accounts ─────────────────────────

accounts = client.accounts.list()
for acc in accounts:
    print(f"{acc.id} | {acc.provider} | {acc.identifier} | {acc.status}")
    # acc_abc | gmail   | recruiter@company.com | connected
    # acc_def | outlook | sales@company.com     | connected
    # acc_ghi | imap    | support@company.com   | credentials  ← needs reconnect

# Check audit trail
# [NOT YET BUILT] Python SDK for audit log
logs = client.audit_log.list(limit=20)
for log in logs:
    print(f"{log.timestamp} | {log.action} | {log.actor} | {log.details}")
```

---

## Actor 3: Developer

The engineer at the Publisher's company who writes the integration code. This is the primary SDK user.

### 3a. Setup

```python
from ondapile import Ondapile

# Initialize — one client for everything
client = Ondapile(
    api_key="sk_live_abc123",          # from Publisher's dashboard
    base_url="https://api.yourcompany.com"  # ondapile instance URL
)
# [NOT YET BUILT] Python SDK — showing ideal interface
```

```javascript
// Node.js (SDK exists today)
import { OndapileClient } from "@ondapile/sdk";

const client = new OndapileClient({
    apiKey: "sk_live_abc123",
    baseUrl: "https://api.yourcompany.com"
});
```

### 3b. Connect accounts programmatically

```python
# ── Connect IMAP account (direct, no OAuth needed) ────────

account = client.accounts.create(
    provider="imap",
    identifier="support@company.com",
    name="Support Inbox",
    credentials={
        "username": "support@company.com",
        "password": "app-specific-password",
        "imap_host": "imap.gmail.com",
        "imap_port": 993,
        "smtp_host": "smtp.gmail.com",
        "smtp_port": 587,
    }
)
print(account.id)      # acc_0401db0611414b72aeca08ad30ef7f63
print(account.status)  # connected


# ── Connect Gmail via OAuth (hosted auth for end users) ───

auth = client.accounts.create_hosted_auth(
    provider="gmail",
    redirect_url="https://myapp.com/connected",
)
# → Redirect end user to auth.url
# → After OAuth consent, webhook fires: account_connected


# ── Connect WhatsApp via QR ───────────────────────────────
# [NOT YET BUILT in email-only v1, but adapter exists]

wa_account = client.accounts.create(provider="whatsapp")
print(wa_account.qr_url)  # https://api.yourcompany.com/wa/qr/acc_xxx
# → Show QR to end user, they scan with WhatsApp
# → Webhook fires: account_connected
```

### 3c. Email operations

```python
ACCOUNT = "acc_0401db0611414b72aeca08ad30ef7f63"

# ── List inbox ─────────────────────────────────────────────

emails = client.emails.list(
    account_id=ACCOUNT,
    folder="INBOX",          # [NOT YET BUILT] folder filter
    limit=20,
    # [NOT YET BUILT] cursor-based pagination
    # cursor="next_abc123",
)

for email in emails:
    print(f"{'●' if not email.read else '○'} {email.from_address} — {email.subject}")
    print(f"  {email.snippet}")
    print(f"  {email.date} | {len(email.attachments)} attachments")
    print()

# ● alice@example.com — Q1 Report
#   Here's the quarterly report you requested...
#   2026-03-25T10:30:00Z | 2 attachments


# ── Read full email ────────────────────────────────────────

email = client.emails.get("email_abc123")
print(email.subject)
print(email.body_html)      # full HTML body
print(email.body_plain)     # plain text fallback
print(email.from_address)   # alice@example.com
print(email.to)             # [bob@example.com]
print(email.cc)             # [carol@example.com]
print(email.headers)        # raw email headers
# [NOT YET BUILT] thread_id for conversation grouping


# ── Send email ─────────────────────────────────────────────

sent = client.emails.send(
    account_id=ACCOUNT,
    to=["alice@example.com"],
    subject="Re: Q1 Report",
    body_html="<p>Thanks Alice, looks great!</p>",
    # Optional fields:
    cc=["manager@company.com"],
    bcc=["archive@company.com"],
    # [NOT YET BUILT] reply_to header
    # reply_to=["noreply@company.com"],
)
print(sent.id)         # email_def456
print(sent.status)     # sent
# Webhook fires: email_sent


# ── Reply to an email ─────────────────────────────────────

reply = client.emails.reply(
    email_id="email_abc123",
    body_html="<p>Thanks for the update.</p>",
    # reply_all=True,  # [NOT YET BUILT] reply-all toggle
)
# Sets In-Reply-To and References headers automatically
# Webhook fires: email_sent


# ── Forward an email ──────────────────────────────────────

client.emails.forward(
    email_id="email_abc123",
    to=["colleague@company.com"],
    body_html="<p>FYI — see below.</p>",
)
# Webhook fires: email_sent


# ── Send with attachments ─────────────────────────────────

with open("report.pdf", "rb") as f:
    sent = client.emails.send(
        account_id=ACCOUNT,
        to=["alice@example.com"],
        subject="Updated Report",
        body_html="<p>Attached is the updated version.</p>",
        attachments=[
            {"filename": "report.pdf", "content": f.read(), "content_type": "application/pdf"}
        ]
    )


# ── Download attachment ───────────────────────────────────

attachment = client.emails.download_attachment(
    email_id="email_abc123",
    attachment_id="att_001",
)
with open(attachment.filename, "wb") as f:
    f.write(attachment.content)


# ── Mark read/unread ──────────────────────────────────────

client.emails.update("email_abc123", read=True)
client.emails.update("email_abc123", read=False)


# ── Star/flag ─────────────────────────────────────────────

client.emails.update("email_abc123", starred=True)


# ── Move to folder ────────────────────────────────────────

client.emails.update("email_abc123", folder="Archive")
# Webhook fires: email_moved


# ── List folders ──────────────────────────────────────────

folders = client.emails.list_folders(account_id=ACCOUNT)
for folder in folders:
    print(f"{folder.name} ({folder.unread_count} unread)")
# INBOX (12 unread)
# Sent (0 unread)
# Drafts (3 unread)
# Archive (0 unread)
# [Custom] Recruiting (5 unread)


# ── Search ────────────────────────────────────────────────

results = client.emails.list(
    account_id=ACCOUNT,
    q="from:alice subject:report",   # provider-native search syntax
    limit=10,
)


# ── Delete ────────────────────────────────────────────────

client.emails.delete("email_abc123")


# ── Drafts ────────────────────────────────────────────────
# [NOT YET BUILT] Draft endpoints

draft = client.drafts.create(
    account_id=ACCOUNT,
    to=["alice@example.com"],
    subject="Draft: Meeting Notes",
    body_html="<p>Notes from today...</p>",
)
# → Syncs to provider's Drafts folder

client.drafts.update(draft.id, body_html="<p>Updated notes...</p>")
client.drafts.send(draft.id)   # sends and removes from Drafts
client.drafts.delete(draft.id) # discard without sending
```

### 3d. Messaging operations (WhatsApp, LinkedIn, etc.)

```python
# [NOT YET BUILT for v1 — email only, but the API routes exist]

# ── List chats ────────────────────────────────────────────

chats = client.chats.list(account_id="acc_whatsapp_xxx")
for chat in chats:
    print(f"{chat.id} | {chat.attendees[0].name} | {chat.last_message.text[:50]}")


# ── Read messages in a chat ───────────────────────────────

messages = client.chats.get_messages("chat_abc")
for msg in messages:
    print(f"{'→' if msg.is_outbound else '←'} {msg.sender.name}: {msg.text}")
    # → You: Hey, are you available for a call?
    # ← Sarah: Sure, let me know when!


# ── Send a message in existing chat ───────────────────────

client.chats.send_message(
    chat_id="chat_abc",
    text="Great, I'll call at 3pm.",
)


# ── Start a new chat ─────────────────────────────────────

chat = client.chats.create(
    account_id="acc_linkedin_xxx",
    attendees_ids=["linkedin_profile_id"],
    text="Hi! I came across your profile and would love to connect.",
)
# Creates or finds existing 1:1 chat, sends first message
# Webhook fires: message_sent


# ── List attendees (contacts) ─────────────────────────────

attendees = client.attendees.list(account_id="acc_linkedin_xxx")
for person in attendees:
    print(f"{person.id} | {person.name} | {person.provider_id}")
```

### 3e. Webhook handling

```python
# ── Register webhooks ─────────────────────────────────────

webhook = client.webhooks.create(
    url="https://myapp.com/webhooks/ondapile",
    events=["*"],  # all events
)
WEBHOOK_SECRET = webhook.secret


# ── Handle incoming webhooks (Flask example) ──────────────

from flask import Flask, request, abort
import hmac
import hashlib

app = Flask(__name__)

@app.route("/webhooks/ondapile", methods=["POST"])
def handle_webhook():
    # 1. Verify signature
    signature = request.headers.get("X-Ondapile-Signature", "")
    expected = "sha256=" + hmac.new(
        WEBHOOK_SECRET.encode(),
        request.data,
        hashlib.sha256
    ).hexdigest()

    if not hmac.compare_digest(signature, expected):
        abort(401)

    # 2. Parse event
    event = request.json
    event_type = event["event"]
    account_id = event["account_id"]
    data = event["data"]

    # 3. Route by event type
    match event_type:
        # ── Email events ──
        case "email_received":
            # New email arrived in connected inbox
            process_incoming_email(account_id, data)

        case "email_sent":
            # Email was sent through the API
            log_sent_email(data["email_id"])

        case "email_opened":
            # Recipient opened the email (tracking pixel fired)
            record_open(data["email_id"], data["opened_at"])

        case "link_clicked":
            # Recipient clicked a tracked link
            record_click(data["email_id"], data["url"], data["clicked_at"])

        case "email_moved":
            # Email was moved to a different folder
            update_folder(data["email_id"], data["from_folder"], data["to_folder"])

        # ── Account events ──
        case "account_connected":
            # End user successfully connected their account
            activate_user_account(account_id)

        case "account_credentials":
            # Session/token expired — end user needs to reconnect
            send_reconnect_email(account_id)

        case "account_disconnected":
            # Account was removed
            deactivate_user_account(account_id)

        # ── Messaging events ──
        # [NOT YET BUILT for v1]
        case "message_received":
            handle_incoming_message(account_id, data)

        case "message_sent":
            log_outbound_message(data)

    return "OK", 200
```

### 3f. Account management

```python
# ── List all connected accounts ───────────────────────────

accounts = client.accounts.list()
for acc in accounts:
    print(f"{acc.id} | {acc.provider:10} | {acc.identifier:30} | {acc.status}")
# acc_abc | gmail      | recruiter@company.com          | connected
# acc_def | outlook    | sales@company.com              | connected
# acc_ghi | imap       | support@company.com            | credentials
# acc_jkl | whatsapp   | +1234567890                    | connected
# acc_mno | linkedin   | john-doe-123                   | connected


# ── Get single account details ───────────────────────────

acc = client.accounts.get("acc_abc")
print(acc.provider)        # gmail
print(acc.identifier)      # recruiter@company.com
print(acc.status)          # connected
print(acc.capabilities)    # ["email", "calendar"]
print(acc.connected_at)    # 2026-03-20T14:00:00Z
print(acc.last_sync)       # 2026-03-25T10:30:00Z


# ── Reconnect expired account ────────────────────────────

# When webhook fires account_credentials:
reconnect = client.accounts.reconnect("acc_ghi")
# For IMAP: re-validates credentials immediately
# For OAuth: returns a new auth URL for the end user to visit


# ── Delete account ────────────────────────────────────────

client.accounts.delete("acc_abc")
# Removes account + all synced data
# Webhook fires: account_disconnected
```

---

## Actor 4: End User

The recruiter, salesperson, or support agent who connects their real email/messaging account. They **never see ondapile** — they interact through the Publisher's app.

```
There is no SDK code for this actor — they click links in the Publisher's UI.

WHAT THE END USER SEES:
═══════════════════════════════════════════════════════════════

Step 1: In the Publisher's app (e.g., ATS dashboard)
┌──────────────────────────────────────────────────────────┐
│                                                          │
│  Settings > Email Integration                            │
│                                                          │
│  Connect your email to send and receive messages         │
│  directly from [Publisher App Name].                     │
│                                                          │
│  ┌─────────────────┐  ┌─────────────────┐               │
│  │  Connect Gmail  │  │ Connect Outlook │               │
│  └─────────────────┘  └─────────────────┘               │
│                                                          │
│  ┌─────────────────┐                                     │
│  │  Connect IMAP   │  (manual server settings)           │
│  └─────────────────┘                                     │
│                                                          │
└──────────────────────────────────────────────────────────┘

Step 2: Clicks "Connect Gmail" → redirected to Google OAuth
┌──────────────────────────────────────────────────────────┐
│                                                          │
│  Google                                                  │
│                                                          │
│  [Publisher App] wants to access your Google Account      │
│                                                          │
│  This will allow [Publisher App] to:                      │
│  ✓ Read your email messages                              │
│  ✓ Send email on your behalf                             │
│  ✓ Manage your email labels                              │
│                                                          │
│         ┌──────────┐  ┌──────────┐                       │
│         │  Cancel  │  │  Allow   │                       │
│         └──────────┘  └──────────┘                       │
│                                                          │
└──────────────────────────────────────────────────────────┘
# [NOT YET BUILT] White-label: Google shows Publisher's app name,
#                 not "ondapile" — requires Publisher to register
#                 their own Google OAuth app, or Operator configures
#                 custom branding.

Step 3: After clicking "Allow"
┌──────────────────────────────────────────────────────────┐
│                                                          │
│  ✅ Account Connected                                     │
│                                                          │
│  You can close this window.                              │
│                                                          │
└──────────────────────────────────────────────────────────┘
# → Browser redirects back to Publisher's redirect_url
# → Webhook fires to Publisher: account_connected

Step 4: Back in Publisher's app
┌──────────────────────────────────────────────────────────┐
│                                                          │
│  Settings > Email Integration                            │
│                                                          │
│  ✅ Gmail connected: recruiter@company.com                │
│     Connected Mar 25, 2026                               │
│     [Disconnect]                                         │
│                                                          │
│  ┌─────────────────┐                                     │
│  │ Connect Outlook │  (add another account)               │
│  └─────────────────┘                                     │
│                                                          │
└──────────────────────────────────────────────────────────┘

RECONNECTION (when session expires):

Step 1: End user gets email from Publisher
┌──────────────────────────────────────────────────────────┐
│  From: noreply@publisher-app.com                         │
│  Subject: Your email connection needs attention           │
│                                                          │
│  Hi Sarah,                                               │
│                                                          │
│  Your Gmail connection has expired. Click below to        │
│  reconnect so you can keep sending/receiving emails      │
│  from [Publisher App].                                    │
│                                                          │
│  [Reconnect Now]                                         │
│                                                          │
└──────────────────────────────────────────────────────────┘
# This email is sent by the PUBLISHER's app, triggered by
# the account_credentials webhook event.

Step 2: Clicks "Reconnect Now" → same OAuth flow → done
```

---

## Actor 5: Org Admin

The team lead at the Publisher company who manages access and billing. Uses the dashboard exclusively.

```
ORG ADMIN DASHBOARD FLOWS:
═══════════════════════════════════════════════════════════════

1. TEAM MANAGEMENT (/dashboard/settings/team)
┌──────────────────────────────────────────────────────────┐
│  Team Members                           [+ Invite]       │
│                                                          │
│  ┌────────────────────────────────────────────────────┐  │
│  │ Sarah Chen          owner    sarah@company.com    │  │
│  │ Mike Johnson        admin    mike@company.com     │  │
│  │ Lisa Park           member   lisa@company.com  [⋮]│  │
│  └────────────────────────────────────────────────────┘  │
│                                                          │
│  Invite: [email___________] [role: member ▼] [Send]      │
│                                                          │
│  Roles:                                                  │
│    owner  — full access (auto-assigned to creator)       │
│    admin  — manage team + API keys                       │
│    member — use API keys, view accounts                  │
│                                                          │
│  # [NOT YET BUILT] Granular permissions per route        │
│  #   e.g., can a 'member' create webhooks?               │
└──────────────────────────────────────────────────────────┘

2. API KEY MANAGEMENT (/dashboard/api-keys)
┌──────────────────────────────────────────────────────────┐
│  API Keys                              [+ Create Key]    │
│                                                          │
│  ┌────────────────────────────────────────────────────┐  │
│  │ Production Key   sk_live_...abc   full     [Revoke]│  │
│  │ Read-Only Key    sk_live_...def   read     [Revoke]│  │
│  │ Email Service    sk_live_...ghi   email    [Revoke]│  │
│  └────────────────────────────────────────────────────┘  │
│                                                          │
│  Create: [name___________] [perms: full ▼] [Create]      │
│                                                          │
│  ⚠️  Keys are shown once at creation. Store securely.    │
└──────────────────────────────────────────────────────────┘

3. BILLING (/dashboard/settings/billing)
┌──────────────────────────────────────────────────────────┐
│  # [NOT YET BUILT] Entire section is mocked              │
│                                                          │
│  Current Plan: Trial (free)                              │
│  Connected Accounts: 3 / 5 (trial limit)                 │
│  API Calls This Month: 1,247                             │
│                                                          │
│  ┌─────────┐ ┌──────────┐ ┌──────────┐                  │
│  │  Free   │ │ Starter  │ │   Pro    │                  │
│  │ 5 accts │ │ 25 accts │ │ 100 accts│                  │
│  │  $0/mo  │ │  $49/mo  │ │ $199/mo  │                  │
│  │[Current]│ │[Upgrade] │ │[Upgrade] │                  │
│  └─────────┘ └──────────┘ └──────────┘                  │
│                                                          │
│  # [NOT YET BUILT] Stripe integration                    │
│  # [NOT YET BUILT] Usage metering                        │
│  # [NOT YET BUILT] Invoice history                       │
└──────────────────────────────────────────────────────────┘

4. MONITORING (/dashboard/logs)
┌──────────────────────────────────────────────────────────┐
│  Audit Log                                               │
│                                                          │
│  10:30 AM  email_sent      acc_abc   →  alice@example    │
│  10:28 AM  email_received  acc_abc   ←  bob@example      │
│  10:15 AM  account_created acc_def   outlook connected   │
│  09:45 AM  webhook_failed  wh_001   → retry scheduled    │
│  09:30 AM  api_key_created           by sarah@company    │
│                                                          │
│  # [NOT YET BUILT] Real-time usage dashboard             │
│  # [NOT YET BUILT] Webhook delivery success rate         │
│  # [NOT YET BUILT] Account health overview               │
└──────────────────────────────────────────────────────────┘
```

---

## Actor 6: Webhook Consumer

The Publisher's backend server. No human interaction — it's code that receives and processes events.

```python
# ── Express.js webhook handler (Node SDK exists) ──────────

# Node.js version (works today)
import express from "express";
import { WebhooksClient } from "@ondapile/sdk";

const app = express();
app.use(express.raw({ type: "application/json" }));

const WEBHOOK_SECRET = "whsec_xxx";

app.post("/webhooks/ondapile", (req, res) => {
    // 1. Verify signature
    const signature = req.headers["x-ondapile-signature"];
    const isValid = WebhooksClient.verifySignature(
        req.body, signature, WEBHOOK_SECRET
    );
    if (!isValid) return res.status(401).send("Invalid signature");

    // 2. Parse
    const event = JSON.parse(req.body);

    // 3. Process (idempotently — you may receive duplicates)
    switch (event.event) {
        case "email_received":
            // Payload:
            // {
            //   event: "email_received",
            //   account_id: "acc_abc",
            //   timestamp: "2026-03-25T10:30:00Z",
            //   data: {
            //     email_id: "email_xyz",
            //     from: "alice@example.com",
            //     subject: "Q1 Report",
            //     snippet: "Here's the quarterly...",
            //     folder: "INBOX",
            //     has_attachments: true
            //   }
            // }
            await syncEmailToDatabase(event.data);
            break;

        case "email_sent":
            await markAsSent(event.data.email_id);
            break;

        case "email_opened":
            // Tracking pixel was loaded
            await recordOpen(event.data.email_id, event.data.opened_at);
            break;

        case "link_clicked":
            // Tracked link was clicked
            await recordClick(event.data.email_id, event.data.url);
            break;

        case "account_connected":
            // New account ready — start syncing
            await activateAccount(event.account_id);
            break;

        case "account_credentials":
            // Session expired — prompt user to reconnect
            await triggerReconnectFlow(event.account_id);
            break;

        case "account_disconnected":
            await deactivateAccount(event.account_id);
            break;

        // [NOT YET BUILT for v1 — messaging events]
        case "message_received":
            await handleIncomingMessage(event.account_id, event.data);
            break;
    }

    res.status(200).send("OK");
});
```

```python
# ── Python webhook handler (Flask) ────────────────────────
# [NOT YET BUILT] Python SDK — but this is what it would look like

from ondapile import WebhookVerifier
from flask import Flask, request

app = Flask(__name__)
verifier = WebhookVerifier(secret="whsec_xxx")

@app.route("/webhooks/ondapile", methods=["POST"])
def webhook():
    # Verify
    if not verifier.verify(request.data, request.headers.get("X-Ondapile-Signature")):
        return "Unauthorized", 401

    event = request.json

    # The webhook consumer's job is simple:
    # 1. Verify signature (security)
    # 2. Acknowledge quickly (return 200 within 5s)
    # 3. Process asynchronously (queue the work)

    queue.enqueue(process_event, event)
    return "OK", 200


# ── What the webhook delivery looks like ──────────────────
#
# POST https://myapp.com/webhooks/ondapile
# Headers:
#   Content-Type: application/json
#   X-Ondapile-Signature: sha256=a1b2c3d4e5f6...
#   User-Agent: Ondapile-Webhook/1.0
#
# Body:
# {
#   "event": "email_received",
#   "account_id": "acc_0401db0611414b72aeca08ad30ef7f63",
#   "timestamp": "2026-03-25T10:30:00Z",
#   "data": {
#     "email_id": "email_abc123",
#     "from": "alice@example.com",
#     "to": ["bob@company.com"],
#     "subject": "Q1 Report",
#     "snippet": "Here's the quarterly report...",
#     "folder": "INBOX",
#     "has_attachments": true,
#     "attachment_count": 2
#   }
# }
#
# Retry policy (on non-2xx response):
#   Attempt 1: immediate
#   Attempt 2: wait 10 seconds
#   Attempt 3: wait 60 seconds
#   Attempt 4: wait 5 minutes
#   After 4 failures: marked as failed, no more retries
#   Retry poll interval: every 30 seconds
```

---

## Complete Flow: AI Email Auto-Responder

Putting it all together — a Publisher building an AI-powered email tool where a recruiter connects Gmail and the system auto-drafts responses.

```python
from ondapile import Ondapile, WebhookVerifier
from flask import Flask, request
import openai

client = Ondapile(api_key="sk_live_abc123", base_url="https://api.yourcompany.com")
verifier = WebhookVerifier(secret="whsec_xxx")
ai = openai.OpenAI()

app = Flask(__name__)

# ── Step 1: Publisher generates hosted auth link ──────────
# (Called when recruiter clicks "Connect Gmail" in the ATS UI)

@app.route("/api/connect-email")
def connect_email():
    auth = client.accounts.create_hosted_auth(
        provider="gmail",
        redirect_url="https://myapp.com/settings?connected=true",
    )
    return {"auth_url": auth.url}
    # → Frontend redirects recruiter to auth.url
    # → Recruiter consents on Google OAuth screen
    # → Webhook fires: account_connected


# ── Step 2: Handle incoming emails via webhook ────────────

@app.route("/webhooks/ondapile", methods=["POST"])
def webhook():
    if not verifier.verify(request.data, request.headers.get("X-Ondapile-Signature")):
        return "Unauthorized", 401

    event = request.json

    if event["event"] == "email_received":
        # Get full email content
        email = client.emails.get(event["data"]["email_id"])

        # Skip if it's from ourselves or automated
        if email.from_address.endswith("@mycompany.com"):
            return "OK", 200

        # Generate AI response
        completion = ai.chat.completions.create(
            model="gpt-4",
            messages=[
                {"role": "system", "content": "Draft a professional reply. Be concise."},
                {"role": "user", "content": f"Subject: {email.subject}\n\n{email.body_plain}"}
            ]
        )
        draft_body = completion.choices[0].message.content

        # Send reply through the same account
        client.emails.reply(
            email_id=email.id,
            body_html=f"<p>{draft_body}</p>",
        )
        # → Appears in recruiter's Gmail as a sent message
        # → Looks like the recruiter typed it themselves
        # → Webhook fires: email_sent

    elif event["event"] == "account_credentials":
        # Token expired — email the recruiter to reconnect
        notify_user_to_reconnect(event["account_id"])

    return "OK", 200


# ── Step 3: Recruiter checks their Gmail ──────────────────
#
# The recruiter opens Gmail and sees:
#   Sent: "Re: Interested in Senior Engineer role"
#   "Thank you for your interest! I'd love to schedule
#    a call this week..."
#
# It looks like they sent it. Because ondapile acts
# AS the recruiter (auth on behalf), not as a bot.
```

---

## What's Built vs What's Hypothetical

| Feature | Status | Notes |
|---------|--------|-------|
| Go API server (49 endpoints) | ✅ Built | `internal/api/router.go` |
| Node.js SDK | ✅ Built | `sdk/node/` |
| IMAP email adapter | ✅ Built | Send, list, get working |
| Gmail OAuth flow | ✅ Built | OAuth callback + token exchange |
| Outlook OAuth flow | ⚠️ Partial | Callback exists, adapter stubbed |
| Webhook dispatcher + HMAC | ✅ Built | PostgreSQL queue, retry logic |
| Email tracking pixels | ✅ Built | `/t/:id` pixel, `/l/:id` link redirect |
| Dashboard (signup, keys, team) | ✅ Built | TanStack Start frontend |
| **Python SDK** | ❌ Not built | All Python examples are hypothetical |
| **Draft CRUD endpoints** | ❌ Not built | `POST /drafts` not in router |
| **Cursor-based pagination** | ❌ Not built | Currently offset-based |
| **Thread grouping** | ❌ Not built | No `thread_id` on emails |
| **Billing (Stripe)** | ❌ Not built | Dashboard shows mocked data |
| **Usage metering** | ❌ Not built | Hardcoded API call counts |
| **Admin dashboard** | ❌ Not built | No `/admin` routes |
| **White-label auth branding** | ❌ Not built | No custom domain support |
| **Permission enforcement** | ❌ Not built | `email`-scoped key can still hit `/chats` |
| **Messaging in v1** | ❌ Deferred | Routes exist, adapters partial |

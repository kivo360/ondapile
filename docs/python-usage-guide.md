# Ondapile Python Usage Guide

How to build an email app on top of Ondapile's unified API.

## Architecture

```
┌─────────────┐      ┌──────────────────┐      ┌──────────────────┐
│  Your App   │─────▶│  Ondapile API    │─────▶│  Gmail / Outlook │
│  (Python)   │◀─────│  localhost:8080   │◀─────│  / IMAP Server   │
└─────────────┘      └──────────────────┘      └──────────────────┘
     HTTP requests        Unified REST API         Provider adapters
```

Your app never talks to email providers directly. Every call goes through Ondapile.

---

## Setup

```python
import requests

ONDAPILE_URL = "http://localhost:8080"
API_KEY = "your-api-key-here"

HEADERS = {
    "X-API-Key": API_KEY,
    "Content-Type": "application/json",
}

def api(method, path, **kwargs):
    """Helper to call Ondapile API."""
    url = f"{ONDAPILE_URL}{path}"
    resp = requests.request(method, url, headers=HEADERS, **kwargs)
    resp.raise_for_status()
    return resp.json()
```

---

## 1. Connect an Email Account

### Option A: IMAP (direct credentials)

```python
account = api("POST", "/api/v1/accounts", json={
    "provider": "IMAP",
    "identifier": "ryan@usesenseiiwyze.com",
    "name": "Ryan's Work Email",
    "credentials": {
        "imap_host": "imap.gmail.com",
        "imap_port": "993",
        "smtp_host": "smtp.gmail.com",
        "smtp_port": "587",
        "username": "ryan@usesenseiiwyze.com",
        "password": "app-password-here"    # Gmail App Password, not your real password
    }
})

account_id = account["id"]
print(f"Connected: {account_id} — Status: {account['status']}")
# Connected: acc_abc123... — Status: CONNECTING → OPERATIONAL
```

### Option B: Gmail OAuth (browser redirect)

```python
# 1. Create a hosted auth link — user opens this in their browser
auth = api("POST", "/api/v1/accounts/hosted-auth", json={
    "provider": "GOOGLE",
    "type": "email",
})
print(f"Send user to: {auth['url']}")

# 2. User clicks link → Google OAuth → grants access → redirects back to Ondapile
# 3. Ondapile stores tokens automatically, account goes OPERATIONAL
# 4. List accounts to find the new one
accounts = api("GET", "/api/v1/accounts")
for acc in accounts["items"]:
    print(f"{acc['id']} — {acc['provider']} — {acc['status']}")
```

---

## 2. List Accounts

```python
accounts = api("GET", "/api/v1/accounts")

for acc in accounts["items"]:
    print(f"  {acc['id']}")
    print(f"    Provider: {acc['provider']}")
    print(f"    Name:     {acc['name']}")
    print(f"    Status:   {acc['status']}")
    print()

# Example output:
#   acc_0401db0611414b72aeca08ad30ef7f63
#     Provider: IMAP
#     Name:     ryan@usesenseiiwyze.com
#     Status:   OPERATIONAL
```

---

## 3. Read Emails

### List all emails for an account

```python
ACCOUNT_ID = "acc_0401db0611414b72aeca08ad30ef7f63"

emails = api("GET", f"/api/v1/emails?account_id={ACCOUNT_ID}")

for em in emails["items"]:
    print(f"Subject: {em['subject']}")
    print(f"From:    {em['from_attendee']['identifier']}")
    print(f"Date:    {em['date']}")
    print(f"Read:    {em['read']}")
    print(f"Folder:  {em['folders']}")
    print(f"Body:    {em['body_plain'][:100]}...")
    print()
```

### Get a specific email by ID

```python
email_id = "8b95b9b4-fae9-4544-ab17-b85a2dd5b99f"
email = api("GET", f"/api/v1/emails/{email_id}")

print(f"Subject: {email['subject']}")
print(f"Full body: {email['body_plain']}")
print(f"Headers: {len(email['headers'])} headers")
print(f"Attachments: {email['has_attachments']}")
```

### List email folders

```python
folders = api("GET", f"/api/v1/emails/folders?account_id={ACCOUNT_ID}")

# Returns: list of folder objects (not wrapped in "items")
for folder in folders:
    print(f"  {folder['name']:10s}  total={folder['total']}  unread={folder['unread']}")

# Example output:
#   INBOX       total=2  unread=0
#   SENT        total=0  unread=0
#   DRAFTS      total=0  unread=0
#   TRASH       total=0  unread=0
#   SPAM        total=0  unread=0
#   ARCHIVE     total=0  unread=0
```

---

## 4. Send an Email

```python
sent = api("POST", "/api/v1/emails", json={
    "account_id": ACCOUNT_ID,
    "to": [
        {"identifier": "someone@example.com", "identifier_type": "EMAIL_ADDRESS"}
    ],
    "subject": "Hello from Ondapile",
    "body_html": "<p>This email was sent through the Ondapile API.</p>",
    "body_plain": "This email was sent through the Ondapile API.",
})

print(f"Sent! Email ID: {sent['id']}")
```

---

## 5. Reply to an Email

```python
# Reply to a specific email
original_email_id = "8b95b9b4-fae9-4544-ab17-b85a2dd5b99f"

reply = api("POST", f"/api/v1/emails/{original_email_id}/reply", json={
    "account_id": ACCOUNT_ID,
    "body_html": "<p>Thanks for the email! Here's my reply.</p>",
    "body_plain": "Thanks for the email! Here's my reply.",
})

print(f"Reply sent! ID: {reply['id']}")
```

---

## 6. Forward an Email

```python
forward = api("POST", f"/api/v1/emails/{original_email_id}/forward", json={
    "account_id": ACCOUNT_ID,
    "to": [
        {"identifier": "teammate@example.com", "identifier_type": "EMAIL_ADDRESS"}
    ],
    "body_html": "<p>FYI — see below.</p>",
})
```

---

## 7. Register a Webhook (Get Notified of New Emails)

Instead of polling, tell Ondapile to POST to your server when things happen:

```python
webhook = api("POST", "/api/v1/webhooks", json={
    "url": "https://your-app.com/webhook/ondapile",
    "events": ["email.received", "email.sent", "account.updated"],
})

print(f"Webhook ID: {webhook['id']}")
print(f"Secret:     {webhook['secret']}")  # Use this to verify HMAC signatures
```

### Handle incoming webhooks in your app

```python
from flask import Flask, request, jsonify
import hmac
import hashlib

app = Flask(__name__)
WEBHOOK_SECRET = "your-webhook-secret"

@app.post("/webhook/ondapile")
def handle_webhook():
    # Verify HMAC signature
    payload = request.get_data()
    signature = request.headers.get("X-Signature")
    expected = hmac.new(
        WEBHOOK_SECRET.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()

    if not hmac.compare_digest(signature or "", expected):
        return jsonify({"error": "invalid signature"}), 401

    event = request.json
    event_type = event.get("event")

    if event_type == "email.received":
        email = event["data"]
        print(f"New email from {email['from_attendee']['identifier']}")
        print(f"Subject: {email['subject']}")

        # Your logic here — maybe call an LLM to draft a reply
        handle_incoming_email(email)

    return jsonify({"ok": True})
```

---

## 8. Build an Auto-Responder (Full Example)

```python
"""
auto_responder.py — Reads new emails and sends AI-generated replies via Ondapile.
"""

import requests
import time

ONDAPILE_URL = "http://localhost:8080"
API_KEY = "your-api-key"
ACCOUNT_ID = "acc_your_account_id"
HEADERS = {"X-API-Key": API_KEY, "Content-Type": "application/json"}

SEEN_IDS = set()


def get_emails():
    """Fetch all inbox emails."""
    resp = requests.get(
        f"{ONDAPILE_URL}/api/v1/emails",
        headers=HEADERS,
        params={"account_id": ACCOUNT_ID},
    )
    resp.raise_for_status()
    return resp.json()["items"]


def send_reply(email_id, body):
    """Reply to an email."""
    resp = requests.post(
        f"{ONDAPILE_URL}/api/v1/emails/{email_id}/reply",
        headers=HEADERS,
        json={
            "account_id": ACCOUNT_ID,
            "body_html": f"<p>{body}</p>",
            "body_plain": body,
        },
    )
    resp.raise_for_status()
    return resp.json()


def generate_reply(subject, body_plain):
    """Your AI/LLM logic goes here. Stub for now."""
    return f"Thanks for your email about '{subject}'. I'll get back to you shortly."


def poll_and_respond():
    """Main loop: check for new emails, reply to unseen ones."""
    print("Auto-responder started. Polling every 30s...")

    while True:
        emails = get_emails()

        for em in emails:
            if em["id"] in SEEN_IDS:
                continue
            SEEN_IDS.add(em["id"])

            sender = em["from_attendee"]["identifier"]
            subject = em["subject"]
            body = em.get("body_plain", "")

            print(f"New email from {sender}: {subject}")

            # Generate and send reply
            reply_text = generate_reply(subject, body)
            result = send_reply(em["id"], reply_text)
            print(f"  → Replied: {result['id']}")

        time.sleep(30)


if __name__ == "__main__":
    poll_and_respond()
```

---

## 9. Webhook-Driven Version (No Polling)

```python
"""
webhook_responder.py — Webhook-driven auto-responder. No polling needed.

1. Start this Flask server
2. Register webhook: POST /api/v1/webhooks with url pointing here
3. Ondapile pushes new emails to you in real-time
"""

from flask import Flask, request, jsonify
import requests

app = Flask(__name__)

ONDAPILE_URL = "http://localhost:8080"
API_KEY = "your-api-key"
HEADERS = {"X-API-Key": API_KEY, "Content-Type": "application/json"}


def generate_reply(subject, body):
    """Your AI/LLM logic here."""
    return f"Thanks for your message about '{subject}'. I'll review and respond soon."


@app.post("/webhook")
def handle():
    event = request.json
    event_type = event.get("event")

    if event_type == "email.received":
        email = event["data"]
        account_id = event["account_id"]
        sender = email["from_attendee"]["identifier"]
        subject = email["subject"]
        body = email.get("body_plain", "")

        print(f"Incoming: {sender} — {subject}")

        reply_text = generate_reply(subject, body)

        # Send reply through Ondapile
        requests.post(
            f"{ONDAPILE_URL}/api/v1/emails/{email['id']}/reply",
            headers=HEADERS,
            json={
                "account_id": account_id,
                "body_html": f"<p>{reply_text}</p>",
                "body_plain": reply_text,
            },
        )
        print(f"  → Replied to {sender}")

    return jsonify({"ok": True})


if __name__ == "__main__":
    app.run(port=5000)
```

---

## API Quick Reference

| Action | Method | Endpoint | Key Params |
|--------|--------|----------|------------|
| Health check | `GET` | `/health` | — |
| List accounts | `GET` | `/api/v1/accounts` | — |
| Connect account | `POST` | `/api/v1/accounts` | `provider`, `identifier`, `credentials` |
| Delete account | `DELETE` | `/api/v1/accounts/:id` | — |
| Reconnect account | `POST` | `/api/v1/accounts/:id/reconnect` | — |
| List emails | `GET` | `/api/v1/emails` | `account_id` (required) |
| Get email | `GET` | `/api/v1/emails/:id` | — |
| Send email | `POST` | `/api/v1/emails` | `account_id`, `to`, `subject`, `body_html` |
| Reply to email | `POST` | `/api/v1/emails/:id/reply` | `account_id`, `body_html` |
| Forward email | `POST` | `/api/v1/emails/:id/forward` | `account_id`, `to`, `body_html` |
| List folders | `GET` | `/api/v1/emails/folders` | `account_id` (required) |
| Download attachment | `GET` | `/api/v1/emails/:id/attachments/:att_id` | — |
| List webhooks | `GET` | `/api/v1/webhooks` | — |
| Create webhook | `POST` | `/api/v1/webhooks` | `url`, `events` |
| Delete webhook | `DELETE` | `/api/v1/webhooks/:id` | — |
| List chats | `GET` | `/api/v1/chats` | — |
| Send message | `POST` | `/api/v1/chats/:id/messages` | `text` |
| Search | `POST` | `/api/v1/search` | `query` |
| Metrics | `GET` | `/metrics` | — |

## Auth

Every request needs one of:
```
X-API-Key: your-api-key
# or
Authorization: Bearer your-api-key
```

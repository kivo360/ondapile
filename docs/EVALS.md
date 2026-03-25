# Ondapile v1 Evaluation Suite

> Automated and manual tests to verify the platform works end-to-end.
> Run these after any deployment or significant code change.
>
> Created: 2026-03-25

---

## Quick Smoke Test (30 seconds)

```bash
# Server is running?
curl -s http://localhost:8080/health | jq .status
# Expected: "ok"

# Auth works?
curl -s http://localhost:8080/api/v1/accounts -H "X-API-Key: $ONDAPILE_API_KEY" | jq '.items | length'
# Expected: number (0+)

# Bad auth rejected?
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/accounts -H "X-API-Key: wrong-key"
# Expected: 401

# Go build clean?
go build ./...
# Expected: exit 0, no output

# All tests pass?
go test ./tests/integration/ -count=1 -timeout 120s
# Expected: ok, PASS
```

---

## Eval 1: Platform Operations

### E1.1 Health & Metrics
```bash
# Health endpoint
curl -s http://localhost:8080/health | jq .
# ✅ PASS: { "status": "ok" }

# Metrics endpoint
curl -s http://localhost:8080/metrics | jq 'keys'
# ✅ PASS: Returns JSON with db_pool, goroutines, uptime, etc.
```

### E1.2 Authentication
```bash
# API key in X-API-Key header
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/accounts \
  -H "X-API-Key: $ONDAPILE_API_KEY"
# ✅ PASS: 200

# API key as Bearer token
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/accounts \
  -H "Authorization: Bearer $ONDAPILE_API_KEY"
# ✅ PASS: 200

# No API key
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/accounts
# ✅ PASS: 401

# Wrong API key
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/accounts \
  -H "X-API-Key: totally-wrong"
# ✅ PASS: 401
```

### E1.3 Rate Limiting
```bash
# Rapid-fire 200 requests — should see 429s after burst limit
for i in $(seq 1 200); do
  CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/accounts \
    -H "X-API-Key: $ONDAPILE_API_KEY")
  if [ "$CODE" = "429" ]; then echo "Rate limited at request $i"; break; fi
done
# ✅ PASS: Rate limited before request 200 (burst limit: 100)
```

---

## Eval 2: Account Management

### E2.1 Connect IMAP Account
```bash
curl -s -X POST http://localhost:8080/api/v1/accounts \
  -H "X-API-Key: $ONDAPILE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "IMAP",
    "identifier": "test@example.com",
    "name": "Test IMAP",
    "credentials": {
      "imap_host": "imap.gmail.com",
      "imap_port": "993",
      "smtp_host": "smtp.gmail.com",
      "smtp_port": "587",
      "username": "test@example.com",
      "password": "app-password-here"
    }
  }' | jq '{id, provider, status}'
# ✅ PASS: { "id": "acc_...", "provider": "IMAP", "status": "CONNECTING" or "OPERATIONAL" }
```

### E2.2 List Accounts
```bash
curl -s http://localhost:8080/api/v1/accounts \
  -H "X-API-Key: $ONDAPILE_API_KEY" | jq '.items | length'
# ✅ PASS: >= 1
```

### E2.3 Get Account Detail
```bash
ACCOUNT_ID="acc_your_account_id"
curl -s http://localhost:8080/api/v1/accounts/$ACCOUNT_ID \
  -H "X-API-Key: $ONDAPILE_API_KEY" | jq '{id, provider, status, capabilities}'
# ✅ PASS: Returns account with all fields populated
```

### E2.4 Delete Account
```bash
curl -s -X DELETE http://localhost:8080/api/v1/accounts/$ACCOUNT_ID \
  -H "X-API-Key: $ONDAPILE_API_KEY" | jq .
# ✅ PASS: { "deleted": true } or similar

# Verify deleted
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/accounts/$ACCOUNT_ID \
  -H "X-API-Key: $ONDAPILE_API_KEY"
# ✅ PASS: 404
```

---

## Eval 3: Email Operations (14 Required)

> Requires a connected email account with `account_id` set.

```bash
export ACCOUNT_ID="acc_your_connected_account"
export API="http://localhost:8080/api/v1"
export AUTH="-H 'X-API-Key: $ONDAPILE_API_KEY'"
```

### E3.1 List Emails
```bash
curl -s "$API/emails?account_id=$ACCOUNT_ID" $AUTH | jq '.items | length'
# ✅ PASS: >= 0 (number of emails in inbox)
```

### E3.2 Get Single Email
```bash
EMAIL_ID=$(curl -s "$API/emails?account_id=$ACCOUNT_ID&limit=1" $AUTH | jq -r '.items[0].id')
curl -s "$API/emails/$EMAIL_ID?account_id=$ACCOUNT_ID" $AUTH | jq '{id, subject, from_attendee, body}'
# ✅ PASS: Returns full email with subject, from, body (HTML + plain)
```

### E3.3 Send Email
```bash
curl -s -X POST "$API/emails" $AUTH \
  -H "Content-Type: application/json" \
  -d "{
    \"account_id\": \"$ACCOUNT_ID\",
    \"to\": [{\"identifier\": \"recipient@example.com\", \"identifier_type\": \"EMAIL_ADDRESS\"}],
    \"subject\": \"Ondapile Eval Test $(date +%s)\",
    \"body_html\": \"<p>Sent from eval suite</p>\"
  }" | jq '{id, subject}'
# ✅ PASS: Returns sent email object with ID
```

### E3.4 Reply to Email
```bash
curl -s -X POST "$API/emails/$EMAIL_ID/reply" $AUTH \
  -H "Content-Type: application/json" \
  -d "{
    \"account_id\": \"$ACCOUNT_ID\",
    \"body_html\": \"<p>Eval reply</p>\"
  }" | jq '{id, subject}'
# ✅ PASS: Returns reply with subject starting with "Re:"
```

### E3.5 Forward Email
```bash
curl -s -X POST "$API/emails/$EMAIL_ID/forward" $AUTH \
  -H "Content-Type: application/json" \
  -d "{
    \"account_id\": \"$ACCOUNT_ID\",
    \"to\": [{\"identifier\": \"forward@example.com\", \"identifier_type\": \"EMAIL_ADDRESS\"}],
    \"body_html\": \"<p>FYI</p>\"
  }" | jq '{id, subject}'
# ✅ PASS: Returns forwarded email with subject starting with "Fwd:"
```

### E3.6 Mark Read
```bash
curl -s -X PUT "$API/emails/$EMAIL_ID" $AUTH \
  -H "Content-Type: application/json" \
  -d '{"read": true}' | jq .read
# ✅ PASS: true
```

### E3.7 Star/Flag
```bash
curl -s -X PUT "$API/emails/$EMAIL_ID" $AUTH \
  -H "Content-Type: application/json" \
  -d '{"starred": true}' | jq .
# ✅ PASS: 200 response (starred field may not be in response yet)
```

### E3.8 Move to Folder
```bash
curl -s -X PUT "$API/emails/$EMAIL_ID" $AUTH \
  -H "Content-Type: application/json" \
  -d '{"folder": "ARCHIVE"}' | jq .
# ✅ PASS: 200 response
```

### E3.9 List Folders
```bash
curl -s "$API/emails/folders?account_id=$ACCOUNT_ID" $AUTH | jq '.[].name'
# ✅ PASS: Returns array of folder names (INBOX, SENT, DRAFTS, etc.)
```

### E3.10 Delete Email
```bash
curl -s -X DELETE "$API/emails/$EMAIL_ID?account_id=$ACCOUNT_ID" $AUTH | jq .
# ✅ PASS: { "deleted": true }
```

### E3.11 Search
```bash
curl -s "$API/emails?account_id=$ACCOUNT_ID&q=eval" $AUTH | jq '.items | length'
# ✅ PASS: >= 0 (returns matching emails)
```

### E3.12 Download Attachment
```bash
# Find an email with attachments
ATT_EMAIL=$(curl -s "$API/emails?account_id=$ACCOUNT_ID" $AUTH | jq -r '[.items[] | select(.has_attachments==true)][0].id')
ATT_ID=$(curl -s "$API/emails/$ATT_EMAIL?account_id=$ACCOUNT_ID" $AUTH | jq -r '.attachments[0].id')
curl -s -o /tmp/attachment "$API/emails/$ATT_EMAIL/attachments/$ATT_ID?account_id=$ACCOUNT_ID" $AUTH
ls -la /tmp/attachment
# ✅ PASS: File downloaded, size > 0
```

---

## Eval 4: Webhooks

### E4.1 Create Webhook
```bash
curl -s -X POST "$API/webhooks" $AUTH \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://httpbin.org/post",
    "events": ["email.sent", "email.received", "email.opened"]
  }' | jq '{id, url, events, secret}'
# ✅ PASS: Returns webhook with ID, URL, events array, and secret (whsec_ prefix)
```

### E4.2 List Webhooks
```bash
curl -s "$API/webhooks" $AUTH | jq '.items | length'
# ✅ PASS: >= 1
```

### E4.3 Delete Webhook
```bash
WH_ID=$(curl -s "$API/webhooks" $AUTH | jq -r '.items[0].id')
curl -s -X DELETE "$API/webhooks/$WH_ID" $AUTH | jq .
# ✅ PASS: Deleted
```

### E4.4 Webhook Fires on Email Send
```bash
# 1. Start a webhook receiver
python3 -c "
from http.server import HTTPServer, BaseHTTPRequestHandler
import json
class H(BaseHTTPRequestHandler):
    def do_POST(self):
        data = self.rfile.read(int(self.headers['Content-Length']))
        print(json.loads(data))
        self.send_response(200)
        self.end_headers()
HTTPServer(('',9999), H).handle_request()
" &
RECEIVER_PID=$!

# 2. Create webhook pointing to local receiver
curl -s -X POST "$API/webhooks" $AUTH \
  -H "Content-Type: application/json" \
  -d '{"url": "http://localhost:9999", "events": ["email.sent"]}'

# 3. Send an email
curl -s -X POST "$API/emails" $AUTH \
  -H "Content-Type: application/json" \
  -d "{\"account_id\": \"$ACCOUNT_ID\", \"to\": [{\"identifier\": \"test@test.com\", \"identifier_type\": \"EMAIL_ADDRESS\"}], \"subject\": \"Webhook Test\", \"body_html\": \"<p>Test</p>\"}"

# 4. Wait for webhook
sleep 2
kill $RECEIVER_PID 2>/dev/null
# ✅ PASS: Python server printed webhook payload with event "email.sent"
```

---

## Eval 5: Email Tracking

### E5.1 Tracking Pixel Returns GIF
```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/t/any-email-id
# ✅ PASS: 200

curl -s -D - http://localhost:8080/t/any-email-id | head -5
# ✅ PASS: Content-Type: image/gif
```

### E5.2 Link Redirect Works
```bash
curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/l/any-email-id?url=https://example.com"
# ✅ PASS: 302

curl -s -D - "http://localhost:8080/l/any-email-id?url=https://example.com" | grep Location
# ✅ PASS: Location: https://example.com
```

### E5.3 Link Redirect Missing URL
```bash
curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/l/any-email-id"
# ✅ PASS: 400
```

---

## Eval 6: Public Routes (No Auth Required)

```bash
# Health — no auth
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health
# ✅ PASS: 200

# Metrics — no auth
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/metrics
# ✅ PASS: 200

# OAuth success page — no auth
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/oauth/success
# ✅ PASS: 200

# Tracking pixel — no auth
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/t/test
# ✅ PASS: 200

# Link redirect — no auth
curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/l/test?url=https://example.com"
# ✅ PASS: 302

# API without auth — rejected
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/accounts
# ✅ PASS: 401
```

---

## Eval 7: Integration Test Suite

```bash
# Run all Go integration tests
go test ./tests/integration/ -count=1 -timeout 120s -v 2>&1 | tail -5
# ✅ PASS: "ok ondapile/tests/integration"

# Count passing tests
go test ./tests/integration/ -count=1 -timeout 120s -v 2>&1 | grep -c "^--- PASS"
# ✅ PASS: 200+ tests pass

# Run Node SDK tests
cd sdk/node && npm test 2>&1 | tail -3
# ✅ PASS: "27 tests passed"
```

---

## Eval 8: Build & Compilation

```bash
# Go build (all packages)
go build ./...
# ✅ PASS: exit 0, no output

# Go vet
go vet ./...
# ✅ PASS: exit 0, no output

# Node SDK type check
cd sdk/node && npx tsc --noEmit
# ✅ PASS: exit 0

# Docker build (if Dockerfile exists)
docker build -t ondapile:eval .
# ✅ PASS: Image builds successfully
```

---

## Eval 9: Database Schema

```bash
# All expected tables exist
psql ondapile -c "
SELECT table_name FROM information_schema.tables 
WHERE table_schema = 'public' 
ORDER BY table_name;
" | grep -E "accounts|chats|messages|emails|webhooks|webhook_deliveries|attendees|oauth_tokens|audit_log|calendars|calendar_events"
# ✅ PASS: All 11 tables listed

# Email table has tracking column
psql ondapile -c "\d emails" | grep tracking
# ✅ PASS: tracking | jsonb

# Organization scoping exists
psql ondapile -c "\d accounts" | grep organization_id
# ✅ PASS: organization_id | text

# Indexes exist
psql ondapile -c "SELECT indexname FROM pg_indexes WHERE tablename = 'emails' ORDER BY indexname;"
# ✅ PASS: idx_emails_account, idx_emails_date, idx_emails_folder, idx_emails_unread
```

---

## Eval 10: Encryption

```bash
# Credentials are encrypted (not plaintext)
psql ondapile -c "SELECT id, length(credentials_enc) as enc_length FROM accounts WHERE credentials_enc IS NOT NULL LIMIT 3;"
# ✅ PASS: enc_length > 0 for accounts with credentials

# Credentials are NOT stored as readable text
psql ondapile -c "SELECT credentials_enc::text FROM accounts WHERE credentials_enc IS NOT NULL LIMIT 1;"
# ✅ PASS: Output is binary garbage, not readable JSON
```

---

## Eval 11: Audit Log

```bash
# Audit log has entries
curl -s "$API/audit-log" $AUTH | jq '.items | length'
# ✅ PASS: >= 0

# Audit log entry has required fields
curl -s "$API/audit-log" $AUTH | jq '.items[0] | keys'
# ✅ PASS: Has action, actor_id, created_at, resource_type, etc.
```

---

## Eval Summary Checklist

| # | Category | Evals | Status |
|---|----------|-------|--------|
| 1 | Platform (health, auth, rate limit) | 4 | Run manually |
| 2 | Account management (CRUD) | 4 | Run manually |
| 3 | Email operations (14 required) | 12 | Run manually |
| 4 | Webhooks (CRUD + delivery) | 4 | Run manually |
| 5 | Email tracking (pixel, link) | 3 | Run manually |
| 6 | Public routes (no auth) | 6 | Run manually |
| 7 | Integration tests (Go + Node) | 200+ automated | `go test`, `npm test` |
| 8 | Build & compilation | 4 | `go build`, `go vet` |
| 9 | Database schema | 4 | `psql` queries |
| 10 | Encryption | 2 | `psql` queries |
| 11 | Audit log | 2 | API calls |
| **Total** | | **245+** | |

---

## Running All Automated Evals

```bash
#!/bin/bash
# eval-all.sh — Run all automated evaluations

set -e

echo "=== Eval 7: Integration Tests ==="
go test ./tests/integration/ -count=1 -timeout 120s
echo "✅ Integration tests passed"

echo "=== Eval 7: SDK Tests ==="
cd sdk/node && npm test && cd ../..
echo "✅ SDK tests passed"

echo "=== Eval 8: Build ==="
go build ./...
echo "✅ Build passed"

echo "=== Eval 8: Vet ==="
go vet ./...
echo "✅ Vet passed"

echo "=== Eval 1: Health ==="
STATUS=$(curl -sf http://localhost:8080/health | jq -r .status)
[ "$STATUS" = "ok" ] && echo "✅ Health check passed" || echo "❌ Health check failed"

echo "=== Eval 1: Auth ==="
CODE=$(curl -sf -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/accounts -H "X-API-Key: $ONDAPILE_API_KEY")
[ "$CODE" = "200" ] && echo "✅ Auth passed" || echo "❌ Auth failed (got $CODE)"

echo "=== Eval 1: Auth Rejection ==="
CODE=$(curl -sf -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/accounts -H "X-API-Key: wrong")
[ "$CODE" = "401" ] && echo "✅ Auth rejection passed" || echo "❌ Auth rejection failed (got $CODE)"

echo "=== Eval 5: Tracking Pixel ==="
CODE=$(curl -sf -o /dev/null -w "%{http_code}" http://localhost:8080/t/eval-test)
[ "$CODE" = "200" ] && echo "✅ Tracking pixel passed" || echo "❌ Tracking pixel failed (got $CODE)"

echo "=== Eval 5: Link Redirect ==="
CODE=$(curl -sf -o /dev/null -w "%{http_code}" "http://localhost:8080/l/eval-test?url=https://example.com")
[ "$CODE" = "302" ] && echo "✅ Link redirect passed" || echo "❌ Link redirect failed (got $CODE)"

echo ""
echo "=== ALL AUTOMATED EVALS COMPLETE ==="
```

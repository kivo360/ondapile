# Ondapile Scenario Evals

> Multi-step, agent-executable evaluation scenarios. Each scenario is a story: a user does a sequence of things, and a sub-agent verifies the results using a mix of HTTP assertions, DB queries, and LLM judgment.
>
> These are NOT unit tests. They test **behavior across system boundaries** — frontend → auth → API → DB → webhook → tracking.
>
> A sub-agent reads a scenario, executes each step, collects evidence, and renders a verdict.
>
> Created: 2026-03-25

---

## How Sub-Agents Execute These

```
For each scenario:
  1. Read the SETUP section — create any prerequisite state
  2. Execute each STEP in order — make real HTTP calls, observe responses
  3. After all steps, run the VERIFY section — check assertions + fuzzy criteria
  4. Render VERDICT:
     - PASS: all assertions true + fuzzy criteria score >= 4/5
     - PARTIAL: assertions pass but fuzzy criteria score 2-3/5
     - FAIL: any assertion false OR fuzzy score < 2/5
  5. Write evidence to eval log
```

**Sub-agent prompt template:**

```
You are an eval agent. Execute the following scenario against a running ondapile instance at {BASE_URL}.

For each STEP:
- Execute the HTTP request or browser action described
- Record the response status, body, and headers
- Note any unexpected behavior

For VERIFY:
- Check each assertion (deterministic — true/false)
- Evaluate each fuzzy criterion (score 1-5 with reasoning)

Report format:
  SCENARIO: {name}
  STEPS: {n}/{n} executed
  ASSERTIONS: {passed}/{total}
  FUZZY SCORE: {avg}/5
  VERDICT: PASS | PARTIAL | FAIL
  EVIDENCE: [detailed log]
```

---

## S1: First-Time Publisher Onboarding

> A new publisher signs up, creates an org, gets an API key, connects an email account, sends their first email, and receives a webhook.

### Setup
```
- ondapile running at BASE_URL
- No pre-existing user for this test email
```

### Steps

```
STEP 1: Sign up
  POST {BASE_URL}/api/auth/sign-up/email
  Body: { name: "Eval Publisher", email: "s1-{timestamp}@eval.com", password: "S1Eval1234!" }
  
  ASSERT: status = 200
  ASSERT: response.user.id is non-empty string
  SAVE: user_id, set-cookie header as SESSION_COOKIE

STEP 2: Verify org was auto-created
  GET {BASE_URL}/api/auth/organization/list
  Headers: { Cookie: SESSION_COOKIE }
  
  ASSERT: status = 200
  ASSERT: response is array with length >= 1
  ASSERT: response[0].id is non-empty
  SAVE: org_id = response[0].id

STEP 3: Set active organization
  POST {BASE_URL}/api/auth/organization/set-active
  Headers: { Cookie: SESSION_COOKIE, Content-Type: application/json }
  Body: { organizationId: org_id }
  
  ASSERT: status = 200

STEP 4: Create API key
  POST {BASE_URL}/api/auth/api-key/create
  Headers: { Cookie: SESSION_COOKIE, Content-Type: application/json }
  Body: { name: "s1-eval-key", organizationId: org_id }
  
  ASSERT: status = 200
  ASSERT: response.key starts with "sk_live_"
  SAVE: api_key = response.key

STEP 5: Verify API key works with Go backend
  GET {BASE_URL_API}/api/v1/accounts
  Headers: { Authorization: "Bearer {api_key}" }
  
  ASSERT: status = 200
  ASSERT: response has "items" array

STEP 6: Create webhook
  POST {BASE_URL_API}/api/v1/webhooks
  Headers: { Authorization: "Bearer {api_key}", Content-Type: application/json }
  Body: { url: "https://httpbin.org/post", events: ["email.sent", "account.connected"] }
  
  ASSERT: status = 201
  ASSERT: response.id starts with "whk_"
  ASSERT: response.secret is non-empty
  SAVE: webhook_id, webhook_secret

STEP 7: Connect IMAP account
  POST {BASE_URL_API}/api/v1/accounts
  Headers: { Authorization: "Bearer {api_key}", Content-Type: application/json }
  Body: {
    provider: "IMAP",
    identifier: "s1-eval-{timestamp}@test.com",
    name: "Eval IMAP Account",
    credentials: {
      imap_host: "imap.gmail.com", imap_port: "993",
      smtp_host: "smtp.gmail.com", smtp_port: "587",
      username: "test@test.com", password: "test"
    }
  }
  
  ASSERT: status = 201
  ASSERT: response.id starts with "acc_"
  SAVE: account_id

STEP 8: List accounts (should include new account)
  GET {BASE_URL_API}/api/v1/accounts
  Headers: { Authorization: "Bearer {api_key}" }
  
  ASSERT: status = 200
  ASSERT: response.items contains account with id = account_id
```

### Verify

**Assertions (deterministic):**
- [ ] All 8 steps returned expected status codes
- [ ] API key has `sk_live_` prefix
- [ ] Account was created with correct provider and identifier
- [ ] Webhook was created with secret

**Fuzzy criteria (model-graded, 1-5 each):**
- **Flow coherence**: Did each step logically follow from the previous? Did the session persist correctly across steps?
- **Error quality**: If any step failed, was the error message clear and actionable?
- **Response shape**: Do all responses follow a consistent JSON structure? (`object`, `id`, `created_at` fields?)
- **Timing**: Did all steps complete within 2 seconds each? Any unexplained delays?

**Pass: All assertions true + fuzzy avg >= 4/5**

---

## S2: Email Lifecycle (Read → Reply → Track → Webhook)

> A developer reads an email, replies to it, the reply triggers tracking, and a webhook fires.

### Setup
```
- api_key from S1 or pre-created
- account_id with at least 1 email in inbox
- OR: seed an email directly in the database for testing
```

### Steps

```
STEP 1: List emails
  GET {BASE_URL_API}/api/v1/emails?account_id={account_id}
  Headers: { Authorization: "Bearer {api_key}" }
  
  ASSERT: status = 200
  ASSERT: response.items is array
  SAVE: email_id = response.items[0].id (if any exist)
  
  IF no emails exist:
    Insert test email via DB:
    INSERT INTO emails (id, account_id, provider, subject, body, body_plain, 
      from_attendee, to_attendees, date_sent, metadata)
    VALUES ('eml_eval_s2', account_id, 'IMAP', 'S2 Eval Test', 
      '<p>Hello from eval</p>', 'Hello from eval',
      '{"display_name":"Sender","identifier":"sender@test.com","identifier_type":"EMAIL_ADDRESS"}',
      '[{"display_name":"Recip","identifier":"recip@test.com","identifier_type":"EMAIL_ADDRESS"}]',
      NOW(), '{}')
    SAVE: email_id = 'eml_eval_s2'

STEP 2: Get single email
  GET {BASE_URL_API}/api/v1/emails/{email_id}?account_id={account_id}
  Headers: { Authorization: "Bearer {api_key}" }
  
  ASSERT: status = 200
  ASSERT: response.id = email_id
  ASSERT: response.subject is non-empty string
  ASSERT: response.body OR response.body_plain is non-empty

STEP 3: Reply to email
  POST {BASE_URL_API}/api/v1/emails/{email_id}/reply
  Headers: { Authorization: "Bearer {api_key}", Content-Type: application/json }
  Body: {
    account_id: account_id,
    body_html: "<p>This is an eval reply at {timestamp}</p>"
  }
  
  ASSERT: status = 200
  OBSERVE: Does the response contain a new email ID?
  OBSERVE: Does the subject start with "Re:"?

STEP 4: Mark email as read
  PUT {BASE_URL_API}/api/v1/emails/{email_id}
  Headers: { Authorization: "Bearer {api_key}", Content-Type: application/json }
  Body: { read: true }
  
  ASSERT: status = 200

STEP 5: Move email to archive
  PUT {BASE_URL_API}/api/v1/emails/{email_id}
  Headers: { Authorization: "Bearer {api_key}", Content-Type: application/json }
  Body: { folder: "ARCHIVE" }
  
  ASSERT: status = 200

STEP 6: Hit tracking pixel
  GET {BASE_URL}/t/{email_id}
  (no auth header — public endpoint)
  
  ASSERT: status = 200
  ASSERT: content-type = image/gif
  ASSERT: body length > 0

STEP 7: Hit tracking link
  GET {BASE_URL}/l/{email_id}?url=https://example.com/eval-s2
  (no auth header — public endpoint, don't follow redirect)
  
  ASSERT: status = 302
  ASSERT: location header = https://example.com/eval-s2

STEP 8: List folders
  GET {BASE_URL_API}/api/v1/emails/folders?account_id={account_id}
  Headers: { Authorization: "Bearer {api_key}" }
  
  ASSERT: status = 200
  OBSERVE: Does the response contain INBOX, SENT, ARCHIVE folders?

STEP 9: Delete email
  DELETE {BASE_URL_API}/api/v1/emails/{email_id}
  Headers: { Authorization: "Bearer {api_key}" }
  
  ASSERT: status = 200

STEP 10: Verify email is gone
  GET {BASE_URL_API}/api/v1/emails/{email_id}?account_id={account_id}
  Headers: { Authorization: "Bearer {api_key}" }
  
  OBSERVE: Does this return 404 or an empty result?
```

### Verify

**Assertions (deterministic):**
- [ ] Steps 1-9 returned expected status codes
- [ ] Tracking pixel served image/gif
- [ ] Link redirect returned 302 with correct location
- [ ] Email was deleted successfully

**Fuzzy criteria (model-graded, 1-5 each):**
- **Reply quality**: Did the reply response include threading info (subject with "Re:", provider_id with thread context)?
- **State transitions**: After marking read + moving to archive, does a subsequent GET reflect these changes?
- **Tracking persistence**: After hitting the pixel, does the email's tracking JSONB show `opens >= 1`?
- **Deletion completeness**: After DELETE, is the email truly gone from list queries? Or does it linger?
- **Error graceful**: If the IMAP account is disconnected, do operations fail gracefully with clear errors instead of 500s?

---

## S3: Multi-Tenant Isolation

> Two publishers sign up. Each creates accounts, emails, webhooks. Verify one can't see the other's data.

### Steps

```
STEP 1: Create Publisher A (signup → org → key)
  [Same flow as S1 steps 1-5 with email "s3-a-{ts}@eval.com"]
  SAVE: a_key, a_org_id

STEP 2: Create Publisher B (signup → org → key)
  [Same flow as S1 steps 1-5 with email "s3-b-{ts}@eval.com"]
  SAVE: b_key, b_org_id

STEP 3: Publisher A creates account
  POST /api/v1/accounts with a_key
  Body: { provider: "IMAP", identifier: "a-inbox@test.com", name: "A's inbox", credentials: {} }
  SAVE: a_account_id

STEP 4: Publisher B creates account
  POST /api/v1/accounts with b_key
  Body: { provider: "IMAP", identifier: "b-inbox@test.com", name: "B's inbox", credentials: {} }
  SAVE: b_account_id

STEP 5: Publisher A lists accounts — should NOT see B's account
  GET /api/v1/accounts with a_key
  ASSERT: response.items does NOT contain b_account_id
  ASSERT: all items have organization_id matching a_org_id (or null for legacy)

STEP 6: Publisher B lists accounts — should NOT see A's account
  GET /api/v1/accounts with b_key
  ASSERT: response.items does NOT contain a_account_id

STEP 7: Publisher A creates webhook
  POST /api/v1/webhooks with a_key
  SAVE: a_webhook_id

STEP 8: Publisher B lists webhooks — should NOT see A's webhook
  GET /api/v1/webhooks with b_key
  ASSERT: response.items does NOT contain a_webhook_id

STEP 9: Cross-access attempt — A tries to get B's account
  GET /api/v1/accounts/{b_account_id} with a_key
  ASSERT: status = 404 OR 403 (not 200)

STEP 10: Cross-access attempt — B tries to delete A's webhook
  DELETE /api/v1/webhooks/{a_webhook_id} with b_key
  ASSERT: status = 404 OR 403 (not 200)
```

### Verify

**Assertions:**
- [ ] Account lists are org-scoped (no cross-org leakage)
- [ ] Webhook lists are org-scoped
- [ ] Cross-org GET returns 404/403
- [ ] Cross-org DELETE returns 404/403

**Fuzzy criteria:**
- **Isolation completeness**: Are there ANY paths where org A's data leaks to org B? Check audit log, metrics, search endpoints.
- **Error response quality**: When cross-access is blocked, is the error message generic ("not found") rather than revealing ("belongs to another org")?
- **ID enumeration risk**: Can org B guess org A's account IDs and probe for their existence?

---

## S4: Webhook Delivery Reliability

> Create a webhook, trigger events, verify delivery with retries.

### Setup
```
- api_key with permissions
- A local webhook receiver (or httpbin.org/post for fire-and-forget testing)
```

### Steps

```
STEP 1: Start local webhook receiver
  (Sub-agent starts a simple HTTP server on localhost:9999 that logs requests)

STEP 2: Create webhook pointing to receiver
  POST /api/v1/webhooks
  Body: { url: "http://localhost:9999/webhook", events: ["email.sent", "email.opened"], secret: "eval-secret-s4" }
  SAVE: webhook_id

STEP 3: Trigger email.sent event (send an email)
  POST /api/v1/emails
  Body: { account_id: account_id, to: [...], subject: "S4 webhook test", body_html: "<p>Test</p>" }

STEP 4: Wait 3 seconds, check receiver log
  OBSERVE: Did the receiver get a POST with:
    - X-Ondapile-Signature header?
    - JSON body with event = "email.sent"?
    - account_id in the payload?

STEP 5: Verify HMAC signature
  Using webhook_secret "eval-secret-s4":
    computed = HMAC-SHA256(raw_body, "eval-secret-s4")
    ASSERT: X-Ondapile-Signature header matches "sha256={computed}"

STEP 6: Trigger email.opened event (hit tracking pixel)
  GET /t/{sent_email_id}
  
  Wait 3 seconds.
  OBSERVE: Did the receiver get a POST with event = "email.opened"?

STEP 7: Stop receiver, create failing webhook
  POST /api/v1/webhooks
  Body: { url: "http://localhost:9998/nonexistent", events: ["email.sent"] }
  (Port 9998 is not listening — delivery will fail)
  SAVE: failing_webhook_id

STEP 8: Trigger event for failing webhook
  POST /api/v1/emails (send another email)

STEP 9: Wait 30 seconds, check webhook_deliveries table
  SELECT * FROM webhook_deliveries WHERE webhook_id = failing_webhook_id
  OBSERVE: 
    - attempts > 0?
    - delivered = false?
    - next_retry is set to a future time?
```

### Verify

**Assertions:**
- [ ] Successful webhook has valid HMAC signature
- [ ] Payload contains correct event type and data
- [ ] Failed webhook has attempts > 0 and delivered = false
- [ ] Retry is scheduled with exponential backoff

**Fuzzy criteria:**
- **Delivery latency**: How quickly did the first delivery arrive after the event? Under 5 seconds?
- **Payload completeness**: Does the payload contain all expected fields (event, timestamp, account_id, data)?
- **Retry behavior**: After the first failure, is the next_retry approximately 10s later (matching the documented backoff)?
- **Idempotency hint**: Does the payload include any deduplication hint (event ID, timestamp) so consumers can handle retries?

---

## S5: Hosted Auth End-User Flow

> Simulates what an end user sees when connecting their email account through a Publisher's app.

### Steps

```
STEP 1: Publisher generates hosted auth link
  POST /api/v1/accounts/hosted-auth
  Headers: { Authorization: "Bearer {api_key}" }
  Body: { provider: "gmail", redirect_url: "https://myapp.com/connected" }
  
  ASSERT: status = 200
  ASSERT: response.url is a valid URL
  SAVE: auth_url

STEP 2: (Browser eval) Navigate to auth_url
  OBSERVE: 
    - Does the page load without errors?
    - Does it show a provider selection or directly redirect to Google?
    - Is the UI clean and functional?

STEP 3: Check OAuth success page
  GET /oauth/success
  ASSERT: status = 200
  ASSERT: body contains "Account Connected" or similar success message

STEP 4: Check WhatsApp QR page structure
  POST /api/v1/accounts
  Body: { provider: "WHATSAPP", identifier: "eval-wa", credentials: {} }
  SAVE: wa_account_id
  
  GET /wa/qr/{wa_account_id}
  ASSERT: status = 200
  OBSERVE: Does the page contain a QR code element or image?
```

### Verify

**Fuzzy criteria:**
- **UX quality**: Is the hosted auth page professional-looking? Does it load fast? Are there broken images or layout issues?
- **Error handling**: If the provider is misconfigured (no Google client ID), does the auth page show a clear error rather than a crash?
- **Redirect behavior**: After OAuth consent, does the browser properly redirect to the Publisher's redirect_url?
- **QR page usability**: Is the WhatsApp QR page scannable? Does it show clear instructions?

---

## S6: Dashboard Full Journey (Playwright)

> A sub-agent uses Playwright to walk through the entire dashboard as an Org Admin.

### Steps

```
STEP 1: Sign up via browser
  goto /auth/signup
  Fill name, email, password
  Click submit
  ASSERT: redirected to /dashboard

STEP 2: Navigate sidebar
  Click "Accounts" → ASSERT: /dashboard/accounts loads
  Click "API Keys" → ASSERT: /dashboard/api-keys loads
  Click "Webhooks" → ASSERT: /dashboard/webhooks loads
  Click "Logs" → ASSERT: /dashboard/logs loads
  Click "Settings" → ASSERT: /dashboard/settings loads

STEP 3: Create API key
  On /dashboard/api-keys, click "Create"
  Fill name: "playwright-eval-key"
  Click submit
  ASSERT: key with sk_live_ prefix appears
  OBSERVE: Is there a "copy to clipboard" button?

STEP 4: Create webhook
  On /dashboard/webhooks, click "Create"
  Fill URL: "https://httpbin.org/post"
  Select events
  Click submit
  ASSERT: webhook appears in list

STEP 5: Visit settings pages
  Navigate to /dashboard/settings/team
  ASSERT: current user visible as "owner"
  
  Navigate to /dashboard/settings/billing
  OBSERVE: Does it show plan info (even if mocked)?
  
  Navigate to /dashboard/settings/oauth
  OBSERVE: Does it show OAuth credential forms?

STEP 6: Check for console errors
  Collect all console.error messages across all pages
  ASSERT: no errors mentioning "undefined", "null reference", "hydration"
  ALLOW: errors about favicon, network requests to external services
```

### Verify

**Assertions:**
- [ ] All 10+ dashboard pages rendered without 500 errors
- [ ] API key creation showed the key value
- [ ] Webhook creation succeeded
- [ ] Team page showed current user

**Fuzzy criteria:**
- **Visual polish**: Are pages loading with proper layout, no broken spacing, no flash of unstyled content?
- **Responsiveness**: Do pages load within 2 seconds? Any spinners stuck?
- **Data freshness**: After creating an API key, does it appear in the list immediately without refresh?
- **Empty states**: For pages with no data (empty webhooks, empty accounts), is there a helpful empty state message?
- **Navigation consistency**: Does the sidebar highlight the current page? Does the browser URL update correctly?

---

## S7: Error Boundary Testing

> Deliberately break things to verify the app handles errors gracefully.

### Steps

```
STEP 1: Send malformed JSON
  POST /api/v1/emails
  Headers: { Authorization: "Bearer {api_key}", Content-Type: application/json }
  Body: "not valid json {{{{"
  
  ASSERT: status = 400 (not 500)
  OBSERVE: Is the error response valid JSON with {object:"error", code, message}?

STEP 2: Send request with missing required fields
  POST /api/v1/emails
  Headers: { Authorization: "Bearer {api_key}", Content-Type: application/json }
  Body: {}
  
  ASSERT: status = 422 or 400 (not 500)
  OBSERVE: Does the error identify WHICH field is missing?

STEP 3: Access nonexistent resource
  GET /api/v1/accounts/acc_this_does_not_exist
  Headers: { Authorization: "Bearer {api_key}" }
  
  ASSERT: status = 404 (not 500)

STEP 4: Use expired/revoked API key
  (Revoke the key first if possible, or use a random string)
  GET /api/v1/accounts
  Headers: { Authorization: "Bearer sk_live_revoked_key_here" }
  
  ASSERT: status = 401

STEP 5: Send request to nonexistent route
  GET /api/v1/this_route_does_not_exist
  Headers: { Authorization: "Bearer {api_key}" }
  
  ASSERT: status = 404

STEP 6: Send oversized payload
  POST /api/v1/emails
  Headers: { Authorization: "Bearer {api_key}", Content-Type: application/json }
  Body: { body_html: "A".repeat(10_000_000) }  // 10MB string
  
  OBSERVE: Does the server reject gracefully or time out?
  ASSERT: status != 500 (should be 413 or similar)

STEP 7: Rapid-fire requests (rate limit test)
  Send 200 GET /api/v1/accounts in a tight loop
  
  OBSERVE: At what request count does 429 appear?
  ASSERT: some requests return 429 before request 200
  OBSERVE: Does the 429 response include Retry-After or rate limit headers?
```

### Verify

**Assertions:**
- [ ] No step returned 500 (internal server error)
- [ ] All error responses are valid JSON
- [ ] Rate limiting activates within burst limit

**Fuzzy criteria:**
- **Error message quality**: Are error messages helpful to a developer? Do they say WHAT went wrong and HOW to fix it?
- **Consistency**: Do all errors follow the same shape `{object:"error", status, code, message}`?
- **Information leakage**: Do error responses avoid exposing internal details (stack traces, file paths, SQL queries)?
- **Rate limit UX**: Does the rate limit response include helpful headers (X-RateLimit-Remaining, Retry-After)?

---

## S8: Complete AI Auto-Responder Flow

> End-to-end simulation of the user-flows.md "Complete Flow" example — Publisher builds an AI email responder.

### Steps

```
STEP 1: Publisher signs up and gets API key
  [S1 steps 1-5]
  SAVE: api_key

STEP 2: Register webhook for email.received
  POST /api/v1/webhooks
  Body: { url: "{local_receiver}/webhook", events: ["email.received", "email.sent"] }

STEP 3: Connect IMAP account
  POST /api/v1/accounts
  Body: { provider: "IMAP", ... }
  SAVE: account_id

STEP 4: Seed an incoming email (simulate email_received)
  INSERT email directly into database with account_id
  Trigger webhook dispatch manually or wait for poll

STEP 5: Verify email.received webhook fired
  Check local receiver log
  ASSERT: received event with type "email.received"
  SAVE: received_email_id from webhook payload

STEP 6: Simulate AI processing — read the email
  GET /api/v1/emails/{received_email_id}?account_id={account_id}
  ASSERT: status = 200
  ASSERT: response has subject and body_plain

STEP 7: Send reply (simulating AI response)
  POST /api/v1/emails/{received_email_id}/reply
  Body: {
    account_id: account_id,
    body_html: "<p>Thank you for your email. This is an automated response.</p>"
  }
  ASSERT: status = 200

STEP 8: Verify email.sent webhook fired
  Check local receiver log
  ASSERT: received event with type "email.sent"

STEP 9: Mark original email as read
  PUT /api/v1/emails/{received_email_id}
  Body: { read: true }
  ASSERT: status = 200

STEP 10: Move to "Handled" folder
  PUT /api/v1/emails/{received_email_id}
  Body: { folder: "ARCHIVE" }
  ASSERT: status = 200
```

### Verify

**Assertions:**
- [ ] Full cycle: receive → read → reply → webhook → mark read → archive
- [ ] Both webhooks fired (email.received + email.sent)
- [ ] Reply preserved threading (Re: subject)

**Fuzzy criteria:**
- **End-to-end coherence**: Does the entire flow work without manual intervention? Could this run as a cron job?
- **Webhook reliability**: Did both webhooks arrive within 5 seconds?
- **Reply quality**: Does the reply look like it came from the original account (not a generic "noreply")?
- **State correctness**: After the full cycle, is the email marked read AND in the ARCHIVE folder?

---

## Scenario Registry

| ID | Name | Steps | Assertions | Fuzzy Criteria | Estimated Time |
|----|------|-------|------------|----------------|----------------|
| S1 | First-Time Onboarding | 8 | 8 | 4 | 30s |
| S2 | Email Lifecycle | 10 | 8 | 5 | 45s |
| S3 | Multi-Tenant Isolation | 10 | 6 | 3 | 30s |
| S4 | Webhook Delivery | 9 | 4 | 4 | 60s |
| S5 | Hosted Auth Flow | 4 | 3 | 4 | 20s |
| S6 | Dashboard Journey | 6 | 5 | 5 | 45s |
| S7 | Error Boundaries | 7 | 5 | 4 | 30s |
| S8 | AI Auto-Responder | 10 | 6 | 4 | 60s |
| **Total** | | **64** | **45** | **33** | **~5 min** |

---

## Running Scenarios

```bash
# Run a single scenario
# (Sub-agent reads this doc, executes steps, writes report)
ondapile-eval run S1

# Run all scenarios
ondapile-eval run all

# Run scenarios with Playwright (S5, S6)
ondapile-eval run S5 --browser
ondapile-eval run S6 --browser
```

In practice, a sub-agent executes these by:
1. Reading this markdown file
2. Translating each STEP into an HTTP request or Playwright action
3. Checking ASSERTs deterministically
4. Scoring OBSERVE/fuzzy criteria via LLM judgment
5. Writing a structured eval report

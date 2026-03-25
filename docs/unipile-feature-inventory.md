# Unipile.com Feature Inventory & User Workflow Mapping

**Research Date:** March 25, 2026  
**Source:** https://unipile.com, https://developer.unipile.com, https://www.unipile.com/pricing-api/

---

## 1. FEATURES

### 1.1 Core API Platform Features

| Feature | Description |
|--------|-------------|
| **Unified API** | Single REST API connecting to 8+ providers with unified schema |
| **500+ Endpoints** | Comprehensive endpoints across messaging, email, calendar |
| **Real-time Webhooks** | Push notifications for events without polling |
| **Auth on Behalf** | OAuth/session-based authentication as actual user (not bot) |
| **Account Protection** | Built-in proxy, rate limiting, quota management |
| **SDK Support** | Node.js SDK, N8N integration, MCP support |

### 1.2 Messaging Features

| Feature | Providers | Notes |
|---------|-----------|-------|
| Send & Reply Messages | LinkedIn, WhatsApp, Instagram, Telegram | |
| List Messages/Chats/Attendees | All | |
| Sync History | All | Unlimited historical data |
| Send Files/Attachments | All | |
| Receive Files Attachments | All | |
| Send Voice Notes | LinkedIn | Platform-specific |
| Send Embed Video | LinkedIn | Platform-specific |
| List Reactions | All | |
| Read Receipts | All | |
| Start New Chat | All | |

### 1.3 Email Features

| Feature | Providers | Notes |
|---------|-----------|-------|
| Send Email | Gmail, Outlook, IMAP | |
| List Emails | Gmail, Outlook, IMAP | |
| Retrieve Email | Gmail, Outlook, IMAP | |
| Delete Email | Gmail, Outlook, IMAP | |
| Update Email (Put) | Gmail, Outlook, IMAP | |
| Retrieve Attachment | Gmail, Outlook, IMAP | |
| List Folders | Gmail, Outlook, IMAP | |
| Create Draft | Gmail, Outlook, IMAP | |
| List Email Contacts | Gmail, Outlook, IMAP | |
| Email Tracking | Gmail, Outlook, IMAP | Webhook-based open/click tracking |

### 1.4 Calendar Features

| Feature | Providers | Notes |
|---------|-----------|-------|
| List Calendars | Google Calendar, Outlook Calendar | |
| Retrieve Calendar | Google Calendar, Outlook Calendar | |
| List Events by Calendar | Google Calendar, Outlook Calendar | |
| Create Event | Google Calendar, Outlook Calendar | |
| Retrieve Event | Google Calendar, Outlook Calendar | |
| Edit Event | Google Calendar, Outlook Calendar | |
| Delete Event | Google Calendar, Outlook Calendar | |
| Calendar Webhooks | Google Calendar, Outlook Calendar | Real-time sync |

### 1.5 LinkedIn-Specific Features

| Feature | Description |
|---------|-------------|
| LinkedIn Recruiter Search | Advanced search with 30+ filters |
| Send Invitation | Connect with prospects |
| Cancel Invitation | Withdraw connection requests |
| Handle Received Invitation | Accept/decline |
| List All Invitations (Sent/Received) | Full invitation management |
| Send InMail | Direct messages to 2nd/3rd degree connections |
| Get InMail Credit Balance | Track remaining credits |
| Retrieve/Edit Own Profile | Profile management |
| List All Relations/Followers/Following | Network data |
| List All Posts, Comments, Reactions | Content engagement |
| Retrieve Company Profile | Company data |
| LinkedIn Job Posting | Create, edit, publish, close job postings |
| Get Job Applicants | List and retrieve applicants |
| Download Applicant Resume | PDF extraction |
| Endorse Profile Skill | Skill endorsements |
| Get Raw Data | Access native LinkedIn API data |

### 1.6 Data Enrichment Features

| Feature | Description |
|---------|-------------|
| Profile Enrichment | Name, title, company, experience, skills, location |
| Job Change Tracking | Monitor career transitions |
| Company Data | Enrich with company insights |
| LinkedIn Search Data | Roles, interests, experiences |
| Contact Information | Email, phone from enriched profiles |

### 1.7 Automation/Sequencing Features

| Feature | Description |
|---------|-------------|
| Multi-Channel Outreach Sequences | LinkedIn + Email + WhatsApp + Calendar |
| Automated Follow-ups | Timed sequence steps |
| LinkedIn Voice Notes | Audio messaging in sequences |
| Post Reactions | Automated engagement |
| WhatsApp Reminders | Channel-specific automation |
| 1-Click Meeting Creation | From any conversation context |

### 1.8 Admin/Dashboard Features

| Feature | Description |
|---------|-------------|
| Account Management | Link, reconnect, delete accounts |
| Status Monitoring | Operational, Auth Required, Interrupted, Paused states |
| API Key Management | Generate/revoke access tokens |
| Webhook Configuration | Create, delete webhooks |
| Testing Mode | Development sandbox |
| Event Logs | Webhook and API call logging |
| Daily Limits Analytics | Track usage per channel |

---

## 2. USER TYPES

### 2.1 Primary User Types (Actors)

| Actor | Description | Key Workflows |
|-------|-------------|----------------|
| **SaaS Publisher (API Consumer)** | Software companies integrating Unipile into their products | Connect accounts → Send/receive messages → Webhook handling |
| **Developer/Engineer** | Technical implementers building integrations | API integration → Auth setup → Testing → Deployment |
| **End User (Business User)** | Recruiter, Salesperson, Support agent using integrated app | Authenticate accounts → Send messages → View inbox |
| **Platform Admin** | Manages API keys, webhooks, billing | Dashboard configuration → Monitor accounts → Manage limits |

### 2.2 Industry-Specific User Types

| Industry | Use Case | User |
|----------|----------|------|
| **ATS (Applicant Tracking System)** | Recruiting | Recruiter, HR Manager, Sourcer |
| **CRM** | Sales | Sales Rep, Account Executive, SDR |
| **Outreach Software** | Sales Engagement | Sales Development Rep, Marketer |
| **AI Agent Publishers** | Automation | AI/ML Engineer, Bot Developer |
| **No-Code Builders** | Workflow Automation | Citizen Developer, Automation Admin |
| **iPaaS** | Integration | Integration Specialist |

---

## 3. WORKFLOWS

### 3.1 Account Connection Workflow

```
1. Developer generates Hosted Auth URL via API
2. User redirected to provider's auth screen (OAuth/QR/Session)
3. User grants permission
4. Unipile receives callback with account_id
5. Webhook sent to notify_url with account_id + user mapping
6. Developer stores account_id mapping
7. API calls execute on behalf of user
```

### 3.2 Messaging Workflow

```
1. List all chats: GET /api/v1/chats
2. Select conversation: GET /api/v1/chats/:chatId/messages
3. Send message: POST /api/v1/chats/:chatId/messages
4. Receive webhook: message.received event
5. Mark as read via API
```

### 3.3 Outreach Sequence Workflow

```
1. Search LinkedIn: POST /api/v1/linkedin/search
2. Enrich profiles: GET /api/v1/users/:id
3. Send invitation: POST /api/v1/users/invitations
4. Schedule follow-up: Calendar API create event
5. Send email via: POST /api/v1/emails
6. Send WhatsApp: POST /api/v1/chats/:id/messages
7. Detect acceptance via webhook
8. Continue sequence based on response
```

### 3.4 Calendar Integration Workflow

```
1. List calendars: GET /api/v1/calendars
2. List events: GET /api/v1/calendars/:id/events
3. Create event: POST /api/v1/calendars/:id/events
4. Update event: PATCH /api/v1/calendars/events/:id
5. Delete event: DELETE /api/v1/calendars/events/:id
6. Receive calendar webhook on changes
```

### 3.5 Reconnection Workflow

```
1. Webhook received: status = "CREDENTIALS"
2. Notify user to reconnect (in-app/email)
3. Generate reconnect link: type: "reconnect"
4. User re-authenticates
5. Webhook received: status = "RECONNECTED"
6. Normal operations resume
```

---

## 4. INTEGRATIONS

### 4.1 Supported Providers

| Category | Provider | Auth Method | Status |
|----------|----------|-------------|--------|
| **Social/Messaging** | LinkedIn | Session-based | ✅ Supported |
| | Instagram | Session-based | ✅ Supported |
| | WhatsApp | QR Code | ✅ Supported |
| | Telegram | Phone/Token | ✅ Supported |
| | X (Twitter) | - | 🚫 Not currently supported |
| | Messenger | - | 🚫 Not currently supported |
| **Email** | Gmail | OAuth 2.0 | ✅ Supported |
| | Outlook | OAuth 2.0 | ✅ Supported |
| | IMAP | Credentials | ✅ Supported |
| **Calendar** | Google Calendar | OAuth 2.0 | ✅ Supported |
| | Outlook Calendar | OAuth 2.0 | ✅ Supported |

### 4.2 Integration Methods

| Method | Description |
|--------|-------------|
| **Hosted Auth** | Pre-built white-label auth wizard |
| **Custom Auth** | Build your own auth flow |
| **OAuth 2.0** | Google, Microsoft identity |
| **Session Auth** | LinkedIn, Instagram cookies |
| **QR Code** | WhatsApp mobile pairing |
| **IMAP Credentials** | Email server login |

---

## 5. API STRUCTURE

### 5.1 API Categories & Key Endpoints

#### Accounts API
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/accounts | List all connected accounts |
| POST | /api/v1/accounts | Connect account (native auth) |
| GET | /api/v1/accounts/:id | Retrieve account details |
| POST | /api/v1/accounts/:id/reconnect | Reconnect expired account |
| DELETE | /api/v1/accounts/:id | Delete account |
| PATCH | /api/v1/accounts/:id | Update proxy settings |
| POST | /api/v1/hosted/accounts/link | Generate hosted auth URL |

#### Messaging API
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/chats | List all conversations |
| POST | /api/v1/chats | Start new chat |
| GET | /api/v1/chats/:id | Get chat details |
| GET | /api/v1/chats/:id/messages | List messages in chat |
| POST | /api/v1/chats/:id/messages | Send message |
| POST | /api/v1/chats/:id/reactions | Add reaction |
| DELETE | /api/v1/chats/:id | Delete chat |

#### Users/Profiles API
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/users/invitations/sent | List sent invitations |
| GET | /api/v1/users/invitations/received | List received invitations |
| POST | /api/v1/users/invitations/received | Handle invitation |
| GET | /api/v1/users/profile | Get own profile |
| PATCH | /api/v1/users/profile | Edit own profile |
| GET | /api/v1/users/relations | List connections |
| POST | /api/v1/users/invitations | Send invitation |
| GET | /api/v1/users/:id | Get user profile |

#### LinkedIn-Specific API
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/linkedin/search | Search LinkedIn |
| GET | /api/v1/linkedin/hiring-projects | List recruiter projects |
| POST | /api/v1/linkedin/job-postings | Create job posting |
| GET | /api/v1/linkedin/job-postings | List job postings |
| GET | /api/v1/linkedin/job-postings/:id/applicants | List applicants |

#### Email API
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/emails | List all emails |
| POST | /api/v1/emails | Send email |
| GET | /api/v1/emails/:id | Get email details |
| DELETE | /api/v1/emails/:id | Delete email |
| GET | /api/v1/folders | List folders |
| POST | /api/v1/drafts | Create draft |

#### Calendar API
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/calendars | List calendars |
| GET | /api/v1/calendars/:id/events | List events |
| POST | /api/v1/calendars/:id/events | Create event |
| GET | /api/v1/calendars/events/:id | Get event |
| PATCH | /api/v1/calendars/events/:id | Update event |
| DELETE | /api/v1/calendars/events/:id | Delete event |

#### Webhooks API
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/webhooks | List webhooks |
| POST | /api/v1/webhooks | Create webhook |
| DELETE | /api/v1/webhooks/:id | Delete webhook |

### 5.2 Webhook Event Types

| Event | Trigger |
|-------|---------|
| account.status | Account status changes |
| message.received | New incoming message |
| message.sent | Message sent confirmation |
| email.received | New email |
| email.sent | Email sent confirmation |
| email.opened | Email opened (tracking) |
| email.clicked | Link clicked (tracking) |
| invitation.accepted | Connection request accepted |
| calendar.event.created | New calendar event |
| calendar.event.updated | Event modified |
| calendar.event.deleted | Event deleted |

### 5.3 Authentication Model

| Component | Description |
|-----------|-------------|
| **DSN (Data Source Name)** | Unique identifier for your API instance |
| **API Key (X-API-KEY)** | Authentication header for all requests |
| **Access Tokens** | Generated via dashboard or API |
| **account_id** | User mapping identifier after auth |
| **Webhook Auth** | Optional secret key header for verification |

---

## 6. PRICING MODEL

### 6.1 Pricing Structure

| Tier | Accounts | Price per Account/month |
|------|----------|------------------------|
| 1 | Up to 10 | €5.00 |
| 2 | 11-50 | €5.00 |
| 3 | 51-200 | (contact sales) |
| 4 | 201-1,000 | (contact sales) |
| 5 | 1,001-5,000 | (contact sales) |
| 6 | 5,000+ | (contact sales) |

### 6.2 Pricing Notes

- **Billing Unit**: Connected accounts (not messages)
- **Example**: 3 Emails + 2 LinkedIn + 6 WhatsApp = 11 accounts × €5 = €55/month
- **No usage fees**: Unlimited API calls within account limits
- **Minimum**: €49/month (up to 10 accounts)
- **Trial**: 7-day free trial, no credit card required

### 6.3 Account Definition

- Gmail + Calendar = 1 billed account
- Outlook + Calendar = 1 billed account
- LinkedIn = 1 account
- WhatsApp = 1 account
- Instagram = 1 account
- Telegram = 1 account
- IMAP = 1 account

---

## 7. SECURITY & COMPLIANCE

| Feature | Status |
|---------|--------|
| SOC 2 Type II | ✅ Certified |
| GDPR | ✅ Compliant |
| Data Encryption | ✅ In transit and at rest |
| Account Isolation | ✅ Session sandboxing |
| Token Abstraction | ✅ Server-side credential management |
| Webhook Verification | ✅ Optional auth headers |

---

## 8. SUPPORT & RESOURCES

| Resource | Link |
|----------|------|
| API Dashboard | https://dashboard.unipile.com |
| API Reference | https://developer.unipile.com/reference |
| Documentation | https://developer.unipile.com/docs |
| Quickstart Video | 5-minute integration guide |
| Slack Community | 1,000+ developers |
| Live Chat | Real-time support |
| Book a Call | Founder meetings |
| Changelog | https://developer.unipile.com/changelog |

---

## 9. CUSTOMER TESTIMONIALS

**3,000+ Companies** use Unipile, including:
- Lemlist
- Reply.io
- Relevance AI
- Valley
- Recruit CRM
- Artisan

---

## 10. DOWNSTREAM PRD ACTORS (Recommended)

Based on the user types identified, these are the actors to define in your PRD:

| Actor | Role | Primary Workflows |
|-------|------|------------------|
| **Platform Developer** | Integrates Unipile API into SaaS product | API setup, webhook handling, account management |
| **End User (Business)** | Uses integrated product for work | Authenticate accounts, send messages, view inbox |
| **Platform Admin** | Manages platform configuration | API keys, webhooks, monitoring |
| **AI Agent Operator** | Builds/triggers AI automations | Search, sequences, data enrichment |
| **Integration Manager** | Manages external tool connections | N8N, Make, Zapier workflows |

---

*Document generated from comprehensive scraping of unipile.com on March 25, 2026*

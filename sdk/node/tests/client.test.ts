import { describe, it, expect, vi, beforeEach, type Mock } from "vitest";
import { OndapileClient, OndapileError } from "../src/index.js";
import type {
  Account,
  Chat,
  Message,
  Email,
  CalendarEvent,
  Webhook,
  Attendee,
  PaginatedList,
} from "../src/types.js";

describe("Ondapile SDK", () => {
  const apiKey = "test-api-key";
  const baseUrl = "http://localhost:8080";

  let fetchMock: Mock;

  beforeEach(() => {
    fetchMock = vi.fn();
    global.fetch = fetchMock;
  });

  describe("BaseHttpClient", () => {
    it("should include X-API-KEY header on every request", async () => {
      const mockResponse: PaginatedList<Chat> = {
        object: "list",
        items: [],
        cursor: null,
        has_more: false,
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      await client.chats.list();

      expect(fetchMock).toHaveBeenCalledWith(
        `${baseUrl}/api/v1/chats`,
        expect.objectContaining({
          headers: expect.objectContaining({
            "X-API-KEY": apiKey,
            "Content-Type": "application/json",
          }),
        })
      );
    });

    it("should build correct URL with query params", async () => {
      const mockResponse: PaginatedList<Chat> = {
        object: "list",
        items: [],
        cursor: "next-cursor",
        has_more: true,
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      await client.chats.list({ limit: 25, account_id: "acc_xxx" });

      expect(fetchMock).toHaveBeenCalledWith(
        `${baseUrl}/api/v1/chats?limit=25&account_id=acc_xxx`,
        expect.any(Object)
      );
    });

    it("should omit undefined query params", async () => {
      const mockResponse: PaginatedList<Chat> = {
        object: "list",
        items: [],
        cursor: null,
        has_more: false,
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      await client.chats.list({ limit: 25 });

      const url = fetchMock.mock.calls[0][0];
      expect(url).toBe(`${baseUrl}/api/v1/chats?limit=25`);
      expect(url).not.toContain("undefined");
    });

    it("should throw OndapileError on non-2xx responses", async () => {
      const errorResponse = {
        object: "error",
        status: 404,
        code: "NOT_FOUND",
        message: "Chat not found",
        details: null,
      };

      fetchMock.mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: () => Promise.resolve(errorResponse),
      });

      const client = new OndapileClient({ apiKey, baseUrl });

      await expect(client.chats.get("invalid-id")).rejects.toThrow();
    });

    it("should handle request timeout", async () => {
      fetchMock.mockImplementation(
        () =>
          new Promise((_, reject) =>
            setTimeout(() => reject(new Error("Timeout")), 100)
          )
      );

      const client = new OndapileClient({
        apiKey,
        baseUrl,
        timeout: 50,
      });

      await expect(client.chats.list()).rejects.toThrow();
    });
  });

  describe("AccountsClient", () => {
    it("should list accounts", async () => {
      const mockResponse: PaginatedList<Account> = {
        object: "list",
        items: [
          {
            object: "account",
            id: "acc_xxx",
            provider: "WHATSAPP",
            name: "Test Account",
            identifier: "1234567890",
            status: "OPERATIONAL",
            status_detail: null,
            capabilities: ["messaging"],
            created_at: "2025-03-01T10:00:00Z",
            last_synced_at: null,
            proxy: null,
            metadata: {},
          },
        ],
        cursor: null,
        has_more: false,
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      const result = await client.accounts.list();

      expect(result.items).toHaveLength(1);
      expect(result.items[0].provider).toBe("WHATSAPP");
    });

    it("should create an account", async () => {
      const mockAccount: Account = {
        object: "account",
        id: "acc_new",
        provider: "WHATSAPP",
        name: "New Account",
        identifier: "0987654321",
        status: "CONNECTING",
        status_detail: null,
        capabilities: [],
        created_at: "2025-03-19T10:00:00Z",
        last_synced_at: null,
        proxy: null,
        metadata: {},
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockAccount),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      const result = await client.accounts.create({
        provider: "WHATSAPP",
      });

      expect(result.id).toBe("acc_new");
      expect(fetchMock).toHaveBeenCalledWith(
        `${baseUrl}/api/v1/accounts`,
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ provider: "WHATSAPP" }),
        })
      );
    });

    it("should get auth challenge", async () => {
      const mockChallenge = {
        type: "QR_CODE",
        payload: "qr-data-here",
        expiry: 1711036800,
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockChallenge),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      const result = await client.accounts.getAuthChallenge("acc_xxx");

      expect(result.type).toBe("QR_CODE");
      expect(result.payload).toBe("qr-data-here");
    });

    it("should delete an account", async () => {
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({}),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      await client.accounts.delete("acc_xxx");

      expect(fetchMock).toHaveBeenCalledWith(
        `${baseUrl}/api/v1/accounts/acc_xxx`,
        expect.objectContaining({
          method: "DELETE",
        })
      );
    });
  });

  describe("ChatsClient", () => {
    it("should send a message with correct JSON body", async () => {
      const mockMessage: Message = {
        object: "message",
        id: "msg_xxx",
        chat_id: "chat_xxx",
        account_id: "acc_xxx",
        provider: "WHATSAPP",
        provider_id: "provider-msg-id",
        text: "Hello!",
        sender_id: "sender_xxx",
        is_sender: true,
        timestamp: "2025-03-19T10:00:00Z",
        attachments: [],
        reactions: [],
        quoted: null,
        seen: false,
        seen_by: {},
        delivered: false,
        edited: false,
        deleted: false,
        hidden: false,
        is_event: false,
        event_type: null,
        metadata: {},
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockMessage),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      const result = await client.chats.sendMessage("chat_xxx", {
        text: "Hello!",
      });

      expect(result.text).toBe("Hello!");
      expect(fetchMock).toHaveBeenCalledWith(
        `${baseUrl}/api/v1/chats/chat_xxx/messages`,
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ text: "Hello!" }),
        })
      );
    });

    it("should list chats with cursor pagination", async () => {
      const mockResponse: PaginatedList<Chat> = {
        object: "list",
        items: [
          {
            object: "chat",
            id: "chat_1",
            account_id: "acc_xxx",
            provider: "WHATSAPP",
            provider_id: "provider-chat-id",
            type: "ONE_TO_ONE",
            name: null,
            attendees: [],
            last_message: null,
            unread_count: 0,
            is_group: false,
            is_archived: false,
            created_at: "2025-03-19T10:00:00Z",
            updated_at: "2025-03-19T10:00:00Z",
            metadata: {},
          },
        ],
        cursor: "next-page-cursor",
        has_more: true,
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      const result = await client.chats.list({ cursor: "initial-cursor" });

      expect(result.has_more).toBe(true);
      expect(result.cursor).toBe("next-page-cursor");
    });

    it("should archive a chat", async () => {
      const mockChat: Chat = {
        object: "chat",
        id: "chat_xxx",
        account_id: "acc_xxx",
        provider: "WHATSAPP",
        provider_id: "provider-chat-id",
        type: "ONE_TO_ONE",
        name: null,
        attendees: [],
        last_message: null,
        unread_count: 0,
        is_group: false,
        is_archived: true,
        created_at: "2025-03-19T10:00:00Z",
        updated_at: "2025-03-19T10:00:00Z",
        metadata: {},
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockChat),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      const result = await client.chats.update("chat_xxx", { is_archived: true });

      expect(result.is_archived).toBe(true);
      expect(fetchMock).toHaveBeenCalledWith(
        `${baseUrl}/api/v1/chats/chat_xxx`,
        expect.objectContaining({
          method: "PATCH",
          body: JSON.stringify({ is_archived: true }),
        })
      );
    });
  });

  describe("MessagesClient", () => {
    it("should add a reaction to a message", async () => {
      const mockMessage: Message = {
        object: "message",
        id: "msg_xxx",
        chat_id: "chat_xxx",
        account_id: "acc_xxx",
        provider: "WHATSAPP",
        provider_id: "provider-msg-id",
        text: "Hello",
        sender_id: "sender_xxx",
        is_sender: false,
        timestamp: "2025-03-19T10:00:00Z",
        attachments: [],
        reactions: [{ value: "👍", sender_id: "me", is_sender: true }],
        quoted: null,
        seen: false,
        seen_by: {},
        delivered: false,
        edited: false,
        deleted: false,
        hidden: false,
        is_event: false,
        event_type: null,
        metadata: {},
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockMessage),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      const result = await client.messages.addReaction("msg_xxx", {
        reaction: "👍",
      });

      expect(result.reactions).toHaveLength(1);
      expect(result.reactions[0].value).toBe("👍");
    });

    it("should delete a message", async () => {
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({}),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      await client.messages.delete("msg_xxx");

      expect(fetchMock).toHaveBeenCalledWith(
        `${baseUrl}/api/v1/messages/msg_xxx`,
        expect.objectContaining({
          method: "DELETE",
        })
      );
    });
  });

  describe("EmailsClient", () => {
    it("should send an email with proper body", async () => {
      const mockEmail: Email = {
        object: "email",
        id: "eml_xxx",
        account_id: "acc_xxx",
        provider: "GMAIL",
        provider_id: { message_id: "msg-id", thread_id: "thread-id" },
        subject: "Test Subject",
        body: "<p>Hello</p>",
        body_plain: "Hello",
        from_attendee: {
          display_name: "Sender",
          identifier: "sender@example.com",
          identifier_type: "EMAIL_ADDRESS",
        },
        to_attendees: [
          {
            display_name: "Recipient",
            identifier: "recipient@example.com",
            identifier_type: "EMAIL_ADDRESS",
          },
        ],
        cc_attendees: [],
        bcc_attendees: [],
        reply_to_attendees: [],
        date: "2025-03-19T10:00:00Z",
        has_attachments: false,
        attachments: [],
        folders: ["SENT"],
        role: "SENT",
        read: true,
        read_date: "2025-03-19T10:00:00Z",
        is_complete: true,
        headers: [],
        tracking: { opens: 0, first_opened_at: null, clicks: 0, links_clicked: [] },
        metadata: {},
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockEmail),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      const result = await client.emails.send({
        account_id: "acc_xxx",
        to: [{ identifier: "recipient@example.com", display_name: "Recipient" }],
        subject: "Test Subject",
        body: "<p>Hello</p>",
      });

      expect(result.subject).toBe("Test Subject");
    });

    it("should list emails with filters", async () => {
      const mockResponse: PaginatedList<Email> = {
        object: "list",
        items: [],
        cursor: null,
        has_more: false,
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      await client.emails.list({
        account_id: "acc_xxx",
        folder: "INBOX",
        has_attachments: true,
      });

      const url = fetchMock.mock.calls[0][0];
      expect(url).toContain("account_id=acc_xxx");
      expect(url).toContain("folder=INBOX");
      expect(url).toContain("has_attachments=true");
    });

    it("should reply to an email", async () => {
      const mockEmail: Email = {
        object: "email",
        id: "eml_reply",
        account_id: "acc_xxx",
        provider: "GMAIL",
        provider_id: { message_id: "msg-id", thread_id: "thread-id" },
        subject: "Re: Test Subject",
        body: "<p>Reply content</p>",
        body_plain: "Reply content",
        from_attendee: {
          display_name: "Sender",
          identifier: "sender@example.com",
          identifier_type: "EMAIL_ADDRESS",
        },
        to_attendees: [
          {
            display_name: "Recipient",
            identifier: "recipient@example.com",
            identifier_type: "EMAIL_ADDRESS",
          },
        ],
        cc_attendees: [],
        bcc_attendees: [],
        reply_to_attendees: [],
        date: "2025-03-19T10:00:00Z",
        has_attachments: false,
        attachments: [],
        folders: ["SENT"],
        role: "SENT",
        read: true,
        read_date: "2025-03-19T10:00:00Z",
        is_complete: true,
        headers: [],
        tracking: { opens: 0, first_opened_at: null, clicks: 0, links_clicked: [] },
        metadata: {},
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockEmail),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      const result = await client.emails.reply("eml_xxx", {
        account_id: "acc_xxx",
        body_html: "<p>Reply content</p>",
      });

      expect(result.subject).toBe("Re: Test Subject");
      expect(fetchMock).toHaveBeenCalledWith(
        `${baseUrl}/api/v1/emails/eml_xxx/reply`,
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({
            account_id: "acc_xxx",
            body_html: "<p>Reply content</p>",
          }),
        })
      );
    });

    it("should forward an email", async () => {
      const mockEmail: Email = {
        object: "email",
        id: "eml_fwd",
        account_id: "acc_xxx",
        provider: "GMAIL",
        provider_id: { message_id: "msg-id", thread_id: "thread-id" },
        subject: "Fwd: Test Subject",
        body: "<p>Forwarded content</p>",
        body_plain: "Forwarded content",
        from_attendee: {
          display_name: "Sender",
          identifier: "sender@example.com",
          identifier_type: "EMAIL_ADDRESS",
        },
        to_attendees: [
          {
            display_name: "ForwardRecipient",
            identifier: "forward@example.com",
            identifier_type: "EMAIL_ADDRESS",
          },
        ],
        cc_attendees: [],
        bcc_attendees: [],
        reply_to_attendees: [],
        date: "2025-03-19T10:00:00Z",
        has_attachments: false,
        attachments: [],
        folders: ["SENT"],
        role: "SENT",
        read: true,
        read_date: "2025-03-19T10:00:00Z",
        is_complete: true,
        headers: [],
        tracking: { opens: 0, first_opened_at: null, clicks: 0, links_clicked: [] },
        metadata: {},
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockEmail),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      const result = await client.emails.forward("eml_xxx", {
        account_id: "acc_xxx",
        to: [{ identifier: "forward@example.com", display_name: "ForwardRecipient" }],
        body_html: "<p>Forwarded content</p>",
      });

      expect(result.subject).toBe("Fwd: Test Subject");
      expect(fetchMock).toHaveBeenCalledWith(
        `${baseUrl}/api/v1/emails/eml_xxx/forward`,
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({
            account_id: "acc_xxx",
            to: [{ identifier: "forward@example.com", display_name: "ForwardRecipient" }],
            body_html: "<p>Forwarded content</p>",
          }),
        })
      );
    });

    it("should search emails with query", async () => {
      const mockResponse: PaginatedList<Email> = {
        object: "list",
        items: [],
        cursor: null,
        has_more: false,
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      await client.emails.search({
        account_id: "acc_xxx",
        q: "test query",
      });

      const url = fetchMock.mock.calls[0][0];
      expect(url).toMatch(/q=test[%20+]query/);
      expect(url).toContain("account_id=acc_xxx");
    });
  });

  describe("CalendarsClient", () => {
    it("should create a calendar event", async () => {
      const mockEvent: CalendarEvent = {
        object: "calendar_event",
        id: "evt_xxx",
        account_id: "acc_xxx",
        calendar_id: "cal_xxx",
        provider: "GOOGLE_CALENDAR",
        provider_id: "provider-event-id",
        title: "Meeting",
        description: "Team sync",
        location: "Conference Room",
        start_at: "2025-03-20T11:00:00Z",
        end_at: "2025-03-20T12:00:00Z",
        all_day: false,
        status: "CONFIRMED",
        attendees: [],
        reminders: [{ method: "popup", minutes_before: 10 }],
        conference_url: null,
        recurrence: null,
        created_at: "2025-03-19T10:00:00Z",
        updated_at: "2025-03-19T10:00:00Z",
        metadata: {},
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockEvent),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      const result = await client.calendars.createEvent("cal_xxx", {
        title: "Meeting",
        description: "Team sync",
        start_at: "2025-03-20T11:00:00Z",
        end_at: "2025-03-20T12:00:00Z",
      });

      expect(result.title).toBe("Meeting");
    });

    it("should delete a calendar event", async () => {
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({}),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      await client.calendars.deleteEvent("cal_xxx", "evt_xxx");

      expect(fetchMock).toHaveBeenCalledWith(
        `${baseUrl}/api/v1/calendars/cal_xxx/events/evt_xxx`,
        expect.objectContaining({
          method: "DELETE",
        })
      );
    });
  });

  describe("WebhooksClient", () => {
    it("should create a webhook", async () => {
      const mockWebhook: Webhook = {
        object: "webhook",
        id: "whk_xxx",
        url: "https://example.com/webhook",
        events: ["message.received", "email.received"],
        secret: "whsec_xxx",
        active: true,
        created_at: "2025-03-19T10:00:00Z",
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockWebhook),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      const result = await client.webhooks.create({
        url: "https://example.com/webhook",
        events: ["message.received", "email.received"],
      });

      expect(result.url).toBe("https://example.com/webhook");
      expect(result.events).toContain("message.received");
    });

    it("should verify webhook signature", async () => {
      // This is a simplified test - actual verification requires proper HMAC
      const { WebhooksClient } = await import("../src/index.js");
      const isValid = await WebhooksClient.verifySignature(
        JSON.stringify({ event: "test" }),
        "sha256=invalid",
        "secret"
      );

      // Should return false for invalid signature
      expect(isValid).toBe(false);
    });
  });

  describe("AttendeesClient", () => {
    it("should get an attendee", async () => {
      const mockAttendee: Attendee = {
        object: "attendee",
        id: "att_xxx",
        account_id: "acc_xxx",
        provider: "WHATSAPP",
        provider_id: "provider-att-id",
        name: "John Doe",
        identifier: "1234567890",
        identifier_type: "PHONE_NUMBER",
        avatar_url: null,
        is_self: false,
        metadata: {},
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockAttendee),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      const result = await client.attendees.get("att_xxx");

      expect(result.name).toBe("John Doe");
      expect(result.identifier_type).toBe("PHONE_NUMBER");
    });

    it("should list attendee chats", async () => {
      const mockResponse: PaginatedList<Chat> = {
        object: "list",
        items: [],
        cursor: null,
        has_more: false,
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const client = new OndapileClient({ apiKey, baseUrl });
      await client.attendees.listChats("att_xxx");

      expect(fetchMock).toHaveBeenCalledWith(
        `${baseUrl}/api/v1/attendees/att_xxx/chats`,
        expect.any(Object)
      );
    });
  });

  describe("OndapileClient initialization", () => {
    it("should create client with all sub-clients", () => {
      const client = new OndapileClient({ apiKey, baseUrl });

      expect(client.accounts).toBeDefined();
      expect(client.chats).toBeDefined();
      expect(client.messages).toBeDefined();
      expect(client.emails).toBeDefined();
      expect(client.calendars).toBeDefined();
      expect(client.webhooks).toBeDefined();
      expect(client.attendees).toBeDefined();
    });

    it("should remove trailing slash from baseUrl", async () => {
      const mockResponse: PaginatedList<Chat> = {
        object: "list",
        items: [],
        cursor: null,
        has_more: false,
      };

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const client = new OndapileClient({
        apiKey,
        baseUrl: "http://localhost:8080/",
      });
      await client.chats.list();

      const url = fetchMock.mock.calls[0][0];
      expect(url).toBe("http://localhost:8080/api/v1/chats");
    });
  });
});

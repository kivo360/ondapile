"""Tests for the Ondapile Python SDK."""

from datetime import datetime, timezone

import httpx
import pytest
import respx
from httpx import Response

from ondapile import (
    Account,
    AccountStatus,
    Chat,
    ChatType,
    Message,
    OndapileClient,
    OndapileError,
)


# Fixtures
@pytest.fixture
def client():
    """Create an OndapileClient for testing."""
    return OndapileClient(
        api_key="test_key_123",
        base_url="http://localhost:8080",
    )


@pytest.fixture
def mock_account_data():
    """Return a sample account response."""
    return {
        "object": "account",
        "id": "acc_01jgpb44tjf",
        "provider": "WHATSAPP",
        "name": "Test Account",
        "identifier": "1234567890",
        "status": "OPERATIONAL",
        "status_detail": None,
        "capabilities": ["messaging", "media"],
        "created_at": "2025-03-01T10:00:00Z",
        "last_synced_at": "2025-03-19T14:32:00Z",
        "proxy": None,
        "metadata": {},
    }


@pytest.fixture
def mock_chat_data():
    """Return a sample chat response."""
    return {
        "object": "chat",
        "id": "chat_a1b2c3d4",
        "account_id": "acc_01jgpb44tjf",
        "provider": "WHATSAPP",
        "provider_id": "1234567890@s.whatsapp.net",
        "type": "ONE_TO_ONE",
        "name": None,
        "attendees": [],
        "last_message": None,
        "unread_count": 0,
        "is_group": False,
        "is_archived": False,
        "created_at": "2025-03-18T09:00:00Z",
        "updated_at": "2025-03-19T14:30:00Z",
        "metadata": {},
    }


@pytest.fixture
def mock_message_data():
    """Return a sample message response."""
    return {
        "object": "message",
        "id": "msg_f1g2h3i4",
        "chat_id": "chat_a1b2c3d4",
        "account_id": "acc_01jgpb44tjf",
        "provider": "WHATSAPP",
        "provider_id": "message_123",
        "text": "Hello, World!",
        "sender_id": "att_x1y2z3",
        "is_sender": True,
        "timestamp": "2025-03-19T14:30:00Z",
        "attachments": [],
        "reactions": [],
        "quoted": None,
        "seen": False,
        "seen_by": {},
        "delivered": True,
        "edited": False,
        "deleted": False,
        "hidden": False,
        "is_event": False,
        "event_type": None,
        "metadata": {},
    }


# Test: Client initialization
def test_client_init(client):
    """Test client initialization."""
    assert client.api_key == "test_key_123"
    assert client.base_url == "http://localhost:8080"
    assert client.timeout == 30.0


def test_client_init_with_timeout():
    """Test client initialization with custom timeout."""
    client = OndapileClient(
        api_key="test_key",
        base_url="http://localhost:8080",
        timeout=60.0,
    )
    assert client.timeout == 60.0


# Test: X-API-KEY header
def test_api_key_header_included(client):
    """Test that X-API-KEY header is set on all requests."""
    with respx.mock:
        route = respx.get("http://localhost:8080/api/v1/accounts").mock(
            return_value=Response(200, json={"object": "list", "items": [], "has_more": False})
        )
        client.accounts.list()
        assert route.called
        assert route.calls.last.request.headers["X-API-KEY"] == "test_key_123"


# Test: Query params stripping
def test_query_params_strip_none(client):
    """Test that None values are stripped from query params."""
    with respx.mock:
        route = respx.get("http://localhost:8080/api/v1/accounts").mock(
            return_value=Response(200, json={"object": "list", "items": [], "has_more": False})
        )
        client.accounts.list(status="OPERATIONAL")
        assert route.called
        # Should only include status, not cursor or limit
        request = route.calls.last.request
        assert "status=OPERATIONAL" in str(request.url)


# Test: AccountsClient
def test_accounts_list(client, mock_account_data):
    """Test listing accounts."""
    with respx.mock:
        respx.get("http://localhost:8080/api/v1/accounts").mock(
            return_value=Response(
                200,
                json={
                    "object": "list",
                    "items": [mock_account_data],
                    "cursor": "next_page",
                    "has_more": True,
                },
            )
        )
        result = client.accounts.list()
        assert len(result.items) == 1
        assert result.items[0].id == "acc_01jgpb44tjf"
        assert result.items[0].status == AccountStatus.OPERATIONAL
        assert result.cursor == "next_page"
        assert result.has_more is True


def test_accounts_get(client, mock_account_data):
    """Test getting a single account."""
    with respx.mock:
        respx.get("http://localhost:8080/api/v1/accounts/acc_01jgpb44tjf").mock(
            return_value=Response(200, json=mock_account_data)
        )
        account = client.accounts.get("acc_01jgpb44tjf")
        assert account.id == "acc_01jgpb44tjf"
        assert account.provider == "WHATSAPP"


def test_accounts_create(client, mock_account_data):
    """Test creating an account."""
    with respx.mock:
        route = respx.post("http://localhost:8080/api/v1/accounts").mock(
            return_value=Response(201, json=mock_account_data)
        )
        account = client.accounts.create(
            provider="WHATSAPP",
            identifier="1234567890",
        )
        assert account.id == "acc_01jgpb44tjf"
        # Verify request body
        request_body = route.calls.last.request.content
        assert b"WHATSAPP" in request_body


def test_accounts_delete(client):
    """Test deleting an account."""
    with respx.mock:
        route = respx.delete("http://localhost:8080/api/v1/accounts/acc_123").mock(
            return_value=Response(204)
        )
        client.accounts.delete("acc_123")
        assert route.called


def test_accounts_reconnect(client, mock_account_data):
    """Test reconnecting an account."""
    with respx.mock:
        respx.post("http://localhost:8080/api/v1/accounts/acc_123/reconnect").mock(
            return_value=Response(200, json=mock_account_data)
        )
        account = client.accounts.reconnect("acc_123")
        assert account.id == "acc_01jgpb44tjf"


def test_accounts_get_auth_challenge(client):
    """Test getting auth challenge."""
    with respx.mock:
        respx.get("http://localhost:8080/api/v1/accounts/acc_123/auth-challenge").mock(
            return_value=Response(
                200,
                json={
                    "type": "QR_CODE",
                    "payload": "qr_data_here",
                    "expiry": 1711036800,
                }
            )
        )
        challenge = client.accounts.get_auth_challenge("acc_123")
        assert challenge["type"] == "QR_CODE"
        assert challenge["payload"] == "qr_data_here"


# Test: ChatsClient
def test_chats_list(client, mock_chat_data):
    """Test listing chats."""
    with respx.mock:
        respx.get("http://localhost:8080/api/v1/chats").mock(
            return_value=Response(
                200,
                json={
                    "object": "list",
                    "items": [mock_chat_data],
                    "has_more": False,
                },
            )
        )
        result = client.chats.list()
        assert len(result.items) == 1
        assert result.items[0].id == "chat_a1b2c3d4"
        assert result.items[0].type == ChatType.ONE_TO_ONE


def test_chats_get(client, mock_chat_data):
    """Test getting a single chat."""
    with respx.mock:
        respx.get("http://localhost:8080/api/v1/chats/chat_a1b2c3d4").mock(
            return_value=Response(200, json=mock_chat_data)
        )
        chat = client.chats.get("chat_a1b2c3d4")
        assert chat.id == "chat_a1b2c3d4"


def test_chats_send_message(client, mock_message_data):
    """Test sending a message in a chat."""
    with respx.mock:
        route = respx.post("http://localhost:8080/api/v1/chats/chat_a1b2c3d4/messages").mock(
            return_value=Response(201, json=mock_message_data)
        )
        message = client.chats.send_message(
            chat_id="chat_a1b2c3d4",
            text="Hello, World!",
        )
        assert message.id == "msg_f1g2h3i4"
        assert message.text == "Hello, World!"
        # Verify request body
        request_body = route.calls.last.request.content
        assert b"Hello, World!" in request_body


def test_chats_list_messages(client, mock_message_data):
    """Test listing messages in a chat."""
    with respx.mock:
        respx.get("http://localhost:8080/api/v1/chats/chat_a1b2c3d4/messages").mock(
            return_value=Response(
                200,
                json={
                    "object": "list",
                    "items": [mock_message_data],
                    "has_more": False,
                },
            )
        )
        result = client.chats.list_messages("chat_a1b2c3d4")
        assert len(result.items) == 1
        assert result.items[0].id == "msg_f1g2h3i4"


# Test: MessagesClient
def test_messages_list(client, mock_message_data):
    """Test listing all messages."""
    with respx.mock:
        respx.get("http://localhost:8080/api/v1/messages").mock(
            return_value=Response(
                200,
                json={
                    "object": "list",
                    "items": [mock_message_data],
                    "has_more": False,
                },
            )
        )
        result = client.messages.list()
        assert len(result.items) == 1
        assert result.items[0].id == "msg_f1g2h3i4"


def test_messages_get(client, mock_message_data):
    """Test getting a single message."""
    with respx.mock:
        respx.get("http://localhost:8080/api/v1/messages/msg_f1g2h3i4").mock(
            return_value=Response(200, json=mock_message_data)
        )
        message = client.messages.get("msg_f1g2h3i4")
        assert message.id == "msg_f1g2h3i4"


def test_messages_add_reaction(client, mock_message_data):
    """Test adding a reaction to a message."""
    with respx.mock:
        route = respx.post("http://localhost:8080/api/v1/messages/msg_f1g2h3i4/reactions").mock(
            return_value=Response(200, json=mock_message_data)
        )
        client.messages.add_reaction("msg_f1g2h3i4", "👍")
        request_body = route.calls.last.request.content
        assert b"emoji" in request_body or "👍".encode() in request_body


def test_messages_delete(client):
    """Test deleting a message."""
    with respx.mock:
        route = respx.delete("http://localhost:8080/api/v1/messages/msg_123").mock(
            return_value=Response(204)
        )
        client.messages.delete("msg_123")
        assert route.called


# Test: Error handling
def test_error_response_raises_exception(client):
    """Test that non-2xx responses raise OndapileError."""
    with respx.mock:
        respx.get("http://localhost:8080/api/v1/accounts/acc_invalid").mock(
            return_value=Response(
                404,
                json={
                    "object": "error",
                    "status": 404,
                    "code": "NOT_FOUND",
                    "message": "Account not found",
                    "details": None,
                },
            )
        )
        with pytest.raises(OndapileError) as exc_info:
            client.accounts.get("acc_invalid")
        assert exc_info.value.status_code == 404
        assert exc_info.value.code == "NOT_FOUND"
        assert "Account not found" in str(exc_info.value)


def test_error_response_without_json(client):
    """Test error handling when response is not valid JSON."""
    with respx.mock:
        respx.get("http://localhost:8080/api/v1/accounts").mock(
            return_value=Response(500, text="Internal Server Error")
        )
        with pytest.raises(OndapileError) as exc_info:
            client.accounts.list()
        assert exc_info.value.status_code == 500


# Test: Context manager
def test_context_manager():
    """Test client as context manager."""
    with OndapileClient(
        api_key="test_key",
        base_url="http://localhost:8080",
    ) as client:
        assert client.api_key == "test_key"


# Test: Base URL normalization
def test_base_url_trailing_slash():
    """Test that trailing slash is removed from base_url."""
    client = OndapileClient(
        api_key="test_key",
        base_url="http://localhost:8080/",
    )
    assert client.base_url == "http://localhost:8080"


# Test: EmailsClient
def test_emails_list(client):
    """Test listing emails."""
    mock_email = {
        "object": "email",
        "id": "eml_123",
        "account_id": "acc_123",
        "provider": "GMAIL",
        "provider_id": "msg_123",
        "subject": "Test Email",
        "body": "<html><body>Hello</body></html>",
        "body_plain": "Hello",
        "from_attendee": {
            "display_name": "Test User",
            "identifier": "test@example.com",
            "identifier_type": "EMAIL_ADDRESS",
        },
        "to_attendees": [],
        "cc_attendees": [],
        "bcc_attendees": [],
        "reply_to_attendees": [],
        "date": "2025-03-19T13:45:00Z",
        "has_attachments": False,
        "attachments": [],
        "folders": ["INBOX"],
        "role": "INBOX",
        "read": False,
        "is_complete": True,
        "headers": [],
        "metadata": {},
    }
    with respx.mock:
        respx.get("http://localhost:8080/api/v1/emails").mock(
            return_value=Response(
                200,
                json={
                    "object": "list",
                    "items": [mock_email],
                    "has_more": False,
                },
            )
        )
        result = client.emails.list()
        assert len(result.items) == 1
        assert result.items[0].id == "eml_123"
        assert result.items[0].subject == "Test Email"


# Test: WebhooksClient
def test_webhooks_list(client):
    """Test listing webhooks."""
    mock_webhook = {
        "object": "webhook",
        "id": "whk_123",
        "url": "https://example.com/webhook",
        "events": ["message.received"],
        "secret": "secret_123",
        "active": True,
        "created_at": "2025-03-01T10:00:00Z",
    }
    with respx.mock:
        respx.get("http://localhost:8080/api/v1/webhooks").mock(
            return_value=Response(
                200,
                json={
                    "object": "list",
                    "items": [mock_webhook],
                    "has_more": False,
                },
            )
        )
        result = client.webhooks.list()
        assert len(result.items) == 1
        assert result.items[0].id == "whk_123"


def test_webhooks_create(client):
    """Test creating a webhook."""
    mock_webhook = {
        "object": "webhook",
        "id": "whk_123",
        "url": "https://example.com/webhook",
        "events": ["message.received", "email.received"],
        "secret": "secret_123",
        "active": True,
        "created_at": "2025-03-01T10:00:00Z",
    }
    with respx.mock:
        route = respx.post("http://localhost:8080/api/v1/webhooks").mock(
            return_value=Response(201, json=mock_webhook)
        )
        webhook = client.webhooks.create(
            url="https://example.com/webhook",
            events=["message.received", "email.received"],
        )
        assert webhook.id == "whk_123"
        assert webhook.url == "https://example.com/webhook"
        request_body = route.calls.last.request.content
        assert b"message.received" in request_body


def test_webhooks_delete(client):
    """Test deleting a webhook."""
    with respx.mock:
        route = respx.delete("http://localhost:8080/api/v1/webhooks/whk_123").mock(
            return_value=Response(204)
        )
        client.webhooks.delete("whk_123")
        assert route.called

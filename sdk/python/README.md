# Ondapile Python SDK

Python SDK for the Ondapile unified communication API.

## Installation

```bash
pip install ondapile
```

## Usage

```python
from ondapile import OndapileClient

client = OndapileClient(
    api_key="your_api_key",
    base_url="https://api.ondapile.local"
)

# List accounts
accounts = client.accounts.list()

# List chats
chats = client.chats.list(limit=25)

# Send a message
client.chats.send_message("chat_abc", text="Hello!")
```

## Documentation

See the [API documentation](https://docs.ondapile.com) for full details.

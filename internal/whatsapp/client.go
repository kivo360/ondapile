package whatsapp

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"
)

// CreateClient creates a new whatsmeow client with SQLite device store.
// Each account gets its own database file at {deviceStorePath}/{accountID}.db
func CreateClient(ctx context.Context, accountID string, deviceStorePath string) (*whatsmeow.Client, error) {
	dbPath := fmt.Sprintf("%s/%s.db", deviceStorePath, accountID)

	// Create SQLite store container
	container, err := sqlstore.New(ctx, "sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", dbPath), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create sqlstore: %w", err)
	}

	// Get the first device, or create one if none exists
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// Create client with no-op logger to keep logs clean
	// Users can enable logging by replacing waLog.Noop with a real logger
	client := whatsmeow.NewClient(deviceStore, waLog.Noop)

	return client, nil
}

// parseJID parses a JID string into a types.JID.
func parseJID(jidStr string) (types.JID, error) {
	return types.ParseJID(jidStr)
}

// newJIDFromPhone creates a JID from a phone number.
// Phone number should be in international format without the + prefix.
func newJIDFromPhone(phoneNumber string) types.JID {
	return types.NewJID(phoneNumber, types.DefaultUserServer)
}

// jidToString converts a JID to its string representation.
func jidToString(jid types.JID) string {
	return jid.String()
}

// isGroupJID checks if a JID represents a group chat.
func isGroupJID(jid types.JID) bool {
	return jid.Server == types.GroupServer
}

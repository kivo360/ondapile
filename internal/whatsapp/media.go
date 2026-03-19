package whatsapp

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
)

// downloadMedia downloads media from WhatsApp using the direct path.
func (p *WhatsAppProvider) downloadMedia(ctx context.Context, client *whatsmeow.Client, directPath string) ([]byte, error) {
	// DownloadMediaWithPath requires encryption details we don't have here.
	// Use a simpler approach: download to memory using DownloadAny if possible,
	// or return an error indicating we need the full message.
	return nil, fmt.Errorf("direct path download requires encryption keys from original message")
}

// DownloadMediaFromMessage downloads media from a specific message.
// This uses the message's media info to download the content.
func DownloadMediaFromMessage(ctx context.Context, client *whatsmeow.Client, msg *waE2E.Message) ([]byte, string, error) {
	var downloadable whatsmeow.DownloadableMessage
	var mimeType string

	// Extract downloadable content based on message type
	switch {
	case msg.ImageMessage != nil:
		downloadable = msg.ImageMessage
		if msg.ImageMessage.Mimetype != nil {
			mimeType = *msg.ImageMessage.Mimetype
		}
	case msg.VideoMessage != nil:
		downloadable = msg.VideoMessage
		if msg.VideoMessage.Mimetype != nil {
			mimeType = *msg.VideoMessage.Mimetype
		}
	case msg.AudioMessage != nil:
		downloadable = msg.AudioMessage
		if msg.AudioMessage.Mimetype != nil {
			mimeType = *msg.AudioMessage.Mimetype
		}
	case msg.DocumentMessage != nil:
		downloadable = msg.DocumentMessage
		if msg.DocumentMessage.Mimetype != nil {
			mimeType = *msg.DocumentMessage.Mimetype
		}
	case msg.StickerMessage != nil:
		downloadable = msg.StickerMessage
		mimeType = "image/webp"
	default:
		return nil, "", fmt.Errorf("message does not contain downloadable media")
	}

	// Download the media
	data, err := client.Download(ctx, downloadable)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download media: %w", err)
	}

	return data, mimeType, nil
}

// GetMediaInfo extracts media information from a message.
func GetMediaInfo(msg *waE2E.Message) (filename, caption, mimeType string, fileSize uint64) {
	switch {
	case msg.ImageMessage != nil:
		if msg.ImageMessage.Caption != nil {
			caption = *msg.ImageMessage.Caption
		}
		if msg.ImageMessage.Mimetype != nil {
			mimeType = *msg.ImageMessage.Mimetype
		}
		fileSize = msg.ImageMessage.GetFileLength()
		filename = "image"

	case msg.VideoMessage != nil:
		if msg.VideoMessage.Caption != nil {
			caption = *msg.VideoMessage.Caption
		}
		if msg.VideoMessage.Mimetype != nil {
			mimeType = *msg.VideoMessage.Mimetype
		}
		fileSize = msg.VideoMessage.GetFileLength()
		filename = "video"

	case msg.AudioMessage != nil:
		if msg.AudioMessage.Mimetype != nil {
			mimeType = *msg.AudioMessage.Mimetype
		}
		fileSize = msg.AudioMessage.GetFileLength()
		if msg.AudioMessage.GetPTT() {
			filename = "voice_message"
		} else {
			filename = "audio"
		}

	case msg.DocumentMessage != nil:
		if msg.DocumentMessage.FileName != nil {
			filename = *msg.DocumentMessage.FileName
		}
		if msg.DocumentMessage.Caption != nil {
			caption = *msg.DocumentMessage.Caption
		}
		if msg.DocumentMessage.Mimetype != nil {
			mimeType = *msg.DocumentMessage.Mimetype
		}
		fileSize = msg.DocumentMessage.GetFileLength()

	case msg.StickerMessage != nil:
		mimeType = "image/webp"
		fileSize = msg.StickerMessage.GetFileLength()
		filename = "sticker"
	}

	return
}

// HasMedia checks if a message contains media.
func HasMedia(msg *waE2E.Message) bool {
	return msg.ImageMessage != nil ||
		msg.VideoMessage != nil ||
		msg.AudioMessage != nil ||
		msg.DocumentMessage != nil ||
		msg.StickerMessage != nil
}

// GetMediaType returns the type of media in a message.
func GetMediaType(msg *waE2E.Message) string {
	switch {
	case msg.ImageMessage != nil:
		return "image"
	case msg.VideoMessage != nil:
		return "video"
	case msg.AudioMessage != nil:
		if msg.AudioMessage.GetPTT() {
			return "voice"
		}
		return "audio"
	case msg.DocumentMessage != nil:
		return "document"
	case msg.StickerMessage != nil:
		return "sticker"
	default:
		return ""
	}
}

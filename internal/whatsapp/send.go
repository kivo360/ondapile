package whatsapp

import (
	"context"
	"fmt"
	"time"

	"ondapile/internal/adapter"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// sendText sends a text message via WhatsApp.
func (p *WhatsAppProvider) sendText(ctx context.Context, client *whatsmeow.Client, chatJID types.JID, text string) (*whatsmeow.SendResponse, error) {
	msg := &waE2E.Message{
		Conversation: proto.String(text),
	}

	resp, err := client.SendMessage(ctx, chatJID, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send text message: %w", err)
	}

	return &resp, nil
}

// sendMedia sends a media message via WhatsApp.
func (p *WhatsAppProvider) sendMedia(ctx context.Context, client *whatsmeow.Client, chatJID types.JID, attachment adapter.AttachmentUpload, caption string) (*whatsmeow.SendResponse, error) {
	// Upload media first
	uploadResp, err := client.Upload(ctx, attachment.Content, getMediaType(attachment.MimeType))
	if err != nil {
		return nil, fmt.Errorf("failed to upload media: %w", err)
	}

	// Build appropriate message based on mime type
	var msg *waE2E.Message

	switch getMediaCategory(attachment.MimeType) {
	case "image":
		msg = buildImageMessage(uploadResp, attachment, caption)
	case "video":
		msg = buildVideoMessage(uploadResp, attachment, caption)
	case "audio":
		msg = buildAudioMessage(uploadResp, attachment)
	case "document":
		msg = buildDocumentMessage(uploadResp, attachment, caption)
	default:
		msg = buildDocumentMessage(uploadResp, attachment, caption)
	}

	resp, err := client.SendMessage(ctx, chatJID, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send media message: %w", err)
	}

	return &resp, nil
}

// buildImageMessage builds an image message proto.
func buildImageMessage(uploadResp whatsmeow.UploadResponse, attachment adapter.AttachmentUpload, caption string) *waE2E.Message {
	return &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(attachment.Content))),
			Mimetype:      proto.String(attachment.MimeType),
			Caption:       proto.String(caption),
		},
	}
}

// buildVideoMessage builds a video message proto.
func buildVideoMessage(uploadResp whatsmeow.UploadResponse, attachment adapter.AttachmentUpload, caption string) *waE2E.Message {
	return &waE2E.Message{
		VideoMessage: &waE2E.VideoMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(attachment.Content))),
			Mimetype:      proto.String(attachment.MimeType),
			Caption:       proto.String(caption),
			// GIF playback flag for GIFs
			GifPlayback: proto.Bool(isGIF(attachment.MimeType)),
		},
	}
}

// buildAudioMessage builds an audio/voice message proto.
func buildAudioMessage(uploadResp whatsmeow.UploadResponse, attachment adapter.AttachmentUpload) *waE2E.Message {
	return &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(attachment.Content))),
			Mimetype:      proto.String(attachment.MimeType),
			// PTT = Push To Talk (voice message)
			PTT: proto.Bool(isVoiceMessage(attachment.MimeType)),
		},
	}
}

// buildDocumentMessage builds a document message proto.
func buildDocumentMessage(uploadResp whatsmeow.UploadResponse, attachment adapter.AttachmentUpload, caption string) *waE2E.Message {
	return &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(attachment.Content))),
			Mimetype:      proto.String(attachment.MimeType),
			Title:         proto.String(attachment.Filename),
			FileName:      proto.String(attachment.Filename),
			Caption:       proto.String(caption),
		},
	}
}

// getMediaType returns the whatsmeow media type for a mime type.
func getMediaType(mimeType string) whatsmeow.MediaType {
	switch getMediaCategory(mimeType) {
	case "image":
		return whatsmeow.MediaImage
	case "video":
		return whatsmeow.MediaVideo
	case "audio":
		return whatsmeow.MediaAudio
	case "document":
		return whatsmeow.MediaDocument
	default:
		return whatsmeow.MediaDocument
	}
}

// getMediaCategory returns a broad category for a mime type.
func getMediaCategory(mimeType string) string {
	if isImageType(mimeType) {
		return "image"
	}
	if isVideoType(mimeType) {
		return "video"
	}
	if isAudioType(mimeType) {
		return "audio"
	}
	return "document"
}

// isImageType checks if mime type is an image.
func isImageType(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp", "image/bmp":
		return true
	default:
		return false
	}
}

// isVideoType checks if mime type is a video.
func isVideoType(mimeType string) bool {
	switch mimeType {
	case "video/mp4", "video/avi", "video/mov", "video/quicktime", "video/webm":
		return true
	default:
		return false
	}
}

// isAudioType checks if mime type is audio.
func isAudioType(mimeType string) bool {
	switch mimeType {
	case "audio/mpeg", "audio/mp3", "audio/ogg", "audio/wav", "audio/webm", "audio/aac", "audio/mp4":
		return true
	default:
		return false
	}
}

// isGIF checks if mime type is a GIF.
func isGIF(mimeType string) bool {
	return mimeType == "image/gif"
}

// isVoiceMessage checks if mime type is a voice message.
// For voice messages, we typically use ogg format with opus codec.
func isVoiceMessage(mimeType string) bool {
	return mimeType == "audio/ogg" || mimeType == "audio/opus"
}

// SendTextMessage is a helper function to send a simple text message.
// This can be used directly if you need to send without going through the full provider flow.
func SendTextMessage(ctx context.Context, client *whatsmeow.Client, chatJID types.JID, text string) (*whatsmeow.SendResponse, error) {
	msg := &waE2E.Message{
		Conversation: proto.String(text),
	}

	resp, err := client.SendMessage(ctx, chatJID, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return &resp, nil
}


// SendQuotedMessage sends a message that quotes/replies to another message.
func (p *WhatsAppProvider) sendQuotedMessage(ctx context.Context, client *whatsmeow.Client, chatJID types.JID, text string, quotedMsgID string) (*whatsmeow.SendResponse, error) {
	// Build extended text message with context info for quoting
	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:    proto.String(quotedMsgID),
				Participant: proto.String(chatJID.String()),
			},
		},
	}

	resp, err := client.SendMessage(ctx, chatJID, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send quoted message: %w", err)
	}

	return &resp, nil
}

// BuildMessageWithMentions creates a message that mentions specific users.
func BuildMessageWithMentions(text string, mentionedJIDs []string) *waE2E.Message {
	return &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waE2E.ContextInfo{
				MentionedJID: mentionedJIDs,
			},
		},
}

}

// SendPresence sends presence status (typing, available, etc.) to a chat.
func SendPresence(ctx context.Context, client *whatsmeow.Client, chatJID types.JID, presence types.ChatPresence, media types.ChatPresenceMedia) error {
	return client.SendChatPresence(ctx, chatJID, presence, media)
}

// MarkRead marks messages as read in a chat.
func MarkRead(ctx context.Context, client *whatsmeow.Client, messageIDs []types.MessageID, senderJID types.JID, chatJID types.JID) error {
	return client.MarkRead(ctx, messageIDs, time.Now(), chatJID, senderJID)
}

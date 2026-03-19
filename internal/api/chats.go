package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
	"ondapile/internal/store"
)

type ChatHandler struct {
	store *store.Store
	chats *store.ChatStore
	msgs  *store.MessageStore
}

func NewChatHandler(s *store.Store) *ChatHandler {
	return &ChatHandler{
		store: s,
		chats: store.NewChatStore(s),
		msgs:  store.NewMessageStore(s),
	}
}

// GET /chats
func (h *ChatHandler) List(c *gin.Context) {
	f := GetProviderFilter(c)
	p := GetPagination(c)

	chats, nextCursor, hasMore, err := h.chats.List(c.Request.Context(), f.AccountID, f.IsGroup, p.Cursor, p.Limit)
	if err != nil {
		Internal(c, "Failed to list chats")
		return
	}

	items := make([]any, len(chats))
	for i, ch := range chats {
		items[i] = ch
	}

	c.JSON(http.StatusOK, model.NewPaginatedList(items, nextCursor, hasMore))
}

// POST /chats
func (h *ChatHandler) Create(c *gin.Context) {
	var req struct {
		AccountID          string `json:"account_id" binding:"required"`
		AttendeeIdentifier string `json:"attendee_identifier" binding:"required"`
		Text               string `json:"text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Validation(c, err.Error())
		return
	}

	accountStore := store.NewAccountStore(h.store)
	account, err := accountStore.GetByID(c.Request.Context(), req.AccountID)
	if err != nil || account == nil {
		NotFound(c, "Account not found")
		return
	}

	prov, err := adapter.Get(account.Provider)
	if err != nil {
		ProviderError(c, "Provider not available")
		return
	}

	chat, err := prov.StartChat(c.Request.Context(), req.AccountID, adapter.StartChatRequest{
		AttendeeIdentifier: req.AttendeeIdentifier,
		Text:               req.Text,
	})
	if err != nil {
		ProviderError(c, "Failed to start chat: "+err.Error())
		return
	}

	c.JSON(http.StatusCreated, chat)
}

// GET /chats/:id
func (h *ChatHandler) Get(c *gin.Context) {
	id := c.Param("id")
	chat, err := h.chats.GetByID(c.Request.Context(), id)
	if err != nil || chat == nil {
		NotFound(c, "Chat not found")
		return
	}

	c.JSON(http.StatusOK, chat)
}

// PATCH /chats/:id
func (h *ChatHandler) Update(c *gin.Context) {
	id := c.Param("id")
	chat, err := h.chats.GetByID(c.Request.Context(), id)
	if err != nil || chat == nil {
		NotFound(c, "Chat not found")
		return
	}

	var req struct {
		Action string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Validation(c, err.Error())
		return
	}

	ctx := c.Request.Context()
	switch req.Action {
	case "archive":
		if err := h.chats.Archive(ctx, id, true); err != nil {
			Internal(c, "Failed to archive chat")
			return
		}
	case "unarchive":
		if err := h.chats.Archive(ctx, id, false); err != nil {
			Internal(c, "Failed to unarchive chat")
			return
		}
	case "mark_read":
		if err := h.chats.ResetUnread(ctx, id); err != nil {
			Internal(c, "Failed to mark chat as read")
			return
		}
	case "pin":
		// Pinned status is stored in metadata - for now just return the chat
		// Pinning can be implemented by updating metadata if needed
	case "unpin":
		// Unpinning can be implemented by updating metadata if needed
	default:
		Validation(c, "Invalid action")
		return
	}

	// Reload chat to get updated state
	chat, err = h.chats.GetByID(ctx, id)
	if err != nil {
		Internal(c, "Failed to reload chat")
		return
	}

	c.JSON(http.StatusOK, chat)
}

// DELETE /chats/:id
func (h *ChatHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.chats.Delete(c.Request.Context(), id); err != nil {
		Internal(c, "Failed to delete chat")
		return
	}

	c.JSON(http.StatusOK, gin.H{"object": "chat", "id": id, "deleted": true})
}

// GET /chats/:id/messages
func (h *ChatHandler) ListMessages(c *gin.Context) {
	id := c.Param("id")
	p := GetPagination(c)

	messages, nextCursor, hasMore, err := h.msgs.ListByChat(c.Request.Context(), id, p.Cursor, p.Limit)
	if err != nil {
		Internal(c, "Failed to list messages")
		return
	}

	items := make([]any, len(messages))
	for i, m := range messages {
		items[i] = m
	}

	c.JSON(http.StatusOK, model.NewPaginatedList(items, nextCursor, hasMore))
}

// POST /chats/:id/messages
func (h *ChatHandler) SendMessage(c *gin.Context) {
	chatID := c.Param("id")

	chat, err := h.chats.GetByID(c.Request.Context(), chatID)
	if err != nil || chat == nil {
		NotFound(c, "Chat not found")
		return
	}

	var req struct {
		Text            string                     `json:"text"`
		Attachments     []adapter.AttachmentUpload `json:"attachments"`
		QuotedMessageID *string                    `json:"quoted_message_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Validation(c, err.Error())
		return
	}

	prov, err := adapter.Get(chat.Provider)
	if err != nil {
		ProviderError(c, "Provider not available")
		return
	}

	msg, err := prov.SendMessage(c.Request.Context(), chat.AccountID, chatID, adapter.SendMessageRequest{
		Text:        req.Text,
		Attachments: req.Attachments,
		QuotedMsgID: req.QuotedMessageID,
	})
	if err != nil {
		ProviderError(c, "Failed to send message: "+err.Error())
		return
	}

	// Update chat's last message
	preview := req.Text
	msgStore := store.NewMessageStore(h.store)
	_ = h.chats.UpdateLastMessage(c.Request.Context(), chatID, &preview)
	msgStore.Create(c.Request.Context(), msg)

	c.JSON(http.StatusCreated, msg)
}

// GET /chats/:id/attendees
func (h *ChatHandler) ListAttendees(c *gin.Context) {
	chatID := c.Param("id")

	chat, err := h.chats.GetByID(c.Request.Context(), chatID)
	if err != nil || chat == nil {
		NotFound(c, "Chat not found")
		return
	}

	prov, err := adapter.Get(chat.Provider)
	if err != nil {
		ProviderError(c, "Provider not available")
		return
	}

	attendees, err := prov.ListAttendees(c.Request.Context(), chat.AccountID, adapter.ListOpts{
		Limit: 100,
	})
	if err != nil {
		Internal(c, "Failed to list attendees")
		return
	}

	c.JSON(http.StatusOK, attendees)
}

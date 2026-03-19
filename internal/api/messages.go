package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
	"ondapile/internal/store"
)

type MessageHandler struct {
	store *store.Store
	msgs  *store.MessageStore
}

func NewMessageHandler(s *store.Store) *MessageHandler {
	return &MessageHandler{
		store: s,
		msgs:  store.NewMessageStore(s),
	}
}

// GET /messages
func (h *MessageHandler) List(c *gin.Context) {
	f := GetProviderFilter(c)
	p := GetPagination(c)

	messages, nextCursor, hasMore, err := h.msgs.List(c.Request.Context(), f.AccountID, p.Cursor, p.Limit)
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

// GET /messages/:id
func (h *MessageHandler) Get(c *gin.Context) {
	id := c.Param("id")
	msg, err := h.msgs.GetByID(c.Request.Context(), id)
	if err != nil || msg == nil {
		NotFound(c, "Message not found")
		return
	}

	c.JSON(http.StatusOK, msg)
}

// DELETE /messages/:id
func (h *MessageHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	msg, err := h.msgs.GetByID(c.Request.Context(), id)
	if err != nil || msg == nil {
		NotFound(c, "Message not found")
		return
	}

	if err := h.msgs.Delete(c.Request.Context(), id); err != nil {
		Internal(c, "Failed to delete message")
		return
	}

	c.JSON(http.StatusOK, gin.H{"object": "message", "id": id, "deleted": true})
}

// GET /messages/:id/attachments/:att_id
func (h *MessageHandler) DownloadAttachment(c *gin.Context) {
	messageID := c.Param("id")
	attachmentID := c.Param("att_id")

	msg, err := h.msgs.GetByID(c.Request.Context(), messageID)
	if err != nil || msg == nil {
		NotFound(c, "Message not found")
		return
	}

	prov, err := adapter.Get(msg.Provider)
	if err != nil {
		ProviderError(c, "Provider not available")
		return
	}

	data, mimeType, err := prov.DownloadAttachment(c.Request.Context(), msg.AccountID, messageID, attachmentID)
	if err != nil {
		ProviderError(c, "Failed to download attachment: "+err.Error())
		return
	}

	c.Data(http.StatusOK, mimeType, data)
}

// POST /messages/:id/reactions
func (h *MessageHandler) AddReaction(c *gin.Context) {
	messageID := c.Param("id")

	msg, err := h.msgs.GetByID(c.Request.Context(), messageID)
	if err != nil || msg == nil {
		NotFound(c, "Message not found")
		return
	}

	var req struct {
		Emoji string `json:"emoji" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Validation(c, err.Error())
		return
	}

	prov, err := adapter.Get(msg.Provider)
	if err != nil {
		ProviderError(c, "Provider not available")
		return
	}

	// Send reaction as a message event
	_, err = prov.SendMessage(c.Request.Context(), msg.AccountID, msg.ChatID, adapter.SendMessageRequest{
		Text: req.Emoji,
	})
	if err != nil {
		ProviderError(c, "Failed to add reaction: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"object": "message", "id": messageID, "reaction_added": req.Emoji})
}

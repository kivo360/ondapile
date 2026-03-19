package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
	"ondapile/internal/store"
)

type AttendeeHandler struct {
	store  *store.Store
	chats  *store.ChatStore
	msgs   *store.MessageStore
}

func NewAttendeeHandler(s *store.Store) *AttendeeHandler {
	return &AttendeeHandler{
		store: s,
		chats: store.NewChatStore(s),
		msgs:  store.NewMessageStore(s),
	}
}

// GET /attendees
func (h *AttendeeHandler) List(c *gin.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		Validation(c, "account_id is required")
		return
	}

	accountStore := store.NewAccountStore(h.store)
	account, err := accountStore.GetByID(c.Request.Context(), accountID)
	if err != nil || account == nil {
		NotFound(c, "Account not found")
		return
	}

	prov, err := adapter.Get(account.Provider)
	if err != nil {
		ProviderError(c, "Provider not available")
		return
	}

	p := GetPagination(c)
	attendees, err := prov.ListAttendees(c.Request.Context(), accountID, adapter.ListOpts{
		Limit: p.Limit,
	})
	if err != nil {
		Internal(c, "Failed to list attendees")
		return
	}

	c.JSON(http.StatusOK, attendees)
}

// GET /attendees/:id
func (h *AttendeeHandler) Get(c *gin.Context) {
	id := c.Param("id")
	accountID := c.Query("account_id")
	if accountID == "" {
		Validation(c, "account_id is required")
		return
	}

	accountStore := store.NewAccountStore(h.store)
	account, err := accountStore.GetByID(c.Request.Context(), accountID)
	if err != nil || account == nil {
		NotFound(c, "Account not found")
		return
	}

	prov, err := adapter.Get(account.Provider)
	if err != nil {
		ProviderError(c, "Provider not available")
		return
	}

	attendee, err := prov.GetAttendee(c.Request.Context(), accountID, id)
	if err != nil || attendee == nil {
		NotFound(c, "Attendee not found")
		return
	}

	c.JSON(http.StatusOK, attendee)
}

// GET /attendees/:id/avatar
func (h *AttendeeHandler) GetAvatar(c *gin.Context) {
	id := c.Param("id")
	accountID := c.Query("account_id")
	if accountID == "" {
		Validation(c, "account_id is required")
		return
	}

	accountStore := store.NewAccountStore(h.store)
	account, err := accountStore.GetByID(c.Request.Context(), accountID)
	if err != nil || account == nil {
		NotFound(c, "Account not found")
		return
	}

	prov, err := adapter.Get(account.Provider)
	if err != nil {
		ProviderError(c, "Provider not available")
		return
	}

	data, mimeType, err := prov.DownloadAttachment(c.Request.Context(), accountID, id, "")
	if err != nil {
		NotFound(c, "Avatar not found")
		return
	}

	c.Data(200, mimeType, data)
}

// GET /attendees/:id/chats
func (h *AttendeeHandler) ListChats(c *gin.Context) {
	id := c.Param("id")
	accountID := c.Query("account_id")
	if accountID == "" {
		Validation(c, "account_id is required")
		return
	}

	p := GetPagination(c)

	// Get the attendee to find their provider_id
	accountStore := store.NewAccountStore(h.store)
	account, err := accountStore.GetByID(c.Request.Context(), accountID)
	if err != nil || account == nil {
		NotFound(c, "Account not found")
		return
	}

	prov, err := adapter.Get(account.Provider)
	if err != nil {
		ProviderError(c, "Provider not available")
		return
	}

	attendee, err := prov.GetAttendee(c.Request.Context(), accountID, id)
	if err != nil || attendee == nil {
		NotFound(c, "Attendee not found")
		return
	}

	// Use the attendee's provider_id to find their 1:1 chats
	chats, nextCursor, hasMore, err := h.chats.ListByAttendee(c.Request.Context(), accountID, attendee.ProviderID, p.Cursor, p.Limit)
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

// GET /attendees/:id/messages
func (h *AttendeeHandler) ListMessages(c *gin.Context) {
	id := c.Param("id")
	accountID := c.Query("account_id")
	if accountID == "" {
		Validation(c, "account_id is required")
		return
	}

	p := GetPagination(c)

	// Get the attendee to find their provider_id
	accountStore := store.NewAccountStore(h.store)
	account, err := accountStore.GetByID(c.Request.Context(), accountID)
	if err != nil || account == nil {
		NotFound(c, "Account not found")
		return
	}

	prov, err := adapter.Get(account.Provider)
	if err != nil {
		ProviderError(c, "Provider not available")
		return
	}

	attendee, err := prov.GetAttendee(c.Request.Context(), accountID, id)
	if err != nil || attendee == nil {
		NotFound(c, "Attendee not found")
		return
	}

	// List messages from this attendee
	messages, nextCursor, hasMore, err := h.msgs.ListBySender(c.Request.Context(), accountID, attendee.ProviderID, p.Cursor, p.Limit)
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

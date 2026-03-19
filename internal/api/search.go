package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ondapile/internal/model"
	"ondapile/internal/search"
	"ondapile/internal/store"
)

// SearchHandler handles semantic search endpoints.
type SearchHandler struct {
	store         *store.Store
	searchService *search.SearchService
	embedder      search.EmbeddingProvider
}

// SearchRequest represents a search query request.
type SearchRequest struct {
	Query     string  `json:"query" binding:"required"`
	AccountID *string `json:"account_id,omitempty"`
	Limit     int     `json:"limit,omitempty"`
}

// NewSearchHandler creates a new search handler.
func NewSearchHandler(s *store.Store, embedder search.EmbeddingProvider) *SearchHandler {
	return &SearchHandler{
		store:         s,
		searchService: search.NewSearchService(s.Pool),
		embedder:      embedder,
	}
}

// POST /api/v1/search
func (h *SearchHandler) Search(c *gin.Context) {
	// Check if search is configured
	if h.embedder == nil {
		Error(c, http.StatusServiceUnavailable, "SEARCH_NOT_CONFIGURED", "Search not configured")
		return
	}

	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Validation(c, err.Error())
		return
	}

	// Set default limit
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Generate embedding for the query
	embedding, err := h.embedder.Embed(c.Request.Context(), req.Query)
	if err != nil {
		Internal(c, "Failed to generate embedding: "+err.Error())
		return
	}

	// Search messages using the embedding
	messages, err := h.searchService.SearchMessages(c.Request.Context(), embedding, req.AccountID, req.Limit)
	if err != nil {
		Internal(c, "Search failed: "+err.Error())
		return
	}

	// Convert to []any for pagination
	items := make([]any, len(messages))
	for i, m := range messages {
		items[i] = m
	}

	c.JSON(http.StatusOK, model.NewPaginatedList(items, "", false))
}

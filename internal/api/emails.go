package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ondapile/internal/adapter"
	"ondapile/internal/email"
	"ondapile/internal/model"
	"ondapile/internal/store"
)

type EmailHandler struct {
	store  *store.Store
	emails *email.EmailStore
}

func NewEmailHandler(s *store.Store) *EmailHandler {
	return &EmailHandler{
		store:  s,
		emails: email.NewEmailStore(s),
	}
}

// POST /emails
func (h *EmailHandler) Send(c *gin.Context) {
	var req struct {
		AccountID   string                `json:"account_id" binding:"required"`
		To          []model.EmailAttendee `json:"to" binding:"required,min=1"`
		CC          []model.EmailAttendee `json:"cc"`
		BCC         []model.EmailAttendee `json:"bcc"`
		Subject     string                `json:"subject" binding:"required"`
		BodyHTML    string                `json:"body_html"`
		BodyPlain   string                `json:"body_plain"`
		ReplyToID   *string               `json:"reply_to_email_id"`
		Attachments []struct {
			Filename string `json:"filename"`
			Content  string `json:"content"` // base64
			MimeType string `json:"mime_type"`
		} `json:"attachments"`
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

	// Convert attachments
	var atts []adapter.AttachmentUpload
	for _, a := range req.Attachments {
		atts = append(atts, adapter.AttachmentUpload{
			Filename: a.Filename,
			MimeType: a.MimeType,
		})
	}

	email, err := prov.SendEmail(c.Request.Context(), req.AccountID, adapter.SendEmailRequest{
		To:          req.To,
		CC:          req.CC,
		BCC:         req.BCC,
		Subject:     req.Subject,
		BodyHTML:    req.BodyHTML,
		BodyPlain:   req.BodyPlain,
		ReplyToID:   req.ReplyToID,
		Attachments: atts,
	})
	if err != nil {
		ProviderError(c, "Failed to send email: "+err.Error())
		return
	}

	c.JSON(http.StatusCreated, email)
}

// GET /emails
func (h *EmailHandler) List(c *gin.Context) {
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

	folder := c.DefaultQuery("folder", "INBOX")
	p := GetPagination(c)

	emails, err := prov.ListEmails(c.Request.Context(), accountID, adapter.ListEmailOpts{
		Folder: folder,
		Limit:  p.Limit,
	})
	if err != nil {
		ProviderError(c, "Failed to list emails: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, emails)
}


// GET /emails/:id
func (h *EmailHandler) Get(c *gin.Context) {
	id := c.Param("id")

	// Extract account_id from query or from the email record
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

	email, err := prov.GetEmail(c.Request.Context(), accountID, id)
	if err != nil || email == nil {
		NotFound(c, "Email not found")
		return
	}

	c.JSON(http.StatusOK, email)
}

// PUT /emails/:id
func (h *EmailHandler) Update(c *gin.Context) {
	id := c.Param("id")

	// Verify email exists
	existingEmail, err := h.emails.GetEmail(c.Request.Context(), id)
	if err != nil {
		Internal(c, "Failed to get email")
		return
	}
	if existingEmail == nil {
		NotFound(c, "Email not found")
		return
	}

	var req struct {
		Folder *string `json:"folder"`
		Read   *bool   `json:"read"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Validation(c, err.Error())
		return
	}

	if req.Folder == nil && req.Read == nil {
		Validation(c, "Either folder or read must be provided")
		return
	}

	if err := h.emails.UpdateEmail(c.Request.Context(), id, req.Folder, req.Read); err != nil {
		slog.Error("failed to update email", "error", err, "id", id)
		Internal(c, "Failed to update email: "+err.Error())
		return
	}

	// Reload email to get updated state
	updatedEmail, err := h.emails.GetEmail(c.Request.Context(), id)
	if err != nil {
		Internal(c, "Failed to reload email")
		return
	}

	c.JSON(http.StatusOK, updatedEmail)
}

// DELETE /emails/:id
func (h *EmailHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	// Verify email exists
	existingEmail, err := h.emails.GetEmail(c.Request.Context(), id)
	if err != nil {
		Internal(c, "Failed to get email")
		return
	}
	if existingEmail == nil {
		NotFound(c, "Email not found")
		return
	}

	if err := h.emails.DeleteEmail(c.Request.Context(), id); err != nil {
		Internal(c, "Failed to delete email")
		return
	}

	c.JSON(http.StatusOK, gin.H{"object": "email", "id": id, "deleted": true})
}

// GET /emails/:id/attachments/:att_id
func (h *EmailHandler) DownloadAttachment(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"object":  "error",
		"status":  501,
		"code":    "NOT_IMPLEMENTED",
		"message": "Email attachment download requires IMAP re-fetch — not yet implemented",
	})
}

// GET /emails/folders
func (h *EmailHandler) ListFolders(c *gin.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		Validation(c, "account_id is required")
		return
	}

	// Get counts from store
	counts, err := h.emails.GetFolderCounts(c.Request.Context(), accountID)
	if err != nil {
		Internal(c, "Failed to get folder counts")
		return
	}

	// Define standard folders
	standardFolders := []string{
		model.FolderInbox,
		model.FolderSent,
		model.FolderDrafts,
		model.FolderTrash,
		model.FolderSpam,
		model.FolderArchive,
	}

	type folderResponse struct {
		Name   string `json:"name"`
		Role   string `json:"role"`
		Total  int    `json:"total"`
		Unread int    `json:"unread"`
	}

	var folders []folderResponse
	for _, role := range standardFolders {
		fc := counts[role]
		if fc == nil {
			fc = &email.FolderCount{Role: role, Total: 0, Unread: 0}
		}
		folders = append(folders, folderResponse{
			Name:   role,
			Role:   role,
			Total:  fc.Total,
			Unread: fc.Unread,
		})
	}

	c.JSON(http.StatusOK, folders)
}

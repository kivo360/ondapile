package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Object  string `json:"object"`
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Error codes
const (
	ErrBadRequest    = "BAD_REQUEST"
	ErrUnauthorized  = "UNAUTHORIZED"
	ErrForbidden     = "FORBIDDEN"
	ErrNotFound      = "NOT_FOUND"
	ErrConflict      = "CONFLICT"
	ErrValidation    = "VALIDATION_ERROR"
	ErrRateLimited   = "RATE_LIMITED"
	ErrProviderError = "PROVIDER_ERROR"
	ErrProviderDown  = "PROVIDER_UNAVAILABLE"
	ErrInternal      = "INTERNAL_ERROR"
)

func Error(c *gin.Context, status int, code string, message string) {
	c.JSON(status, ErrorResponse{
		Object:  "error",
		Status:  status,
		Code:    code,
		Message: message,
	})
	c.Abort()
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, ErrBadRequest, message)
}

func Unauthorized(c *gin.Context) {
	Error(c, http.StatusUnauthorized, ErrUnauthorized, "Invalid or missing API key")
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, ErrNotFound, message)
}

func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, ErrConflict, message)
}

func Validation(c *gin.Context, message string) {
	Error(c, http.StatusUnprocessableEntity, ErrValidation, message)
}

func ProviderError(c *gin.Context, message string) {
	Error(c, http.StatusBadGateway, ErrProviderError, message)
}

func ProviderUnavailable(c *gin.Context) {
	Error(c, http.StatusServiceUnavailable, ErrProviderDown, "Provider temporarily unavailable")
}

func Internal(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, ErrInternal, message)
}

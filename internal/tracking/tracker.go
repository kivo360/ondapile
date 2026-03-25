package tracking

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"ondapile/internal/email"
	"ondapile/internal/webhook"
)

// Tracker handles email open and click tracking.
type Tracker struct {
	store      *TrackingStore
	dispatcher *webhook.Dispatcher
	baseURL    string
}

// NewTracker creates a new Tracker instance.
func NewTracker(emailStore *email.EmailStore, dispatcher *webhook.Dispatcher, baseURL string) *Tracker {
	return &Tracker{
		store:      NewTrackingStore(emailStore),
		dispatcher: dispatcher,
		baseURL:    baseURL,
	}
}

// InjectTracking modifies HTML body to add tracking pixel and wrap links.
// Returns modified HTML with tracking elements injected.
func (t *Tracker) InjectTracking(emailID string, htmlBody string) (string, error) {
	// Generate tracking pixel URL
	pixelURL := fmt.Sprintf("%s/t/%s", t.baseURL, emailID)

	// Create tracking pixel image tag
	pixelTag := fmt.Sprintf(`<img src="%s" width="1" height="1" style="display:none" alt="" />`, pixelURL)

	// Inject pixel before closing </body> tag if present, otherwise append at end
	modified := htmlBody
	if strings.Contains(strings.ToLower(htmlBody), "</body>") {
		// Case-insensitive replacement for </body>
		re := regexp.MustCompile(`(?i)</body>`)
		modified = re.ReplaceAllString(htmlBody, pixelTag+"</body>")
	} else {
		// Append at end if no </body> tag
		modified = htmlBody + pixelTag
	}

	// Rewrite all <a href="URL"> to tracking URLs
	// Match href="..." or href='...'
	linkRegex := regexp.MustCompile(`(?i)<a\s+([^>]*?)href=["']([^"']+)["']([^>]*)>`)
	modified = linkRegex.ReplaceAllStringFunc(modified, func(match string) string {
		// Extract the URL from the match
		submatches := linkRegex.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}

		originalURL := submatches[2]
		prefix := submatches[1]
		suffix := submatches[3]

		// Skip if already a tracking URL or javascript/mailto/tel links
		lowerURL := strings.ToLower(originalURL)
		if strings.HasPrefix(lowerURL, "javascript:") ||
			strings.HasPrefix(lowerURL, "mailto:") ||
			strings.HasPrefix(lowerURL, "tel:") ||
			strings.HasPrefix(lowerURL, "#") ||
			strings.Contains(lowerURL, "/t/") ||
			strings.Contains(lowerURL, "/l/") {
			return match
		}

		// Build tracking URL
		trackingURL := fmt.Sprintf("%s/l/%s?url=%s", t.baseURL, emailID, url.QueryEscape(originalURL))

		// Reconstruct the anchor tag with the tracking URL
		return fmt.Sprintf(`<a %shref="%s"%s>`, prefix, trackingURL, suffix)
	})

	return modified, nil
}

package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"ondapile/internal/model"
	"ondapile/internal/store"
)


type Dispatcher struct {
	store      *store.WebhookStore
	httpClient *http.Client
	mu         sync.Mutex
	running    bool
	stopChan   chan struct{}
}

func NewDispatcher(ws *store.WebhookStore) *Dispatcher {
	return &Dispatcher{
		store: ws,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopChan: make(chan struct{}),
	}
}

// Dispatch sends a webhook event to all registered webhooks matching the event type.
func (d *Dispatcher) Dispatch(ctx context.Context, event string, data any) {
	webhooks, err := d.store.ListActiveForEvent(ctx, event)
	if err != nil {
		slog.Error("failed to list webhooks for event", "event", event, "error", err)
		return
	}

	for _, wh := range webhooks {
		deliveryID, err := d.store.CreateDelivery(ctx, wh.ID, event, data)
		if err != nil {
			slog.Error("failed to create webhook delivery", "error", err)
			continue
		}

		go d.deliver(wh, event, data, deliveryID)
	}
}

func (d *Dispatcher) deliver(wh *model.Webhook, event string, data any, deliveryID int64) {
	payload := model.WebhookEvent{
		Event:     event,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal webhook payload", "error", err)
		return
	}

	sig := SignPayload(wh.Secret, body)

	req, err := http.NewRequest("POST", wh.URL, strings.NewReader(string(body)))
	if err != nil {
		slog.Error("failed to create webhook request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Ondapile-Signature", sig)
	req.Header.Set("X-Ondapile-Event", event)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		slog.Error("webhook delivery failed", "url", wh.URL, "error", err)
		d.scheduleRetry(deliveryID)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_ = d.store.MarkDelivered(context.Background(), deliveryID, resp.StatusCode)
		slog.Info("webhook delivered", "url", wh.URL, "event", event, "status", resp.StatusCode)
	} else {
		slog.Warn("webhook delivery returned non-2xx", "url", wh.URL, "status", resp.StatusCode)
		d.scheduleRetry(deliveryID)
	}
}

func (d *Dispatcher) scheduleRetry(deliveryID int64) {
	backoffs := []time.Duration{10 * time.Second, 60 * time.Second, 5 * time.Minute}

	ctx := context.Background()
	deliveries, _ := d.store.ListPendingDeliveries(ctx)
	for _, del := range deliveries {
		if del.ID != deliveryID {
			continue
		}
		if del.Attempts-1 < len(backoffs) {
			nextRetry := time.Now().Add(backoffs[del.Attempts-1])
			_ = d.store.ScheduleRetry(ctx, deliveryID, nextRetry)
		} else {
			// Max retries exceeded, mark as delivered to stop retrying
			_ = d.store.MarkDelivered(ctx, deliveryID, 502)
		}
		break
	}
}

// StartRetryLoop periodically retries failed deliveries.
func (d *Dispatcher) StartRetryLoop(ctx context.Context) {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return
	}
	d.running = true
	d.mu.Unlock()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopChan:
			return
		case <-ticker.C:
			deliveries, err := d.store.ListPendingDeliveries(ctx)
			if err != nil {
				continue
			}
			for _, del := range deliveries {
				wh, err := d.store.GetByID(ctx, del.WebhookID)
				if err != nil || wh == nil {
					continue
				}
				go d.deliver(wh, del.Event, del.Payload, del.ID)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (d *Dispatcher) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.running {
		close(d.stopChan)
		d.running = false
	}
}

// SignPayload creates HMAC-SHA256 signature for webhook payload verification.
func SignPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil)))
}


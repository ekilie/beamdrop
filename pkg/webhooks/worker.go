package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/ekilie/beamdrop/pkg/db"
)

const (
	MaxAttempts    = 8
	BaseDelay      = 2 * time.Second
	MaxDelay       = 15 * time.Minute
	RequestTimeout = 10 * time.Second
	PollInterval   = 5 * time.Second
	BatchSize      = 20
)

// retryableStatusCodes are HTTP status codes that warrant a retry.
var retryableStatusCodes = map[int]bool{
	408: true, // Request Timeout
	425: true, // Too Early
	429: true, // Too Many Requests
	500: true, 502: true, 503: true, 504: true,
}

// Worker processes webhook deliveries in the background.
type Worker struct {
	client  *http.Client
	stopCh  chan struct{}
	stopped bool
	mu      sync.Mutex
}

// NewWorker creates a new webhook delivery worker.
func NewWorker() *Worker {
	return &Worker{
		client: &http.Client{Timeout: RequestTimeout},
		stopCh: make(chan struct{}),
	}
}

// Start begins the background delivery loop.
func (w *Worker) Start() {
	go w.loop()
	slog.Info("Webhook delivery worker started", "poll_interval", PollInterval)
}

// Stop signals the worker to stop.
func (w *Worker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.stopped {
		close(w.stopCh)
		w.stopped = true
		slog.Info("Webhook delivery worker stopped")
	}
}

func (w *Worker) loop() {
	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processBatch()
		}
	}
}

func (w *Worker) processBatch() {
	deliveries, err := db.GetPendingDeliveries(BatchSize)
	if err != nil {
		slog.Error("Webhook worker: failed to get pending deliveries", "error", err)
		return
	}

	for _, d := range deliveries {
		select {
		case <-w.stopCh:
			return
		default:
			w.processDelivery(d)
		}
	}
}

func (w *Worker) processDelivery(delivery db.WebhookDelivery) {
	wh, err := db.GetWebhook(delivery.WebhookID)
	if err != nil || wh == nil {
		slog.Error("Webhook worker: webhook not found", "webhook_id", delivery.WebhookID)
		db.UpdateDelivery(delivery.ID, db.DeliveryDeadLetter, 0, "webhook deleted", 0)
		return
	}

	if !wh.Enabled {
		db.UpdateDelivery(delivery.ID, db.DeliveryDeadLetter, 0, "webhook disabled", 0)
		return
	}

	// Get the event
	event, err := getEvent(delivery.EventID)
	if err != nil || event == nil {
		db.UpdateDelivery(delivery.ID, db.DeliveryDeadLetter, 0, "event not found", 0)
		return
	}

	// Get webhook secret
	secret, err := db.GetWebhookSecret(wh)
	if err != nil {
		slog.Error("Webhook worker: failed to decrypt secret", "webhook_id", wh.ID, "error", err)
		db.UpdateDelivery(delivery.ID, db.DeliveryDeadLetter, 0, "secret decryption failed", 0)
		return
	}

	// Build payload
	payload := buildPayload(event, delivery)

	// Sign and deliver
	start := time.Now()
	statusCode, deliveryErr := w.deliver(wh.TargetURL, secret, event.EventType, delivery.ID, payload)
	durationMs := int(time.Since(start).Milliseconds())

	if deliveryErr == nil && statusCode >= 200 && statusCode < 300 {
		// Success
		db.UpdateDelivery(delivery.ID, db.DeliveryDelivered, statusCode, "", durationMs)
		db.UpdateWebhook(wh.ID, map[string]any{
			"last_delivery_at": time.Now(),
			"last_error":       "",
		})
		slog.Debug("Webhook delivered", "webhook_id", wh.ID, "event", event.EventType, "status", statusCode)
		return
	}

	// Failed
	errMsg := ""
	if deliveryErr != nil {
		errMsg = deliveryErr.Error()
	} else {
		errMsg = fmt.Sprintf("HTTP %d", statusCode)
	}

	attemptCount := delivery.AttemptCount + 1

	if attemptCount >= MaxAttempts || (statusCode > 0 && !retryableStatusCodes[statusCode]) {
		// Dead letter
		db.UpdateDelivery(delivery.ID, db.DeliveryDeadLetter, statusCode, errMsg, durationMs)
		db.UpdateWebhook(wh.ID, map[string]any{"last_error": errMsg})
		slog.Warn("Webhook delivery dead-lettered", "webhook_id", wh.ID, "event", event.EventType, "attempts", attemptCount)
		return
	}

	// Schedule retry with exponential backoff + jitter
	delay := backoffDelay(attemptCount)
	nextAt := time.Now().Add(delay)

	db.UpdateDelivery(delivery.ID, db.DeliveryFailed, statusCode, errMsg, durationMs)
	db.SetDeliveryNextAttempt(delivery.ID, nextAt)
	db.UpdateWebhook(wh.ID, map[string]any{"last_error": errMsg})

	slog.Debug("Webhook delivery scheduled for retry",
		"webhook_id", wh.ID, "attempt", attemptCount, "next_at", nextAt)
}

func (w *Worker) deliver(targetURL, secret, eventType string, deliveryID uint, payload []byte) (int, error) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	deliveryIDStr := fmt.Sprintf("%d", deliveryID)

	// Signature: HMAC-SHA256(timestamp + "\n" + delivery_id + "\n" + body)
	sigBase := timestamp + "\n" + deliveryIDStr + "\n" + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sigBase))
	signature := "v1=" + hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Beamdrop-Webhook/1.0")
	req.Header.Set("X-Beamdrop-Webhook-Id", fmt.Sprintf("%d", deliveryID))
	req.Header.Set("X-Beamdrop-Event", eventType)
	req.Header.Set("X-Beamdrop-Delivery-Id", deliveryIDStr)
	req.Header.Set("X-Beamdrop-Timestamp", timestamp)
	req.Header.Set("X-Beamdrop-Signature", signature)

	resp, err := w.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func buildPayload(event *db.WebhookEvent, delivery db.WebhookDelivery) []byte {
	payload := map[string]any{
		"event_id":      event.ID,
		"event_type":    event.EventType,
		"resource_type": event.ResourceType,
		"resource_path": event.ResourcePath,
		"actor":         event.Actor,
		"created_at":    event.CreatedAt.UTC().Format(time.RFC3339),
		"attempt":       delivery.AttemptCount + 1,
	}

	// Parse payload JSON if present
	if event.PayloadJSON != "" {
		var data map[string]any
		if json.Unmarshal([]byte(event.PayloadJSON), &data) == nil {
			payload["data"] = data
		}
	}

	out, _ := json.Marshal(payload)
	return out
}

// backoffDelay calculates exponential backoff with jitter.
// delay = min(base * 2^attempt, max) + jitter
func backoffDelay(attempt int) time.Duration {
	delay := float64(BaseDelay) * math.Pow(2, float64(attempt))
	if delay > float64(MaxDelay) {
		delay = float64(MaxDelay)
	}
	jitter := time.Duration(rand.Int63n(int64(2 * time.Second)))
	return time.Duration(delay) + jitter
}

func getEvent(eventID string) (*db.WebhookEvent, error) {
	d := db.GetDB()
	var event db.WebhookEvent
	if err := d.Where("id = ?", eventID).First(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

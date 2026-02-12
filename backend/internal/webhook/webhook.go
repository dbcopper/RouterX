package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"routerx/internal/store"
)

// Dispatcher sends webhook events to registered endpoints.
type Dispatcher struct {
	Store  *store.Store
	Client *http.Client
}

func New(st *store.Store) *Dispatcher {
	return &Dispatcher{
		Store:  st,
		Client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Event represents a webhook payload.
type Event struct {
	Type      string      `json:"type"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// Fire sends an event to all enabled webhooks matching the event type.
// It runs asynchronously and does not block.
func (d *Dispatcher) Fire(ctx context.Context, eventType string, data interface{}) {
	hooks, err := d.Store.GetEnabledWebhooks(ctx, eventType)
	if err != nil || len(hooks) == 0 {
		return
	}
	event := Event{
		Type:      eventType,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      data,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return
	}
	for _, hook := range hooks {
		go d.send(hook, body)
	}
}

func (d *Dispatcher) send(hook store.Webhook, body []byte) {
	req, err := http.NewRequest("POST", hook.URL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "RouterX-Webhook/1.0")
	if hook.Secret != "" {
		mac := hmac.New(sha256.New, []byte(hook.Secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-RouterX-Signature", sig)
	}
	resp, err := d.Client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

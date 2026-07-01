package notify

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Webhook posts alerts as JSON to a single URL. The payload carries both
// "text" (Slack) and "content" (Discord) so it works with either without
// extra config; the structured fields are there for custom consumers.
type Webhook struct {
	url    string
	client *http.Client
}

func NewWebhook(url string) *Webhook {
	return &Webhook{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

type webhookPayload struct {
	Text    string `json:"text"`
	Content string `json:"content"`
	Level   int    `json:"level"`
	Title   string `json:"title"`
	Body    string `json:"body"`
}

// Notify is fire-and-forget: a failing webhook is logged, never returned, so
// it can't take the monitor down with it.
func (w *Webhook) Notify(level int, title, content string) {
	message := title + "\n" + content

	payload := webhookPayload{
		Text:    message,
		Content: message,
		Level:   level,
		Title:   title,
		Body:    content,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("webhook: marshal: %v", err)
		return
	}

	resp, err := w.client.Post(w.url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("webhook: post: %v", err)
		return
	}
	resp.Body.Close()
}

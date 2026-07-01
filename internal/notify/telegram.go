package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// Telegram sends alerts to a chat via the Bot API. It only sends (no command
// handling), so a single sendMessage call is all it needs.
type Telegram struct {
	token  string
	chatID string
	client *http.Client
}

func NewTelegram(token, chatID string) *Telegram {
	// Force IPv4: Railway's IPv6 egress to api.telegram.org can black-hole,
	// making the request hang until the timeout. Dialing tcp4 avoids that.
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp4", addr)
		},
	}

	return &Telegram{
		token:  token,
		chatID: chatID,
		client: &http.Client{Timeout: 10 * time.Second, Transport: transport},
	}
}

type telegramPayload struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

// Notify is fire-and-forget: a failing send is logged, never returned, so it
// can't take the monitor down with it.
func (t *Telegram) Notify(level int, title, content string) {
	text := fmt.Sprintf("%s %s\n%s", levelIcon(level), title, content)

	body, err := json.Marshal(telegramPayload{ChatID: t.chatID, Text: text})
	if err != nil {
		log.Printf("telegram: marshal: %v", err)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)
	resp, err := t.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("telegram: post: %v", err)
		return
	}
	resp.Body.Close()
}

func levelIcon(level int) string {
	switch {
	case level >= LevelCritical:
		return "\U0001F534" // red circle
	case level <= LevelInfo:
		return "\U0001F7E2" // green circle
	default:
		return "\U0001F7E0" // orange circle
	}
}

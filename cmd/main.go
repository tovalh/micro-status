package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/tovalh/micro-status/internal/config"
	"github.com/tovalh/micro-status/internal/monitor"
	"github.com/tovalh/micro-status/internal/notify"
)

func main() {
	// Load .env if present; in production the vars come from the environment.
	_ = godotenv.Load()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("could not load config: %v", err)
	}

	// Console is always on. The webhook is added only when configured.
	notifiers := []monitor.Notifier{
		notify.NewConsole(),
	}
	if cfg.Webhook.URL != "" {
		notifiers = append(notifiers, notify.NewWebhook(cfg.Webhook.URL))
	}
	if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token != "" {
		notifiers = append(notifiers, notify.NewTelegram(token, os.Getenv("TELEGRAM_CHAT_ID")))
	}

	// Announce startup on every channel before the first scan.
	for _, n := range notifiers {
		n.Notify(notify.LevelInfo, "Monitor up",
			fmt.Sprintf("Watching %d service(s). Scanning...", len(cfg.Services)))
	}

	monitor.Run(cfg.Services, config.LoadTiming(), notifiers)
}

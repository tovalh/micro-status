// Package notify holds the alert channels the monitor fans out to. Every
// channel satisfies the same one-method interface.
package notify

import (
	"fmt"
	"time"
)

// Alert severity levels.
const (
	LevelInfo     = 1
	LevelCritical = 3
)

// Console prints alerts to stdout.
type Console struct{}

func NewConsole() *Console { return &Console{} }

func (c *Console) Notify(level int, title, content string) {
	fmt.Printf("\n[ALERT %s] %s\n  %s - %s\n\n",
		levelLabel(level), title, time.Now().Format("15:04:05"), content)
}

func levelLabel(level int) string {
	switch {
	case level >= LevelCritical:
		return "CRITICAL"
	case level <= LevelInfo:
		return "INFO"
	default:
		return "WARN"
	}
}

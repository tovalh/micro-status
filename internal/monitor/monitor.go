package monitor

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"health_status/internal/config"
	"health_status/internal/notify"
	"health_status/internal/service"
)

// Notifier is any channel that can deliver an alert (console, webhook, ...).
type Notifier interface {
	Notify(level int, title, content string)
}

// Run starts one goroutine per service and blocks forever. It only acts when
// a service's state changes. When a service declares a dependencies URL, that
// endpoint is polled too while the service is up.
func Run(services []config.Service, timing config.Timing, notifiers []Notifier) {
	client := &http.Client{Timeout: timing.RequestTimeout}

	// Last known state per service ("up", "down" or "unreachable"). Several
	// goroutines touch it at once, so mu guards it.
	states := map[string]string{}
	var mu sync.Mutex

	for _, svc := range services {
		svc := svc
		go watch(svc, client, timing, notifiers, states, &mu)
	}

	select {}
}

// watch polls a single service until the process exits.
func watch(svc config.Service, client *http.Client, timing config.Timing, notifiers []Notifier, states map[string]string, mu *sync.Mutex) {
	ticker := time.NewTicker(svc.Interval)
	defer ticker.Stop()

	// Last known dependency picture for this service, kept across ticks.
	depState := ""

	for range ticker.C {
		status, err := service.Check(svc.URL, client)

		current := status.State
		if err != nil {
			current = "unreachable"
		}

		fmt.Printf("[%s] %s -> %s (%d) latency %s\n",
			time.Now().Format("15:04:05"), svc.Name, current, status.StatusCode, status.Latency)

		// While the service is up, also poll its dependencies (if any).
		if current == "up" && svc.DependenciesURL != "" {
			depState = checkDependencies(svc.Name, svc.DependenciesURL, client, depState, notifiers)
		}

		mu.Lock()
		prev := states[svc.Name]
		mu.Unlock()

		if current == prev {
			continue
		}

		// On startup prev is "": a healthy service is just the baseline (stay
		// quiet), but a broken one still alerts.
		firstSeen := prev == ""
		if firstSeen && current == "up" {
			mu.Lock()
			states[svc.Name] = current
			mu.Unlock()
			continue
		}

		// A change. Re-check before alerting to filter blips: a fall must stay
		// down, a recovery must stay up. If it doesn't hold, it was a flap:
		// leave the state as-is and don't alert.
		wantUp := current == "up"
		attempts, delay := timing.DownAttempts, timing.DownDelay
		if wantUp {
			attempts, delay = timing.UpAttempts, timing.UpDelay
		}
		if !confirm(svc.URL, client, wantUp, attempts, delay) {
			fmt.Printf("  ~~ %s: %s not confirmed (flap), skipping alert\n", svc.Name, current)
			continue
		}

		mu.Lock()
		states[svc.Name] = current
		mu.Unlock()

		switch {
		case current == "up" && !firstSeen:
			fmt.Printf("  >> CHANGE %s: %s -> up | sending recovery alert\n", svc.Name, prev)
			notifyAll(notifiers, notify.LevelInfo, svc.Name+" recovered",
				fmt.Sprintf("%s is responding OK again (%d)", svc.Name, status.StatusCode))
		case current == "down":
			fmt.Printf("  >> CHANGE %s: %s -> down | sending outage alert\n", svc.Name, prev)
			notifyAll(notifiers, notify.LevelCritical, svc.Name+" down",
				fmt.Sprintf("%s responded with status %d", svc.Name, status.StatusCode))
		case current == "unreachable":
			fmt.Printf("  >> CHANGE %s: %s -> unreachable | sending outage alert\n", svc.Name, prev)
			notifyAll(notifiers, notify.LevelCritical, svc.Name+" unreachable",
				fmt.Sprintf("Could not reach %s: %v", svc.Name, err))
		}
	}
}

// checkDependencies polls a service's dependency endpoint and alerts only when
// the picture changes (same baseline rule as the services). It returns the new
// state so the next tick can compare against it. Three cases are handled:
// network error, a non-200 response, or the { name: state } map itself.
func checkDependencies(svcName, url string, client *http.Client, prev string, notifiers []Notifier) string {
	deps, code, err := service.CheckDependencies(url, client)

	// Collapse the current picture into a single string to compare against the
	// previous one.
	var current, title, content string
	switch {
	case err != nil:
		current = "endpoint-error"
		title = svcName + " dependencies unreachable"
		content = "Could not reach the dependencies endpoint: " + err.Error()
	case code != 200:
		current = fmt.Sprintf("http-%d", code)
		title = svcName + " dependencies endpoint error"
		content = fmt.Sprintf("Dependencies endpoint responded with status %d", code)
	default:
		// Walk the map in a stable order so the log is readable, and collect
		// anything that isn't "ok".
		names := make([]string, 0, len(deps))
		for name := range deps {
			names = append(names, name)
		}
		sort.Strings(names)

		var down []string
		for _, name := range names {
			state := deps[name]
			fmt.Printf("  [dep] %s/%s: %s\n", svcName, name, state)
			if state != "ok" {
				down = append(down, name)
			}
		}

		if len(down) == 0 {
			current = "ok"
			title = svcName + " dependencies recovered"
			content = "All dependencies are responding OK"
		} else {
			current = strings.Join(down, ", ")
			title = svcName + " dependencies down"
			content = "Dependencies down: " + current
		}
	}

	if current == prev {
		return prev
	}
	// Baseline on startup: if everything is OK, stay quiet.
	if prev == "" && current == "ok" {
		return current
	}

	level := notify.LevelCritical
	if current == "ok" {
		level = notify.LevelInfo
	}
	fmt.Printf("  >> CHANGE %s deps: %s -> %s | sending alert\n", svcName, prev, current)
	notifyAll(notifiers, level, title, content)
	return current
}

// confirm re-checks the service `attempts` more times, waiting `delay` between
// checks. It returns true only if every re-check keeps matching wantUp; it
// bails out on the first disagreement, so a transient blip never gets
// confirmed.
func confirm(url string, client *http.Client, wantUp bool, attempts int, delay time.Duration) bool {
	for i := 0; i < attempts; i++ {
		time.Sleep(delay)

		status, err := service.Check(url, client)
		isUp := err == nil && status.State == "up"
		if isUp != wantUp {
			return false
		}
	}
	return true
}

// notifyAll fans the alert out to every configured channel.
func notifyAll(notifiers []Notifier, level int, title, content string) {
	for _, n := range notifiers {
		n.Notify(level, title, content)
	}
}

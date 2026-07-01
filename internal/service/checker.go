package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Status struct {
	State      string
	Latency    time.Duration
	StatusCode int
}

func Check(url string, client *http.Client) (Status, error) {
	start := time.Now()
	resp, err := client.Get(url)
	latency := time.Since(start)
	if err != nil {
		// Couldn't reach the service (network error or timeout).
		return Status{}, err
	}
	defer resp.Body.Close()

	// The service answered. A 4xx/5xx is a valid "I'm unhealthy" reply,
	// not an error, so it goes in the Status.
	state := "up"
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		state = "down"
	}
	return Status{State: state, Latency: latency, StatusCode: resp.StatusCode}, nil
}

// CheckDependencies hits a service's dependency health endpoint. It returns
// the name -> state map ("ok" / anything else) only on a 200. The status code
// is always returned so the caller can tell apart the three cases: network
// error (err != nil), non-200 (code != 200), or a valid map.
func CheckDependencies(url string, client *http.Client) (map[string]string, int, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, resp.StatusCode, nil
	}

	var deps map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&deps); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decoding dependencies: %w", err)
	}
	return deps, resp.StatusCode, nil
}

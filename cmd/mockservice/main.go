// Command mockservice is a tiny HTTP service used to exercise the monitor end
// to end. It exposes a health endpoint, a dependencies endpoint, and a toggle
// that flips it between healthy and failing so an outage can be simulated.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
)

type server struct {
	mu      sync.Mutex
	healthy bool // controls /health
	redisOK bool // controls the redis dependency, independent of /health
}

func (s *server) isHealthy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.healthy
}

func (s *server) toggleHealth() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthy = !s.healthy
	return s.healthy
}

func (s *server) toggleRedis() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.redisOK = !s.redisOK
	return s.redisOK
}

func (s *server) dependencies() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	redis := "ok"
	if !s.redisOK {
		redis = "error"
	}
	return map[string]string{
		"postgres": "ok",
		"redis":    redis,
	}
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if s.isHealthy() {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte("unhealthy"))
}

func (s *server) handleDependencies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.dependencies())
}

func (s *server) handleToggle(w http.ResponseWriter, r *http.Request) {
	if s.toggleHealth() {
		w.Write([]byte("service is now UP\n"))
	} else {
		w.Write([]byte("service is now DOWN\n"))
	}
}

func (s *server) handleToggleRedis(w http.ResponseWriter, r *http.Request) {
	if s.toggleRedis() {
		w.Write([]byte("redis dependency is now OK\n"))
	} else {
		w.Write([]byte("redis dependency is now ERROR\n"))
	}
}

func main() {
	addr := os.Getenv("MOCK_ADDR")
	if addr == "" {
		addr = ":8081"
	}

	s := &server{healthy: true, redisOK: true}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/health/dependencies", s.handleDependencies)
	mux.HandleFunc("/toggle", s.handleToggle)
	mux.HandleFunc("/toggle/redis", s.handleToggleRedis)

	log.Printf("mock service listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

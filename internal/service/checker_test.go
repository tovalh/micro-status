package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name      string
		code      int
		wantState string
	}{
		{"200 is up", http.StatusOK, "up"},
		{"204 is up", http.StatusNoContent, "up"},
		{"503 is down", http.StatusServiceUnavailable, "down"},
		{"404 is down", http.StatusNotFound, "down"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.code)
			}))
			defer srv.Close()

			got, err := Check(srv.URL, srv.Client())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.State != tt.wantState {
				t.Errorf("state = %q, want %q", got.State, tt.wantState)
			}
			if got.StatusCode != tt.code {
				t.Errorf("status = %d, want %d", got.StatusCode, tt.code)
			}
		})
	}
}

func TestCheckUnreachable(t *testing.T) {
	// Point at a server that's already closed so the request can't connect.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	if _, err := Check(url, srv.Client()); err == nil {
		t.Fatal("expected an error reaching a closed server, got nil")
	}
}

func TestCheckDependencies(t *testing.T) {
	t.Run("200 returns the map", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"postgres":"ok","redis":"error"}`))
		}))
		defer srv.Close()

		deps, code, err := CheckDependencies(srv.URL, srv.Client())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != 200 {
			t.Errorf("code = %d, want 200", code)
		}
		if deps["postgres"] != "ok" || deps["redis"] != "error" {
			t.Errorf("unexpected deps: %v", deps)
		}
	})

	t.Run("non-200 returns code, no map", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer srv.Close()

		deps, code, err := CheckDependencies(srv.URL, srv.Client())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != http.StatusBadGateway {
			t.Errorf("code = %d, want %d", code, http.StatusBadGateway)
		}
		if deps != nil {
			t.Errorf("deps = %v, want nil", deps)
		}
	})

	t.Run("bad JSON on 200 is an error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not json"))
		}))
		defer srv.Close()

		if _, _, err := CheckDependencies(srv.URL, srv.Client()); err == nil {
			t.Fatal("expected a decode error, got nil")
		}
	})
}

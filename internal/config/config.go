package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Services []Service
	Webhook  Webhook
}

// Webhook is an optional generic alert channel (e.g. a Slack or Discord
// incoming webhook). The monitor only uses it when URL is non-empty.
type Webhook struct {
	URL string `yaml:"url"`
}

type Service struct {
	Name     string
	URL      string
	Interval time.Duration

	// DependenciesURL is optional. When set, the monitor polls it while the
	// service is up and expects a JSON map of { name: "ok" | <anything else> }.
	// It alerts when that picture changes (e.g. a database goes down even
	// though the service itself still answers).
	DependenciesURL string
}

// rawService mirrors the YAML. Interval is a string ("3s") because YAML
// can't turn that into a time.Duration on its own.
type rawService struct {
	Name            string `yaml:"name"`
	URL             string `yaml:"url"`
	Interval        string `yaml:"interval"`
	DependenciesURL string `yaml:"dependencies_url"`
}

// Load reads the YAML file and returns the validated config.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading %s: %w", path, err)
	}

	var raw struct {
		Services []rawService `yaml:"services"`
		Webhook  Webhook      `yaml:"webhook"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parsing YAML: %w", err)
	}

	if len(raw.Services) == 0 {
		return Config{}, fmt.Errorf("config defines no services")
	}

	services := make([]Service, 0, len(raw.Services))
	for i, s := range raw.Services {
		if s.Name == "" || s.URL == "" {
			return Config{}, fmt.Errorf("service #%d: name and url are required", i+1)
		}
		interval, err := time.ParseDuration(s.Interval)
		if err != nil {
			return Config{}, fmt.Errorf("service %q: invalid interval %q: %w", s.Name, s.Interval, err)
		}
		services = append(services, Service{
			Name:            s.Name,
			URL:             s.URL,
			Interval:        interval,
			DependenciesURL: s.DependenciesURL,
		})
	}

	return Config{Services: services, Webhook: raw.Webhook}, nil
}

// Timing holds the tuning knobs, read from the environment.
type Timing struct {
	RequestTimeout time.Duration
	DownAttempts   int
	DownDelay      time.Duration
	UpAttempts     int
	UpDelay        time.Duration
}

// LoadTiming reads the timing knobs from the environment, falling back to
// defaults when a variable is unset or unparseable.
func LoadTiming() Timing {
	return Timing{
		RequestTimeout: durationEnv("REQUEST_TIMEOUT", 5*time.Second),
		DownAttempts:   intEnv("DOWN_CONFIRM_ATTEMPTS", 2),
		DownDelay:      durationEnv("DOWN_CONFIRM_DELAY", 1*time.Second),
		UpAttempts:     intEnv("UP_CONFIRM_ATTEMPTS", 1),
		UpDelay:        durationEnv("UP_CONFIRM_DELAY", 10*time.Second),
	}
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	if d, err := time.ParseDuration(os.Getenv(key)); err == nil && d > 0 {
		return d
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	if n, err := strconv.Atoi(os.Getenv(key)); err == nil && n >= 0 {
		return n
	}
	return fallback
}

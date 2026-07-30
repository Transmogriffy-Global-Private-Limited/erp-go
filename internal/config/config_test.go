package config

import (
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := Load(mapLookup(nil))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPHost != DefaultHTTPHost {
		t.Fatalf("HTTPHost = %q, want %q", cfg.HTTPHost, DefaultHTTPHost)
	}
	if cfg.HTTPPort != DefaultHTTPPort {
		t.Fatalf("HTTPPort = %d, want %d", cfg.HTTPPort, DefaultHTTPPort)
	}
	if cfg.APIDocsEnabled {
		t.Fatal("APIDocsEnabled = true, want false")
	}
	if cfg.Address() != "127.0.0.1:8080" {
		t.Fatalf("Address() = %q, want %q", cfg.Address(), "127.0.0.1:8080")
	}
}

func TestLoadExplicitValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		environment map[string]string
		wantHost    string
		wantAddress string
	}{
		{
			name: "IPv4 loopback",
			environment: map[string]string{
				"HTTP_HOST":        "127.0.0.1",
				"HTTP_PORT":        "18180",
				"API_DOCS_ENABLED": "TRUE",
			},
			wantHost:    "127.0.0.1",
			wantAddress: "127.0.0.1:18180",
		},
		{
			name: "IPv6 loopback",
			environment: map[string]string{
				"HTTP_HOST":        "::1",
				"HTTP_PORT":        "8081",
				"API_DOCS_ENABLED": "true",
			},
			wantHost:    "::1",
			wantAddress: "[::1]:8081",
		},
		{
			name: "localhost",
			environment: map[string]string{
				"HTTP_HOST":        "LOCALHOST",
				"HTTP_PORT":        "8082",
				"API_DOCS_ENABLED": "false",
			},
			wantHost:    "127.0.0.1",
			wantAddress: "127.0.0.1:8082",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := Load(mapLookup(test.environment))
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.HTTPHost != test.wantHost {
				t.Fatalf("HTTPHost = %q, want %q", cfg.HTTPHost, test.wantHost)
			}
			if cfg.Address() != test.wantAddress {
				t.Fatalf("Address() = %q, want %q", cfg.Address(), test.wantAddress)
			}
		})
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		environment map[string]string
		wantError   string
	}{
		{name: "empty host", environment: map[string]string{"HTTP_HOST": ""}, wantError: "HTTP_HOST must not be empty"},
		{name: "IPv4 wildcard", environment: map[string]string{"HTTP_HOST": "0.0.0.0"}, wantError: "HTTP_HOST must be a loopback host"},
		{name: "IPv6 wildcard", environment: map[string]string{"HTTP_HOST": "::"}, wantError: "HTTP_HOST must be a loopback host"},
		{name: "external host", environment: map[string]string{"HTTP_HOST": "192.0.2.10"}, wantError: "HTTP_HOST must be a loopback host"},
		{name: "non-numeric port", environment: map[string]string{"HTTP_PORT": "http"}, wantError: "HTTP_PORT must be an integer from 1 to 65535"},
		{name: "zero port", environment: map[string]string{"HTTP_PORT": "0"}, wantError: "HTTP_PORT must be an integer from 1 to 65535"},
		{name: "large port", environment: map[string]string{"HTTP_PORT": "65536"}, wantError: "HTTP_PORT must be an integer from 1 to 65535"},
		{name: "invalid docs toggle", environment: map[string]string{"API_DOCS_ENABLED": "yes"}, wantError: "API_DOCS_ENABLED must be true or false"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := Load(mapLookup(test.environment))
			if err == nil {
				t.Fatal("Load() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("Load() error = %q, want it to contain %q", err, test.wantError)
			}
		})
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, found := values[name]
		return value, found
	}
}

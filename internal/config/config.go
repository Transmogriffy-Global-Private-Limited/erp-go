package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultHTTPHost = "127.0.0.1"
	DefaultHTTPPort = 8080
)

// Config contains the complete Step 01A runtime configuration.
type Config struct {
	HTTPHost       string
	HTTPPort       uint16
	APIDocsEnabled bool
}

// Address returns the validated HTTP listen address.
func (c Config) Address() string {
	return net.JoinHostPort(c.HTTPHost, strconv.Itoa(int(c.HTTPPort)))
}

// LoadFromEnv loads Config from the current process environment.
func LoadFromEnv() (Config, error) {
	return Load(os.LookupEnv)
}

// Load loads Config through lookup so configuration behavior is deterministic
// and directly testable.
func Load(lookup func(string) (string, bool)) (Config, error) {
	host, err := loadHost(lookup)
	if err != nil {
		return Config{}, err
	}

	port, err := loadPort(lookup)
	if err != nil {
		return Config{}, err
	}

	docsEnabled, err := loadBool(lookup, "API_DOCS_ENABLED", false)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPHost:       host,
		HTTPPort:       port,
		APIDocsEnabled: docsEnabled,
	}, nil
}

func loadHost(lookup func(string) (string, bool)) (string, error) {
	host, found := lookup("HTTP_HOST")
	if !found {
		host = DefaultHTTPHost
	}

	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("HTTP_HOST must not be empty")
	}

	if strings.EqualFold(host, "localhost") {
		return DefaultHTTPHost, nil
	}

	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return "", fmt.Errorf(
			"HTTP_HOST must be a loopback host (127.0.0.1, ::1, or localhost), got %q",
			host,
		)
	}

	return ip.String(), nil
}

func loadPort(lookup func(string) (string, bool)) (uint16, error) {
	value, found := lookup("HTTP_PORT")
	if !found {
		return DefaultHTTPPort, nil
	}

	value = strings.TrimSpace(value)
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("HTTP_PORT must be an integer from 1 to 65535, got %q", value)
	}

	return uint16(port), nil
}

func loadBool(
	lookup func(string) (string, bool),
	name string,
	defaultValue bool,
) (bool, error) {
	value, found := lookup(name)
	if !found {
		return defaultValue, nil
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be true or false, got %q", name, value)
	}
}

package app

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/config"
)

func TestServeStopsAfterContextCancellation(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)

	go func() {
		result <- Serve(
			ctx,
			config.Config{HTTPHost: "127.0.0.1", HTTPPort: 8080},
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			listener,
		)
	}()

	client := &http.Client{Timeout: time.Second}
	url := "http://" + listener.Addr().String() + "/healthz"

	deadline := time.Now().Add(2 * time.Second)
	for {
		response, requestErr := client.Get(url)
		if requestErr == nil {
			_ = response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("GET /healthz status = %d, want %d", response.StatusCode, http.StatusOK)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not become healthy: %v", requestErr)
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not stop after context cancellation")
	}

	_, err = client.Get(url)
	if err == nil {
		t.Fatal("GET /healthz succeeded after shutdown")
	}
}

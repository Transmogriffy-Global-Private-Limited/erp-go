package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/config"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/httpapi"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// Run listens on the configured loopback address and serves until ctx is
// cancelled or the server fails.
func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	listener, err := net.Listen("tcp", cfg.Address())
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Address(), err)
	}

	return Serve(ctx, cfg, logger, listener)
}

// Serve owns the HTTP server lifecycle for an already-created listener. The
// listener form keeps shutdown behavior deterministic in tests.
func Serve(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
	listener net.Listener,
) error {
	if logger == nil {
		logger = slog.Default()
	}

	server := &http.Server{
		Handler:           httpapi.NewHandler(cfg),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	serveErr := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveErr <- err
	}()

	logger.Info(
		"erp-api listening",
		"address", listener.Addr().String(),
		"api_docs_enabled", cfg.APIDocsEnabled,
	)

	select {
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return fmt.Errorf("graceful HTTP shutdown: %w", err)
	}

	if err := <-serveErr; err != nil {
		return fmt.Errorf("serve HTTP during shutdown: %w", err)
	}

	logger.Info("erp-api stopped")
	return nil
}

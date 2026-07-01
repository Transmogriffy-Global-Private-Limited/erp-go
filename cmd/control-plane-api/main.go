package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/controlapi"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := env("CONTROL_PLANE_HTTP_ADDR", ":8081")
	databaseURL := env("CONTROL_PLANE_DATABASE_URL", env("ERP_DATABASE_URL", ""))

	pool, err := db.Open(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	server := &http.Server{
		Addr:              addr,
		Handler:           logRequests(controlapi.NewHandler(pool)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("control-plane-api listening on %s", addr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func env(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

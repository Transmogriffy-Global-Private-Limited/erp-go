package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/inventory"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	platformmodules "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/modules"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/rbac"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	db             *pgxpool.Pool
	modules        platformmodules.Store
	rbac           rbac.Store
	inventory      inventory.Store
	tenantSessions auth.TenantSessionStore
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := env("ERP_HTTP_ADDR", ":8080")
	databaseURL := env("ERP_DATABASE_URL", "")

	pool, err := db.Open(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	app := &app{
		db:             pool,
		modules:        platformmodules.NewStore(pool),
		rbac:           rbac.NewStore(pool),
		inventory:      inventory.NewStore(pool),
		tenantSessions: auth.NewTenantSessionStore(pool),
	}

	handler := app.routes()

	server := &http.Server{
		Addr:              addr,
		Handler:           logRequests(handler),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("erp-api listening on %s", addr)

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

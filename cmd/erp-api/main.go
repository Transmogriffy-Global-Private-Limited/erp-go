package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	db *pgxpool.Pool
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
		db: pool,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", app.healthHandler)
	mux.HandleFunc("/healthz/db", app.dbHealthHandler)
	mux.Handle("/api/v1/modules", tenantMiddleware(http.HandlerFunc(app.enabledModulesHandler)))

	server := &http.Server{
		Addr:              addr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("erp-api listening on %s", addr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func (a *app) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"service": "erp-api",
		"status":  "ok",
	})
}

func (a *app) dbHealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}

	if err := db.Ping(r.Context(), a.db); err != nil {
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]any{
			"service": "erp-api",
			"status":  "db_unavailable",
			"error":   err.Error(),
		})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"service": "erp-api",
		"status":  "db_ok",
	})
}

func (a *app) enabledModulesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}

	tenantID, _ := tenancy.TenantIDFromContext(r.Context())

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"modules": []map[string]any{
			{"id": "inventory", "name": "Inventory", "status": "planned"},
			{"id": "sales", "name": "Sales", "status": "planned"},
			{"id": "purchase", "name": "Purchase", "status": "planned"},
			{"id": "accounting", "name": "Accounting", "status": "planned"},
		},
	})
}

func tenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			httpx.Error(w, http.StatusBadRequest, "tenant_required", "X-Tenant-ID header is required")
			return
		}

		ctx := tenancy.WithTenantID(r.Context(), tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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

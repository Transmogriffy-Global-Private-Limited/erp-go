package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
)

func main() {
	port := env("PORT", "8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.Handle("/api/v1/modules", tenantMiddleware(http.HandlerFunc(enabledModulesHandler)))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("erp-api listening on :%s", port)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"service": "erp-api",
		"status":  "ok",
	})
}

func enabledModulesHandler(w http.ResponseWriter, r *http.Request) {
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

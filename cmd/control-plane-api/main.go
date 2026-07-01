package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
)

func main() {
	port := env("PORT", "8081")

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/control/v1/tenants", tenantsHandler)
	mux.HandleFunc("/control/v1/modules", modulesHandler)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("control-plane-api listening on :%s", port)

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
		"service": "control-plane-api",
		"status":  "ok",
	})
}

func tenantsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		httpx.JSON(w, http.StatusOK, map[string]any{
			"tenants": []any{},
		})
	case http.MethodPost:
		httpx.JSON(w, http.StatusCreated, map[string]any{
			"message": "tenant creation endpoint reserved",
		})
	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func modulesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"modules": []map[string]any{
			{"id": "inventory", "name": "Inventory", "status": "planned"},
			{"id": "sales", "name": "Sales", "status": "planned"},
			{"id": "purchase", "name": "Purchase", "status": "planned"},
			{"id": "accounting", "name": "Accounting", "status": "planned"},
		},
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

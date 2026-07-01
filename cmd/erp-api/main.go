package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/inventory"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	platformmodules "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/modules"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/rbac"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
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

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", app.tenantLoginHandler)
	mux.HandleFunc("/api/v1/auth/me", app.tenantMeHandler)
	mux.HandleFunc("/api/v1/auth/logout", app.tenantLogoutHandler)
	mux.HandleFunc("/healthz", app.healthHandler)
	mux.HandleFunc("/healthz/db", app.dbHealthHandler)
	mux.Handle("/api/v1/modules", tenantMiddleware(http.HandlerFunc(app.enabledModulesHandler)))
	mux.HandleFunc("/api/v1/inventory/items", app.inventoryItemsSessionHandler)

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

	modules, err := a.modules.ListEnabledForTenant(r.Context(), tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "tenant_modules_query_failed", "failed to load tenant modules")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"modules":   modules,
	})
}

func (a *app) inventoryItemsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.requirePermission(w, r, "inventory.item.read") {
			return
		}
		a.listInventoryItems(w, r)

	case http.MethodPost:
		if !a.requirePermission(w, r, "inventory.item.write") {
			return
		}
		a.createInventoryItem(w, r)

	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *app) listInventoryItems(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())

	items, err := a.inventory.ListItems(r.Context(), tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "inventory_items_query_failed", "failed to load inventory items")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"items":     items,
	})
}

func (a *app) createInventoryItem(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())

	var input inventory.CreateItemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	input.SKU = strings.TrimSpace(input.SKU)
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Status = strings.TrimSpace(input.Status)

	if input.SKU == "" {
		httpx.Error(w, http.StatusBadRequest, "sku_required", "sku is required")
		return
	}

	if input.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "name_required", "name is required")
		return
	}

	if input.Status != "" && input.Status != "active" && input.Status != "inactive" && input.Status != "archived" {
		httpx.Error(w, http.StatusBadRequest, "invalid_status", "status must be active, inactive, or archived")
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())

	item, err := a.inventory.CreateItem(r.Context(), tenantID, userID, input)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "inventory_item_create_failed", "failed to create inventory item")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"tenant_id": tenantID,
		"item":      item,
	})
}

func (a *app) requireModule(moduleID string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := tenancy.TenantIDFromContext(r.Context())
		if !ok {
			httpx.Error(w, http.StatusBadRequest, "tenant_required", "tenant context is required")
			return
		}

		enabled, err := a.modules.IsEnabledForTenant(r.Context(), tenantID, moduleID)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "module_entitlement_check_failed", "failed to check module entitlement")
			return
		}

		if !enabled {
			httpx.Error(w, http.StatusForbidden, "module_not_enabled", "module is not enabled for this tenant")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *app) requirePermission(w http.ResponseWriter, r *http.Request, permissionID string) bool {
	tenantID, ok := tenancy.TenantIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "tenant_required", "tenant context is required")
		return false
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "user_required", "user context is required")
		return false
	}

	allowed, err := a.rbac.HasPermission(r.Context(), tenantID, userID, permissionID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "permission_check_failed", "failed to check permission")
		return false
	}

	if !allowed {
		httpx.Error(w, http.StatusForbidden, "permission_denied", "permission denied")
		return false
	}

	return true
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

func userMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			httpx.Error(w, http.StatusUnauthorized, "user_required", "X-User-ID header is required")
			return
		}

		ctx := auth.WithUserID(r.Context(), userID)
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

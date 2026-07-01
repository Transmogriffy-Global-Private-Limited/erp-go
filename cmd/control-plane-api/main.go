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

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/licensing"
	platformmodules "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/modules"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	db        *pgxpool.Pool
	modules   platformmodules.Store
	tenants   tenancy.Store
	licensing licensing.Store
}

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

	app := &app{
		db:        pool,
		modules:   platformmodules.NewStore(pool),
		tenants:   tenancy.NewStore(pool),
		licensing: licensing.NewStore(pool),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", app.healthHandler)
	mux.HandleFunc("/healthz/db", app.dbHealthHandler)
	mux.Handle("/control/v1/tenants", platformAuthMiddleware(http.HandlerFunc(app.tenantsHandler)))
	mux.Handle("/control/v1/tenants/", platformAuthMiddleware(http.HandlerFunc(app.tenantSubresourceHandler)))
	mux.Handle("/control/v1/modules", platformAuthMiddleware(http.HandlerFunc(app.modulesHandler)))
	mux.Handle("/control/v1/plans", platformAuthMiddleware(http.HandlerFunc(app.plansHandler)))
	mux.Handle("/control/v1/plans/", platformAuthMiddleware(http.HandlerFunc(app.planSubresourceHandler)))

	server := &http.Server{
		Addr:              addr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("control-plane-api listening on %s", addr)

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
		"service": "control-plane-api",
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
			"service": "control-plane-api",
			"status":  "db_unavailable",
			"error":   err.Error(),
		})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"service": "control-plane-api",
		"status":  "db_ok",
	})
}

func (a *app) tenantsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.listTenants(w, r)

	case http.MethodPost:
		a.createTenant(w, r)

	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *app) listTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := a.tenants.ListTenants(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "tenants_query_failed", "failed to load tenants")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenants": tenants,
	})
}

func (a *app) createTenant(w http.ResponseWriter, r *http.Request) {
	var input tenancy.CreateTenantInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	input.Slug = strings.TrimSpace(input.Slug)
	input.LegalName = strings.TrimSpace(input.LegalName)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Status = strings.TrimSpace(input.Status)

	if input.Slug == "" {
		httpx.Error(w, http.StatusBadRequest, "slug_required", "slug is required")
		return
	}

	if input.LegalName == "" {
		httpx.Error(w, http.StatusBadRequest, "legal_name_required", "legal_name is required")
		return
	}

	if input.DisplayName == "" {
		httpx.Error(w, http.StatusBadRequest, "display_name_required", "display_name is required")
		return
	}

	if input.Status != "" && input.Status != "trial" && input.Status != "active" && input.Status != "suspended" && input.Status != "closed" {
		httpx.Error(w, http.StatusBadRequest, "invalid_status", "status must be trial, active, suspended, or closed")
		return
	}

	tenant, err := a.tenants.CreateTenant(r.Context(), input)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "tenant_create_failed", "failed to create tenant")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"tenant": tenant,
	})
}

func (a *app) tenantSubresourceHandler(w http.ResponseWriter, r *http.Request) {
	if tenantID, moduleID, action, ok := parseTenantModulePath(r.URL.Path); ok {
		switch action {
		case "list":
			if r.Method != http.MethodGet {
				httpx.MethodNotAllowed(w, http.MethodGet)
				return
			}

			a.listTenantModules(w, r, tenantID)

		case "enable":
			if r.Method != http.MethodPost {
				httpx.MethodNotAllowed(w, http.MethodPost)
				return
			}

			a.enableTenantModule(w, r, tenantID, moduleID)

		case "disable":
			if r.Method != http.MethodPost {
				httpx.MethodNotAllowed(w, http.MethodPost)
				return
			}

			a.disableTenantModule(w, r, tenantID, moduleID)
		}

		return
	}

	if tenantID, ok := parseTenantSubscriptionPath(r.URL.Path); ok {
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodPost)
			return
		}

		a.assignTenantSubscription(w, r, tenantID)
		return
	}

	http.NotFound(w, r)
}

func parseTenantModulePath(path string) (tenantID string, moduleID string, action string, ok bool) {
	const prefix = "/control/v1/tenants/"

	if !strings.HasPrefix(path, prefix) {
		return "", "", "", false
	}

	rest := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(rest, "/")

	if len(parts) == 2 && parts[0] != "" && parts[1] == "modules" {
		return parts[0], "", "list", true
	}

	if len(parts) == 4 && parts[0] != "" && parts[1] == "modules" && parts[2] != "" {
		if parts[3] == "enable" || parts[3] == "disable" {
			return parts[0], parts[2], parts[3], true
		}
	}

	return "", "", "", false
}

func parseTenantSubscriptionPath(path string) (tenantID string, ok bool) {
	const prefix = "/control/v1/tenants/"

	if !strings.HasPrefix(path, prefix) {
		return "", false
	}

	rest := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(rest, "/")

	if len(parts) == 2 && parts[0] != "" && parts[1] == "subscription" {
		return parts[0], true
	}

	return "", false
}

func (a *app) listTenantModules(w http.ResponseWriter, r *http.Request, tenantID string) {
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

func (a *app) enableTenantModule(w http.ResponseWriter, r *http.Request, tenantID string, moduleID string) {
	module, err := a.modules.EnableForTenant(r.Context(), tenantID, moduleID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "tenant_module_enable_failed", "failed to enable tenant module")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"module":    module,
		"enabled":   true,
	})
}

func (a *app) disableTenantModule(w http.ResponseWriter, r *http.Request, tenantID string, moduleID string) {
	module, err := a.modules.DisableForTenant(r.Context(), tenantID, moduleID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "tenant_module_disable_failed", "failed to disable tenant module")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"module":    module,
		"enabled":   false,
	})
}

func (a *app) assignTenantSubscription(w http.ResponseWriter, r *http.Request, tenantID string) {
	var input licensing.AssignSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	input.PlanID = strings.TrimSpace(input.PlanID)
	input.Status = strings.TrimSpace(input.Status)

	if input.PlanID == "" {
		httpx.Error(w, http.StatusBadRequest, "plan_id_required", "plan_id is required")
		return
	}

	if input.Status != "" && input.Status != "trialing" && input.Status != "active" && input.Status != "past_due" && input.Status != "suspended" && input.Status != "cancelled" {
		httpx.Error(w, http.StatusBadRequest, "invalid_status", "status must be trialing, active, past_due, suspended, or cancelled")
		return
	}

	subscription, modules, err := a.licensing.AssignTenantSubscription(r.Context(), tenantID, input)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "tenant_subscription_assign_failed", "failed to assign tenant subscription")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenant_id":       tenantID,
		"subscription":    subscription,
		"enabled_modules": modules,
	})
}

func (a *app) modulesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}

	modules, err := a.modules.ListAll(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "modules_query_failed", "failed to load modules")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"modules": modules,
	})
}

func (a *app) plansHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.listPlans(w, r)

	case http.MethodPost:
		a.createPlan(w, r)

	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *app) listPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := a.licensing.ListPlans(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "plans_query_failed", "failed to load plans")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"plans": plans,
	})
}

func (a *app) createPlan(w http.ResponseWriter, r *http.Request) {
	var input licensing.CreatePlanInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	input.ID = strings.TrimSpace(input.ID)
	input.Name = strings.TrimSpace(input.Name)
	input.Status = strings.TrimSpace(input.Status)

	if input.ID == "" {
		httpx.Error(w, http.StatusBadRequest, "plan_id_required", "id is required")
		return
	}

	if input.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "plan_name_required", "name is required")
		return
	}

	if input.Status != "" && input.Status != "draft" && input.Status != "active" && input.Status != "archived" {
		httpx.Error(w, http.StatusBadRequest, "invalid_status", "status must be draft, active, or archived")
		return
	}

	plan, err := a.licensing.CreatePlan(r.Context(), input)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "plan_create_failed", "failed to create plan")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"plan": plan,
	})
}

func (a *app) planSubresourceHandler(w http.ResponseWriter, r *http.Request) {
	planID, moduleID, action, ok := parsePlanModulePath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch action {
	case "list":
		if r.Method != http.MethodGet {
			httpx.MethodNotAllowed(w, http.MethodGet)
			return
		}

		a.listPlanModules(w, r, planID)

	case "enable":
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodPost)
			return
		}

		a.enablePlanModule(w, r, planID, moduleID)

	default:
		http.NotFound(w, r)
	}
}

func parsePlanModulePath(path string) (planID string, moduleID string, action string, ok bool) {
	const prefix = "/control/v1/plans/"

	if !strings.HasPrefix(path, prefix) {
		return "", "", "", false
	}

	rest := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(rest, "/")

	if len(parts) == 2 && parts[0] != "" && parts[1] == "modules" {
		return parts[0], "", "list", true
	}

	if len(parts) == 4 && parts[0] != "" && parts[1] == "modules" && parts[2] != "" && parts[3] == "enable" {
		return parts[0], parts[2], "enable", true
	}

	return "", "", "", false
}

func (a *app) listPlanModules(w http.ResponseWriter, r *http.Request, planID string) {
	modules, err := a.licensing.ListPlanModules(r.Context(), planID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "plan_modules_query_failed", "failed to load plan modules")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"plan_id": planID,
		"modules": modules,
	})
}

func (a *app) enablePlanModule(w http.ResponseWriter, r *http.Request, planID string, moduleID string) {
	module, err := a.licensing.EnablePlanModule(r.Context(), planID, moduleID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "plan_module_enable_failed", "failed to enable plan module")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"plan_id": planID,
		"module":  module,
		"enabled": true,
	})
}

func platformAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		platformUserID := strings.TrimSpace(r.Header.Get("X-Platform-User-ID"))
		platformRole := strings.TrimSpace(r.Header.Get("X-Platform-Role"))

		if platformUserID == "" {
			httpx.Error(w, http.StatusUnauthorized, "platform_user_required", "X-Platform-User-ID header is required")
			return
		}

		if platformRole != "superadmin" {
			httpx.Error(w, http.StatusForbidden, "platform_superadmin_required", "X-Platform-Role must be superadmin")
			return
		}

		next.ServeHTTP(w, r)
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

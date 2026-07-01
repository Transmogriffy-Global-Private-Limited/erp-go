package controlapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/licensing"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
)

func (a *App) tenantsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.listTenants(w, r)
	case http.MethodPost:
		a.createTenant(w, r)
	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *App) listTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := a.tenants.ListTenants(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "tenants_query_failed", "failed to load tenants")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenants": tenants,
	})
}

func (a *App) createTenant(w http.ResponseWriter, r *http.Request) {
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

	platformActorID, _ := auth.PlatformUserIDFromContext(r.Context())

	tenant, err := a.tenants.CreateTenant(r.Context(), platformActorID, input)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "tenant_create_failed", "failed to create tenant")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"tenant": tenant,
	})
}

func (a *App) tenantSubresourceHandler(w http.ResponseWriter, r *http.Request) {
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

func (a *App) assignTenantSubscription(w http.ResponseWriter, r *http.Request, tenantID string) {
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

	platformActorID, _ := auth.PlatformUserIDFromContext(r.Context())

	subscription, modules, err := a.licensing.AssignTenantSubscription(r.Context(), platformActorID, tenantID, input)
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

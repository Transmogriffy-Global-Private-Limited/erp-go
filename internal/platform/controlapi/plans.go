package controlapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/licensing"
)

func (a *App) plansHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.listPlans(w, r)
	case http.MethodPost:
		a.createPlan(w, r)
	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *App) listPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := a.licensing.ListPlans(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "plans_query_failed", "failed to load plans")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"plans": plans,
	})
}

func (a *App) createPlan(w http.ResponseWriter, r *http.Request) {
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

	platformActorID, _ := auth.PlatformUserIDFromContext(r.Context())

	plan, err := a.licensing.CreatePlan(r.Context(), platformActorID, input)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "plan_create_failed", "failed to create plan")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"plan": plan,
	})
}

func (a *App) planSubresourceHandler(w http.ResponseWriter, r *http.Request) {
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

func (a *App) listPlanModules(w http.ResponseWriter, r *http.Request, planID string) {
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

func (a *App) enablePlanModule(w http.ResponseWriter, r *http.Request, planID string, moduleID string) {
	platformActorID, _ := auth.PlatformUserIDFromContext(r.Context())

	module, err := a.licensing.EnablePlanModule(r.Context(), platformActorID, planID, moduleID)
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

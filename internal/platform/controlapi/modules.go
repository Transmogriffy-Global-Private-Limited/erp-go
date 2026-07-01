package controlapi

import (
	"net/http"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
)

func (a *App) modulesHandler(w http.ResponseWriter, r *http.Request) {
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

func (a *App) listTenantModules(w http.ResponseWriter, r *http.Request, tenantID string) {
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

func (a *App) enableTenantModule(w http.ResponseWriter, r *http.Request, tenantID string, moduleID string) {
	platformActorID, _ := auth.PlatformUserIDFromContext(r.Context())

	module, err := a.modules.EnableForTenant(r.Context(), platformActorID, tenantID, moduleID)
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

func (a *App) disableTenantModule(w http.ResponseWriter, r *http.Request, tenantID string, moduleID string) {
	platformActorID, _ := auth.PlatformUserIDFromContext(r.Context())

	module, err := a.modules.DisableForTenant(r.Context(), platformActorID, tenantID, moduleID)
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

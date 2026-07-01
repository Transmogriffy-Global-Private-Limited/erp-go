package main

import (
	"net/http"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
)

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
		userID, ok := auth.UserIDFromContext(r.Context())
		userID = strings.TrimSpace(userID)

		if !ok || userID == "" {
			userID = strings.TrimSpace(r.Header.Get("X-User-ID"))
		}

		if userID == "" {
			httpx.Error(w, http.StatusUnauthorized, "user_required", "user context or X-User-ID header is required")
			return
		}

		ctx := auth.WithUserID(r.Context(), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

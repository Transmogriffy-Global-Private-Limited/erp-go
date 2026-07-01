package main

import (
	"net/http"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
)

func (a *app) inventoryItemsSessionHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	sessionToken := strings.TrimSpace(r.Header.Get("X-ERP-Session"))

	if tenantID == "" {
		httpx.Error(w, http.StatusBadRequest, "tenant_required", "X-Tenant-ID header is required")
		return
	}

	if sessionToken == "" {
		httpx.Error(w, http.StatusUnauthorized, "erp_session_required", "X-ERP-Session header is required")
		return
	}

	user, ok, err := a.tenantSessions.UserFromSession(r.Context(), tenantID, sessionToken)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "erp_session_check_failed", "failed to check ERP session")
		return
	}

	if !ok {
		httpx.Error(w, http.StatusForbidden, "erp_session_invalid", "ERP session is invalid")
		return
	}

	ctx := tenancy.WithTenantID(r.Context(), user.TenantID)
	ctx = auth.WithUserID(ctx, user.UserID)

	nextRequest := r.Clone(ctx)
	nextRequest.Header = r.Header.Clone()

	// Internal compatibility for the existing Inventory handler.
	// External callers still must provide X-ERP-Session to reach this point.
	nextRequest.Header.Set("X-Tenant-ID", user.TenantID)
	nextRequest.Header.Set("X-User-ID", user.UserID)

	// Preserve module entitlement enforcement.
	a.requireModule("inventory", http.HandlerFunc(a.inventoryItemsHandler)).ServeHTTP(w, nextRequest)
}

package controlapi

import (
	"net/http"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
)

func (a *App) platformAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		platformUserID := strings.TrimSpace(r.Header.Get("X-Platform-User-ID"))

		if platformUserID == "" {
			httpx.Error(w, http.StatusUnauthorized, "platform_user_required", "X-Platform-User-ID header is required")
			return
		}

		allowed, err := a.platformAuth.IsActiveSuperadmin(r.Context(), platformUserID)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "platform_auth_check_failed", "failed to check platform authorization")
			return
		}

		if !allowed {
			httpx.Error(w, http.StatusForbidden, "platform_superadmin_required", "platform user must be active superadmin")
			return
		}

		ctx := auth.WithPlatformUserID(r.Context(), platformUserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

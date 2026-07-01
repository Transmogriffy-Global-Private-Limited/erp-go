package controlapi

import (
	"net/http"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
)

func (a *App) platformAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionToken := strings.TrimSpace(r.Header.Get("X-Platform-Session"))
		if sessionToken != "" {
			platformUserID, allowed, err := a.platformAuth.SuperadminFromSession(r.Context(), sessionToken)
			if err != nil {
				httpx.Error(w, http.StatusInternalServerError, "platform_session_check_failed", "failed to check platform session")
				return
			}

			if !allowed {
				httpx.Error(w, http.StatusForbidden, "platform_superadmin_required", "platform session must belong to an active superadmin")
				return
			}

			ctx := auth.WithPlatformUserID(r.Context(), platformUserID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Temporary fallback until all local scripts move to X-Platform-Session.
		platformUserID := strings.TrimSpace(r.Header.Get("X-Platform-User-ID"))

		if platformUserID == "" {
			httpx.Error(w, http.StatusUnauthorized, "platform_user_required", "X-Platform-Session or X-Platform-User-ID header is required")
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

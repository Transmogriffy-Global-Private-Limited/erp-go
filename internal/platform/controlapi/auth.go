package controlapi

import (
	"net/http"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
)

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

		ctx := auth.WithPlatformUserID(r.Context(), platformUserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

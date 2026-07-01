package controlapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
)

type platformLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *App) platformLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w, http.MethodPost)
		return
	}

	var input platformLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	input.Email = strings.TrimSpace(input.Email)

	if input.Email == "" {
		httpx.Error(w, http.StatusBadRequest, "email_required", "email is required")
		return
	}

	if input.Password == "" {
		httpx.Error(w, http.StatusBadRequest, "password_required", "password is required")
		return
	}

	session, err := a.platformAuth.LoginSuperadmin(r.Context(), input.Email, input.Password, 24*time.Hour)
	if err != nil {
		if errors.Is(err, auth.ErrPlatformLoginLocked) {
			httpx.Error(w, http.StatusLocked, "platform_login_locked", "platform login is temporarily locked")
			return
		}

		if errors.Is(err, auth.ErrInvalidPlatformLogin) {
			httpx.Error(w, http.StatusUnauthorized, "invalid_platform_login", "invalid platform login")
			return
		}

		httpx.Error(w, http.StatusInternalServerError, "platform_login_failed", "platform login failed")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"session_token":    session.Token,
		"platform_user_id": session.PlatformUserID,
		"expires_at":       session.ExpiresAt,
	})
}

func (a *App) platformLogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w, http.MethodPost)
		return
	}

	sessionToken := strings.TrimSpace(r.Header.Get("X-Platform-Session"))
	if sessionToken == "" {
		httpx.Error(w, http.StatusUnauthorized, "platform_session_required", "X-Platform-Session header is required")
		return
	}

	revoked, err := a.platformAuth.RevokeSession(r.Context(), sessionToken)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "platform_logout_failed", "platform logout failed")
		return
	}

	if !revoked {
		httpx.Error(w, http.StatusForbidden, "platform_session_invalid", "platform session is invalid or already revoked")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"logged_out": true,
	})
}

package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
)

type tenantLoginRequest struct {
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *app) tenantLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w, http.MethodPost)
		return
	}

	var input tenantLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	input.TenantID = strings.TrimSpace(input.TenantID)
	input.Email = strings.TrimSpace(input.Email)

	if input.TenantID == "" {
		httpx.Error(w, http.StatusBadRequest, "tenant_id_required", "tenant_id is required")
		return
	}

	if input.Email == "" {
		httpx.Error(w, http.StatusBadRequest, "email_required", "email is required")
		return
	}

	if input.Password == "" {
		httpx.Error(w, http.StatusBadRequest, "password_required", "password is required")
		return
	}

	session, err := a.tenantSessions.Login(r.Context(), input.TenantID, input.Email, input.Password, 24*time.Hour)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidTenantLogin) {
			httpx.Error(w, http.StatusUnauthorized, "invalid_tenant_login", "invalid tenant login")
			return
		}

		httpx.Error(w, http.StatusInternalServerError, "tenant_login_failed", "tenant login failed")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"session_token": session.Token,
		"tenant_id":     session.TenantID,
		"user_id":       session.UserID,
		"email":         session.Email,
		"expires_at":    session.ExpiresAt,
	})
}

func (a *app) tenantMeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}

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

	httpx.JSON(w, http.StatusOK, map[string]any{
		"user": user,
	})
}

func (a *app) tenantLogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w, http.MethodPost)
		return
	}

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

	revoked, err := a.tenantSessions.RevokeSession(r.Context(), tenantID, sessionToken)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "tenant_logout_failed", "tenant logout failed")
		return
	}

	if !revoked {
		httpx.Error(w, http.StatusForbidden, "erp_session_invalid", "ERP session is invalid or already revoked")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"logged_out": true,
	})
}

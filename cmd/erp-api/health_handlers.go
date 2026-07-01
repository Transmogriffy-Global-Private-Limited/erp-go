package main

import (
	"net/http"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
)

func (a *app) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"service": "erp-api",
		"status":  "ok",
	})
}

func (a *app) dbHealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}

	if err := db.Ping(r.Context(), a.db); err != nil {
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]any{
			"service": "erp-api",
			"status":  "db_unavailable",
			"error":   err.Error(),
		})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"service": "erp-api",
		"status":  "db_ok",
	})
}

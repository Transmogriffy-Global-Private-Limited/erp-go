package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/inventory"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
)

func (a *app) inventoryLocationsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.requirePermission(w, r, "inventory.location.read") {
			return
		}
		a.listInventoryLocations(w, r)

	case http.MethodPost:
		if !a.requirePermission(w, r, "inventory.location.write") {
			return
		}
		a.createInventoryLocation(w, r)

	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *app) listInventoryLocations(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())

	locations, err := a.inventory.ListLocations(r.Context(), tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "inventory_locations_query_failed", "failed to load inventory locations")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"locations": locations,
	})
}

func (a *app) createInventoryLocation(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())

	var input inventory.CreateLocationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Status = strings.TrimSpace(input.Status)

	if input.Code == "" {
		httpx.Error(w, http.StatusBadRequest, "code_required", "code is required")
		return
	}

	if input.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "name_required", "name is required")
		return
	}

	if input.Status != "" && input.Status != "active" && input.Status != "inactive" && input.Status != "archived" {
		httpx.Error(w, http.StatusBadRequest, "invalid_status", "status must be active, inactive, or archived")
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())

	location, err := a.inventory.CreateLocation(r.Context(), tenantID, userID, input)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "inventory_location_create_failed", "failed to create inventory location")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"tenant_id": tenantID,
		"location":  location,
	})
}

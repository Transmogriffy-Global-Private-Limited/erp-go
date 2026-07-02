package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/inventory"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
)

func (a *app) enabledModulesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}

	tenantID, _ := tenancy.TenantIDFromContext(r.Context())

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

func (a *app) inventoryItemsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.requirePermission(w, r, "inventory.item.read") {
			return
		}
		a.listInventoryItems(w, r)

	case http.MethodPost:
		if !a.requirePermission(w, r, "inventory.item.write") {
			return
		}
		a.createInventoryItem(w, r)

	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *app) listInventoryItems(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())

	items, err := a.inventory.ListItems(r.Context(), tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "inventory_items_query_failed", "failed to load inventory items")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"items":     items,
	})
}

func (a *app) createInventoryItem(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())

	var input inventory.CreateItemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	input.SKU = strings.TrimSpace(input.SKU)
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Status = strings.TrimSpace(input.Status)
	input.BaseUnitID = strings.TrimSpace(input.BaseUnitID)

	if input.SKU == "" {
		httpx.Error(w, http.StatusBadRequest, "sku_required", "sku is required")
		return
	}

	if input.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "name_required", "name is required")
		return
	}

	if input.BaseUnitID == "" {
		httpx.Error(w, http.StatusBadRequest, "base_unit_required", "base_unit_id is required")
		return
	}

	if !looksLikeUUID(input.BaseUnitID) {
		httpx.Error(w, http.StatusBadRequest, "invalid_base_unit_id", "base_unit_id must be a UUID")
		return
	}

	if input.Status != "" && input.Status != "active" && input.Status != "inactive" && input.Status != "archived" {
		httpx.Error(w, http.StatusBadRequest, "invalid_status", "status must be active, inactive, or archived")
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())

	item, err := a.inventory.CreateItem(r.Context(), tenantID, userID, input)
	if err != nil {
		if errors.Is(err, inventory.ErrBaseUnitNotFound) {
			httpx.Error(w, http.StatusBadRequest, "base_unit_invalid", "base_unit_id must reference an active inventory unit")
			return
		}

		httpx.Error(w, http.StatusInternalServerError, "inventory_item_create_failed", "failed to create inventory item")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"tenant_id": tenantID,
		"item":      item,
	})
}

func looksLikeUUID(value string) bool {
	if len(value) != 36 {
		return false
	}

	for i, r := range value {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
	}

	return true
}

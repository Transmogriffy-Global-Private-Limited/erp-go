package main

import (
	"encoding/json"
	"math/big"
	"net/http"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/inventory"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
)

func (a *app) inventoryStockMovementsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.requirePermission(w, r, "inventory.stock_movement.read") {
			return
		}
		a.listInventoryStockMovements(w, r)

	case http.MethodPost:
		if !a.requirePermission(w, r, "inventory.stock_movement.write") {
			return
		}
		a.createInventoryStockMovement(w, r)

	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *app) listInventoryStockMovements(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())

	movements, err := a.inventory.ListStockMovements(r.Context(), tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "inventory_stock_movements_query_failed", "failed to load inventory stock movements")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"movements": movements,
	})
}

func (a *app) createInventoryStockMovement(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())

	var input inventory.CreateStockMovementInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	input.MovementType = strings.TrimSpace(input.MovementType)
	input.Reference = strings.TrimSpace(input.Reference)
	input.Notes = strings.TrimSpace(input.Notes)

	if input.MovementType == "" {
		httpx.Error(w, http.StatusBadRequest, "movement_type_required", "movement_type is required")
		return
	}

	if input.MovementType != "adjustment" {
		httpx.Error(w, http.StatusBadRequest, "invalid_movement_type", "direct stock movements must use adjustment; receipts, issues, and transfers use their owning document APIs")
		return
	}

	if len(input.Lines) == 0 {
		httpx.Error(w, http.StatusBadRequest, "lines_required", "at least one stock movement line is required")
		return
	}

	for i := range input.Lines {
		input.Lines[i].ItemID = strings.TrimSpace(input.Lines[i].ItemID)
		input.Lines[i].LocationID = strings.TrimSpace(input.Lines[i].LocationID)
		input.Lines[i].QuantityDelta = strings.TrimSpace(input.Lines[i].QuantityDelta)

		if input.Lines[i].ItemID == "" {
			httpx.Error(w, http.StatusBadRequest, "item_required", "line item_id is required")
			return
		}

		if !looksLikeUUID(input.Lines[i].ItemID) {
			httpx.Error(w, http.StatusBadRequest, "invalid_item_id", "line item_id must be a UUID")
			return
		}

		if input.Lines[i].LocationID == "" {
			httpx.Error(w, http.StatusBadRequest, "location_required", "line location_id is required")
			return
		}

		if !looksLikeUUID(input.Lines[i].LocationID) {
			httpx.Error(w, http.StatusBadRequest, "invalid_location_id", "line location_id must be a UUID")
			return
		}

		if !validNonZeroDecimal(input.Lines[i].QuantityDelta) {
			httpx.Error(w, http.StatusBadRequest, "invalid_quantity_delta", "line quantity_delta must be a non-zero decimal")
			return
		}
	}

	userID, _ := auth.UserIDFromContext(r.Context())

	movement, err := a.inventory.CreateStockMovement(r.Context(), tenantID, userID, input)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "inventory_stock_movement_create_failed", "failed to create inventory stock movement")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"tenant_id": tenantID,
		"movement":  movement,
	})
}

func (a *app) inventoryStockBalancesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}

	if !a.requirePermission(w, r, "inventory.stock_balance.read") {
		return
	}

	tenantID, _ := tenancy.TenantIDFromContext(r.Context())

	balances, err := a.inventory.ListStockBalances(r.Context(), tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "inventory_stock_balances_query_failed", "failed to load inventory stock balances")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"balances":  balances,
	})
}

func validNonZeroDecimal(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}

	rat := new(big.Rat)
	if _, ok := rat.SetString(value); !ok {
		return false
	}

	return rat.Sign() != 0
}

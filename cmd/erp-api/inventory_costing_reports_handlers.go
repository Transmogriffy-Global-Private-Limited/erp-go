package main

import (
	"encoding/json"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/inventory"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
	"net/http"
	"strings"
)

func (a *app) inventoryCostLayersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.requirePermission(w, r, "inventory.cost.read") {
			return
		}
		tid, _ := tenancy.TenantIDFromContext(r.Context())
		v, e := a.inventory.ListCostLayers(r.Context(), tid)
		if e != nil {
			httpx.Error(w, 500, "inventory_cost_layers_query_failed", "failed to load cost layers")
			return
		}
		httpx.JSON(w, 200, map[string]any{"tenant_id": tid, "cost_layers": v})
	case http.MethodPost:
		if !a.requirePermission(w, r, "inventory.cost.write") {
			return
		}
		var in inventory.CreateCostLayerInput
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			httpx.Error(w, 400, "invalid_json", "request body must be valid JSON")
			return
		}
		in.ItemID = strings.TrimSpace(in.ItemID)
		in.UnitCost = strings.TrimSpace(in.UnitCost)
		in.CurrencyCode = strings.ToUpper(strings.TrimSpace(in.CurrencyCode))
		if !looksLikeUUID(in.ItemID) {
			httpx.Error(w, 400, "invalid_item_id", "item_id must be a UUID")
			return
		}
		if !validNonNegativeDecimal(in.UnitCost) {
			httpx.Error(w, 400, "invalid_unit_cost", "unit_cost must be non-negative")
			return
		}
		if !validCurrencyCode(in.CurrencyCode) {
			httpx.Error(w, 400, "invalid_currency_code", "currency_code must contain three uppercase letters")
			return
		}
		tid, _ := tenancy.TenantIDFromContext(r.Context())
		uid, _ := auth.UserIDFromContext(r.Context())
		v, e := a.inventory.CreateCostLayer(r.Context(), tid, uid, in)
		if e != nil {
			httpx.Error(w, 400, "cost_item_not_found", "item must be active and tenant-owned")
			return
		}
		httpx.JSON(w, 201, map[string]any{"tenant_id": tid, "cost_layer": v})
	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}
func (a *app) inventoryValuationReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}
	if !a.requirePermission(w, r, "inventory.cost.read") {
		return
	}
	tid, _ := tenancy.TenantIDFromContext(r.Context())
	v, e := a.inventory.ListValuation(r.Context(), tid)
	if e != nil {
		httpx.Error(w, 500, "inventory_valuation_query_failed", "failed to load valuation")
		return
	}
	httpx.JSON(w, 200, map[string]any{"tenant_id": tid, "valuation": v})
}
func (a *app) inventorySummaryReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}
	if !a.requirePermission(w, r, "inventory.report.read") {
		return
	}
	tid, _ := tenancy.TenantIDFromContext(r.Context())
	v, e := a.inventory.ListInventorySummary(r.Context(), tid)
	if e != nil {
		httpx.Error(w, 500, "inventory_summary_query_failed", "failed to load inventory summary")
		return
	}
	httpx.JSON(w, 200, map[string]any{"tenant_id": tid, "summary": v})
}
func (a *app) inventoryLedgerReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}
	if !a.requirePermission(w, r, "inventory.report.read") {
		return
	}
	tid, _ := tenancy.TenantIDFromContext(r.Context())
	v, e := a.inventory.ListStockMovements(r.Context(), tid)
	if e != nil {
		httpx.Error(w, 500, "inventory_ledger_query_failed", "failed to load stock ledger")
		return
	}
	httpx.JSON(w, 200, map[string]any{"tenant_id": tid, "stock_ledger": v})
}

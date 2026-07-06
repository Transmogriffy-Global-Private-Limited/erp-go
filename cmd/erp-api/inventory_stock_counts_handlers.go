package main

import (
	"encoding/json"
	"errors"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/inventory"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
	"net/http"
	"strings"
)

func (a *app) inventoryStockCountsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.requirePermission(w, r, "inventory.stock_count.read") {
			return
		}
		tid, _ := tenancy.TenantIDFromContext(r.Context())
		v, e := a.inventory.ListStockCounts(r.Context(), tid)
		if e != nil {
			httpx.Error(w, 500, "inventory_stock_counts_query_failed", "failed to load stock counts")
			return
		}
		httpx.JSON(w, 200, map[string]any{"tenant_id": tid, "stock_counts": v})
	case http.MethodPost:
		if !a.requirePermission(w, r, "inventory.stock_count.write") {
			return
		}
		var in inventory.CreateStockCountInput
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			httpx.Error(w, 400, "invalid_json", "request body must be valid JSON")
			return
		}
		in.LocationID = strings.TrimSpace(in.LocationID)
		if !looksLikeUUID(in.LocationID) {
			httpx.Error(w, 400, "invalid_location_id", "location_id must be a UUID")
			return
		}
		if len(in.Lines) == 0 {
			httpx.Error(w, 400, "lines_required", "at least one count line is required")
			return
		}
		seen := map[string]struct{}{}
		for i := range in.Lines {
			l := &in.Lines[i]
			l.ItemID = strings.TrimSpace(l.ItemID)
			l.CountedQuantity = strings.TrimSpace(l.CountedQuantity)
			if !looksLikeUUID(l.ItemID) {
				httpx.Error(w, 400, "invalid_item_id", "item_id must be a UUID")
				return
			}
			if _, ok := seen[l.ItemID]; ok {
				httpx.Error(w, 400, "duplicate_item_id", "each item may appear once per count")
				return
			}
			seen[l.ItemID] = struct{}{}
			if !validNonNegativeDecimal(l.CountedQuantity) {
				httpx.Error(w, 400, "invalid_counted_quantity", "counted_quantity must be a non-negative decimal")
				return
			}
		}
		tid, _ := tenancy.TenantIDFromContext(r.Context())
		uid, _ := auth.UserIDFromContext(r.Context())
		v, e := a.inventory.CreateStockCount(r.Context(), tid, uid, in)
		switch {
		case errors.Is(e, inventory.ErrStockCountReferenceNotFound):
			httpx.Error(w, 400, "stock_count_reference_not_found", "location and items must be active and tenant-owned")
		case errors.Is(e, inventory.ErrStockCountDuplicateItem):
			httpx.Error(w, 400, "duplicate_item_id", "each item may appear once per count")
		case e != nil:
			httpx.Error(w, 500, "inventory_stock_count_create_failed", "failed to post stock count")
		default:
			httpx.JSON(w, 201, map[string]any{"tenant_id": tid, "stock_count": v})
		}
	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

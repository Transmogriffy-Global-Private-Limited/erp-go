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

func (a *app) inventoryTransfersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.requirePermission(w, r, "inventory.transfer.read") {
			return
		}
		tenantID, _ := tenancy.TenantIDFromContext(r.Context())
		transfers, err := a.inventory.ListTransfers(r.Context(), tenantID)
		if err != nil {
			httpx.Error(w, 500, "inventory_transfers_query_failed", "failed to load inventory transfers")
			return
		}
		httpx.JSON(w, 200, map[string]any{"tenant_id": tenantID, "transfers": transfers})
	case http.MethodPost:
		if !a.requirePermission(w, r, "inventory.transfer.write") {
			return
		}
		var input inventory.CreateTransferInput
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			httpx.Error(w, 400, "invalid_json", "request body must be valid JSON")
			return
		}
		if len(input.Lines) == 0 {
			httpx.Error(w, 400, "lines_required", "at least one transfer line is required")
			return
		}
		seen := map[string]struct{}{}
		for i := range input.Lines {
			line := &input.Lines[i]
			line.ItemID = strings.TrimSpace(line.ItemID)
			line.SourceLocationID = strings.TrimSpace(line.SourceLocationID)
			line.DestinationLocationID = strings.TrimSpace(line.DestinationLocationID)
			line.Quantity = strings.TrimSpace(line.Quantity)
			if !looksLikeUUID(line.ItemID) {
				httpx.Error(w, 400, "invalid_item_id", "item_id must be a UUID")
				return
			}
			if !looksLikeUUID(line.SourceLocationID) || !looksLikeUUID(line.DestinationLocationID) {
				httpx.Error(w, 400, "invalid_location_id", "transfer location ids must be UUIDs")
				return
			}
			if line.SourceLocationID == line.DestinationLocationID {
				httpx.Error(w, 400, "same_location", "source and destination locations must differ")
				return
			}
			if _, ok := seen[line.ItemID]; ok {
				httpx.Error(w, 400, "duplicate_item_id", "each item may appear once per transfer")
				return
			}
			seen[line.ItemID] = struct{}{}
			if !validPositiveDecimal(line.Quantity) {
				httpx.Error(w, 400, "invalid_quantity", "quantity must be a positive decimal")
				return
			}
		}
		tenantID, _ := tenancy.TenantIDFromContext(r.Context())
		userID, _ := auth.UserIDFromContext(r.Context())
		transfer, err := a.inventory.CreateTransfer(r.Context(), tenantID, userID, input)
		switch {
		case errors.Is(err, inventory.ErrTransferReferenceNotFound):
			httpx.Error(w, 400, "transfer_reference_not_found", "items and locations must be active and tenant-owned")
		case errors.Is(err, inventory.ErrTransferSameLocation):
			httpx.Error(w, 400, "same_location", "source and destination locations must differ")
		case errors.Is(err, inventory.ErrTransferDuplicateItem):
			httpx.Error(w, 400, "duplicate_item_id", "each item may appear once per transfer")
		case errors.Is(err, inventory.ErrTransferInsufficientStock):
			httpx.Error(w, 409, "insufficient_available_stock", "insufficient available source stock")
		case err != nil:
			httpx.Error(w, 500, "inventory_transfer_create_failed", "failed to post inventory transfer")
		default:
			httpx.JSON(w, 201, map[string]any{"tenant_id": tenantID, "transfer": transfer})
		}
	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

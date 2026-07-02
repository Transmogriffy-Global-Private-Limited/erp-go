package main

import (
	"encoding/json"
	"math/big"
	"net/http"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/purchase"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
)

func (a *app) purchaseReceiptsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.requirePermission(w, r, "purchase.receipt.read") {
			return
		}
		a.listPurchaseReceipts(w, r)

	case http.MethodPost:
		if !a.requirePermission(w, r, "purchase.receipt.write") {
			return
		}
		a.createPurchaseReceipt(w, r)

	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *app) listPurchaseReceipts(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())

	receipts, err := a.purchase.ListReceipts(r.Context(), tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "purchase_receipts_query_failed", "failed to load purchase receipts")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"receipts":  receipts,
	})
}

func (a *app) createPurchaseReceipt(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())

	var input purchase.CreateReceiptInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	input.SupplierName = strings.TrimSpace(input.SupplierName)
	input.Reference = strings.TrimSpace(input.Reference)
	input.Notes = strings.TrimSpace(input.Notes)

	if input.SupplierName == "" {
		httpx.Error(w, http.StatusBadRequest, "supplier_name_required", "supplier_name is required")
		return
	}

	if len(input.Lines) == 0 {
		httpx.Error(w, http.StatusBadRequest, "lines_required", "at least one receipt line is required")
		return
	}

	for i := range input.Lines {
		input.Lines[i].ItemID = strings.TrimSpace(input.Lines[i].ItemID)
		input.Lines[i].LocationID = strings.TrimSpace(input.Lines[i].LocationID)
		input.Lines[i].QuantityReceived = strings.TrimSpace(input.Lines[i].QuantityReceived)

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

		if !validPositiveDecimal(input.Lines[i].QuantityReceived) {
			httpx.Error(w, http.StatusBadRequest, "invalid_quantity_received", "line quantity_received must be a positive decimal")
			return
		}
	}

	userID, _ := auth.UserIDFromContext(r.Context())

	receipt, err := a.purchase.CreateReceipt(r.Context(), tenantID, userID, input)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "purchase_receipt_create_failed", "failed to create purchase receipt")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"tenant_id": tenantID,
		"receipt":   receipt,
	})
}

func validPositiveDecimal(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}

	rat := new(big.Rat)
	if _, ok := rat.SetString(value); !ok {
		return false
	}

	return rat.Sign() > 0
}

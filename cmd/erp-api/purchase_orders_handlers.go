package main

import (
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/purchase"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
)

func (a *app) purchaseOrdersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.requirePermission(w, r, "purchase.order.read") {
			return
		}
		a.listPurchaseOrders(w, r)
	case http.MethodPost:
		if !a.requirePermission(w, r, "purchase.order.create") {
			return
		}
		a.createPurchaseOrder(w, r)
	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *app) purchaseOrderActionHandler(w http.ResponseWriter, r *http.Request) {
	const prefix = "/api/v1/purchase/orders/"
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || !looksLikeUUID(parts[0]) || parts[1] != "approve" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w, http.MethodPost)
		return
	}

	if !a.requirePermission(w, r, "purchase.order.approve") {
		return
	}

	tenantID, _ := tenancy.TenantIDFromContext(r.Context())
	userID, _ := auth.UserIDFromContext(r.Context())
	order, err := a.purchase.ApprovePurchaseOrder(r.Context(), tenantID, userID, parts[0])
	if errors.Is(err, purchase.ErrPurchaseOrderNotFound) {
		httpx.Error(w, http.StatusNotFound, "purchase_order_not_found", "purchase order not found")
		return
	}
	if errors.Is(err, purchase.ErrPurchaseOrderNotDraft) {
		httpx.Error(w, http.StatusConflict, "purchase_order_not_draft", "purchase order is not draft")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "purchase_order_approve_failed", "failed to approve purchase order")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"tenant_id": tenantID, "purchase_order": order})
}

func (a *app) listPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())
	orders, err := a.purchase.ListPurchaseOrders(r.Context(), tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "purchase_orders_query_failed", "failed to load purchase orders")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"tenant_id": tenantID, "purchase_orders": orders})
}

func (a *app) createPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())
	var input purchase.CreatePurchaseOrderInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	input.SupplierID = strings.TrimSpace(input.SupplierID)
	input.SupplierReference = strings.TrimSpace(input.SupplierReference)
	input.Notes = strings.TrimSpace(input.Notes)
	input.CurrencyCode = strings.ToUpper(strings.TrimSpace(input.CurrencyCode))
	if !looksLikeUUID(input.SupplierID) {
		httpx.Error(w, http.StatusBadRequest, "invalid_supplier_id", "supplier_id must be a UUID")
		return
	}
	if !validCurrencyCode(input.CurrencyCode) {
		httpx.Error(w, http.StatusBadRequest, "invalid_currency_code", "currency_code must contain three uppercase letters")
		return
	}
	if len(input.Lines) == 0 {
		httpx.Error(w, http.StatusBadRequest, "lines_required", "at least one purchase order line is required")
		return
	}
	if input.ExpectedAt != nil && !input.OrderedAt.IsZero() && input.ExpectedAt.Before(input.OrderedAt) {
		httpx.Error(w, http.StatusBadRequest, "invalid_expected_at", "expected_at cannot be before ordered_at")
		return
	}

	for i := range input.Lines {
		input.Lines[i].ItemID = strings.TrimSpace(input.Lines[i].ItemID)
		input.Lines[i].Quantity = strings.TrimSpace(input.Lines[i].Quantity)
		input.Lines[i].UnitPrice = strings.TrimSpace(input.Lines[i].UnitPrice)
		if !looksLikeUUID(input.Lines[i].ItemID) {
			httpx.Error(w, http.StatusBadRequest, "invalid_item_id", "line item_id must be a UUID")
			return
		}
		if !validPositiveDecimal(input.Lines[i].Quantity) {
			httpx.Error(w, http.StatusBadRequest, "invalid_quantity", "line quantity must be a positive decimal")
			return
		}
		if !validNonNegativeDecimal(input.Lines[i].UnitPrice) {
			httpx.Error(w, http.StatusBadRequest, "invalid_unit_price", "line unit_price must be a non-negative decimal")
			return
		}
	}

	userID, _ := auth.UserIDFromContext(r.Context())
	order, err := a.purchase.CreatePurchaseOrder(r.Context(), tenantID, userID, input)
	if errors.Is(err, purchase.ErrPurchaseOrderSupplierNotFound) {
		httpx.Error(w, http.StatusBadRequest, "supplier_not_found", "supplier must be active and belong to the tenant")
		return
	}
	if errors.Is(err, purchase.ErrPurchaseOrderItemNotFound) {
		httpx.Error(w, http.StatusBadRequest, "item_not_found", "each item must be active and belong to the tenant")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "purchase_order_create_failed", "failed to create purchase order")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{"tenant_id": tenantID, "purchase_order": order})
}

func validCurrencyCode(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, char := range value {
		if char < 'A' || char > 'Z' {
			return false
		}
	}
	return true
}

func validNonNegativeDecimal(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	rat := new(big.Rat)
	if _, ok := rat.SetString(value); !ok {
		return false
	}
	return rat.Sign() >= 0
}

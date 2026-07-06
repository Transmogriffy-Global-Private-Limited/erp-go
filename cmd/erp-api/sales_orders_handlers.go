package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/sales"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
)

func (a *app) salesOrdersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.requirePermission(w, r, "sales.order.read") {
			return
		}
		a.listSalesOrders(w, r)
	case http.MethodPost:
		if !a.requirePermission(w, r, "sales.order.create") {
			return
		}
		a.createSalesOrder(w, r)
	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *app) salesOrderActionHandler(w http.ResponseWriter, r *http.Request) {
	const prefix = "/api/v1/sales/orders/"
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || !looksLikeUUID(parts[0]) || parts[1] != "confirm" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w, http.MethodPost)
		return
	}
	if !a.requirePermission(w, r, "sales.order.approve") {
		return
	}

	tenantID, _ := tenancy.TenantIDFromContext(r.Context())
	userID, _ := auth.UserIDFromContext(r.Context())
	order, err := a.sales.ConfirmOrder(r.Context(), tenantID, userID, parts[0])
	if errors.Is(err, sales.ErrSalesOrderNotFound) {
		httpx.Error(w, http.StatusNotFound, "sales_order_not_found", "sales order not found")
		return
	}
	if errors.Is(err, sales.ErrSalesOrderNotDraft) {
		httpx.Error(w, http.StatusConflict, "sales_order_not_draft", "sales order is not draft")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "sales_order_confirm_failed", "failed to confirm sales order")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"tenant_id": tenantID, "sales_order": order})
}

func (a *app) listSalesOrders(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())
	orders, err := a.sales.ListOrders(r.Context(), tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "sales_orders_query_failed", "failed to load sales orders")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"tenant_id": tenantID, "sales_orders": orders})
}

func (a *app) createSalesOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())
	var input sales.CreateSalesOrderInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	input.CustomerID = strings.TrimSpace(input.CustomerID)
	input.CustomerReference = strings.TrimSpace(input.CustomerReference)
	input.Notes = strings.TrimSpace(input.Notes)
	input.CurrencyCode = strings.ToUpper(strings.TrimSpace(input.CurrencyCode))
	if !looksLikeUUID(input.CustomerID) {
		httpx.Error(w, http.StatusBadRequest, "invalid_customer_id", "customer_id must be a UUID")
		return
	}
	if !validCurrencyCode(input.CurrencyCode) {
		httpx.Error(w, http.StatusBadRequest, "invalid_currency_code", "currency_code must contain three uppercase letters")
		return
	}
	if len(input.Lines) == 0 {
		httpx.Error(w, http.StatusBadRequest, "lines_required", "at least one sales order line is required")
		return
	}
	if input.RequestedDeliveryAt != nil && !input.OrderedAt.IsZero() && input.RequestedDeliveryAt.Before(input.OrderedAt) {
		httpx.Error(w, http.StatusBadRequest, "invalid_requested_delivery_at", "requested_delivery_at cannot be before ordered_at")
		return
	}

	seenItemIDs := make(map[string]struct{}, len(input.Lines))
	for i := range input.Lines {
		input.Lines[i].ItemID = strings.ToLower(strings.TrimSpace(input.Lines[i].ItemID))
		input.Lines[i].Quantity = strings.TrimSpace(input.Lines[i].Quantity)
		input.Lines[i].UnitPrice = strings.TrimSpace(input.Lines[i].UnitPrice)
		if !looksLikeUUID(input.Lines[i].ItemID) {
			httpx.Error(w, http.StatusBadRequest, "invalid_item_id", "line item_id must be a UUID")
			return
		}
		if _, exists := seenItemIDs[input.Lines[i].ItemID]; exists {
			httpx.Error(w, http.StatusBadRequest, "duplicate_item_id", "each item_id may appear only once per sales order")
			return
		}
		seenItemIDs[input.Lines[i].ItemID] = struct{}{}
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
	order, err := a.sales.CreateOrder(r.Context(), tenantID, userID, input)
	if errors.Is(err, sales.ErrSalesOrderCustomerNotFound) {
		httpx.Error(w, http.StatusBadRequest, "customer_not_found", "customer must be active and belong to the tenant")
		return
	}
	if errors.Is(err, sales.ErrSalesOrderItemNotFound) {
		httpx.Error(w, http.StatusBadRequest, "item_not_found", "each item must be active and belong to the tenant")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "sales_order_create_failed", "failed to create sales order")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{"tenant_id": tenantID, "sales_order": order})
}

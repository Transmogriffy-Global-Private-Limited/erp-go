package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/purchase"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
)

func (a *app) purchaseSuppliersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.requirePermission(w, r, "purchase.supplier.read") {
			return
		}
		a.listPurchaseSuppliers(w, r)

	case http.MethodPost:
		if !a.requirePermission(w, r, "purchase.supplier.write") {
			return
		}
		a.createPurchaseSupplier(w, r)

	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *app) listPurchaseSuppliers(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())

	suppliers, err := a.purchase.ListSuppliers(r.Context(), tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "purchase_suppliers_query_failed", "failed to load purchase suppliers")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"suppliers": suppliers,
	})
}

func (a *app) createPurchaseSupplier(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())

	var input purchase.CreateSupplierInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Address = strings.TrimSpace(input.Address)
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

	supplier, err := a.purchase.CreateSupplier(r.Context(), tenantID, userID, input)
	if errors.Is(err, purchase.ErrSupplierCodeExists) {
		httpx.Error(w, http.StatusConflict, "supplier_code_exists", "supplier code already exists")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "purchase_supplier_create_failed", "failed to create purchase supplier")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"tenant_id": tenantID,
		"supplier":  supplier,
	})
}

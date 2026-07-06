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

func (a *app) salesCustomersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.requirePermission(w, r, "sales.customer.read") {
			return
		}
		a.listSalesCustomers(w, r)
	case http.MethodPost:
		if !a.requirePermission(w, r, "sales.customer.write") {
			return
		}
		a.createSalesCustomer(w, r)
	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *app) listSalesCustomers(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())
	customers, err := a.sales.ListCustomers(r.Context(), tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "sales_customers_query_failed", "failed to load sales customers")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"tenant_id": tenantID, "customers": customers})
}

func (a *app) createSalesCustomer(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())
	var input sales.CreateCustomerInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	input.Phone = strings.TrimSpace(input.Phone)
	input.BillingAddress = strings.TrimSpace(input.BillingAddress)
	input.ShippingAddress = strings.TrimSpace(input.ShippingAddress)
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
	customer, err := a.sales.CreateCustomer(r.Context(), tenantID, userID, input)
	if errors.Is(err, sales.ErrCustomerCodeExists) {
		httpx.Error(w, http.StatusConflict, "customer_code_exists", "customer code already exists")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "sales_customer_create_failed", "failed to create sales customer")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{"tenant_id": tenantID, "customer": customer})
}

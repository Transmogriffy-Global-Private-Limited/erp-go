package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/inventory"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/sales"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
)

func (a *app) salesIssuesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.requirePermission(w, r, "sales.issue.read") {
			return
		}
		a.listSalesIssues(w, r)
	case http.MethodPost:
		if !a.requirePermission(w, r, "sales.issue.write") {
			return
		}
		a.createSalesIssue(w, r)
	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

type reverseSalesIssueInput struct {
	Reason string `json:"reason"`
}

func (a *app) salesIssueActionHandler(w http.ResponseWriter, r *http.Request) {
	const prefix = "/api/v1/sales/issues/"
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || !looksLikeUUID(parts[0]) || parts[1] != "reverse" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w, http.MethodPost)
		return
	}
	if !a.requirePermission(w, r, "sales.issue.reverse") {
		return
	}

	var input reverseSalesIssueInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	input.Reason = strings.TrimSpace(input.Reason)
	if input.Reason == "" {
		httpx.Error(w, http.StatusBadRequest, "reversal_reason_required", "reason is required")
		return
	}

	tenantID, _ := tenancy.TenantIDFromContext(r.Context())
	userID, _ := auth.UserIDFromContext(r.Context())
	issue, err := a.sales.ReverseIssue(r.Context(), tenantID, userID, parts[0], input.Reason)
	switch {
	case errors.Is(err, sales.ErrIssueNotFound):
		httpx.Error(w, http.StatusNotFound, "sales_issue_not_found", "sales issue not found")
	case errors.Is(err, sales.ErrIssueAlreadyReversed):
		httpx.Error(w, http.StatusConflict, "sales_issue_already_reversed", "sales issue is already reversed")
	case errors.Is(err, sales.ErrIssueNotReversible):
		httpx.Error(w, http.StatusConflict, "sales_issue_not_reversible", "sales issue is not reversible")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "sales_issue_reverse_failed", "failed to reverse sales issue")
	default:
		httpx.JSON(w, http.StatusOK, map[string]any{"tenant_id": tenantID, "issue": issue})
	}
}

func (a *app) listSalesIssues(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())
	issues, err := a.sales.ListIssues(r.Context(), tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "sales_issues_query_failed", "failed to load sales issues")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"tenant_id": tenantID, "issues": issues})
}

func (a *app) createSalesIssue(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenancy.TenantIDFromContext(r.Context())
	var input sales.CreateIssueInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	input.SalesOrderID = strings.TrimSpace(input.SalesOrderID)
	input.Reference = strings.TrimSpace(input.Reference)
	input.Notes = strings.TrimSpace(input.Notes)
	if !looksLikeUUID(input.SalesOrderID) {
		httpx.Error(w, http.StatusBadRequest, "invalid_sales_order_id", "sales_order_id must be a UUID")
		return
	}
	if len(input.Lines) == 0 {
		httpx.Error(w, http.StatusBadRequest, "lines_required", "at least one sales issue line is required")
		return
	}

	seenItemIDs := make(map[string]struct{}, len(input.Lines))
	for i := range input.Lines {
		input.Lines[i].ItemID = strings.ToLower(strings.TrimSpace(input.Lines[i].ItemID))
		input.Lines[i].LocationID = strings.ToLower(strings.TrimSpace(input.Lines[i].LocationID))
		input.Lines[i].QuantityIssued = strings.TrimSpace(input.Lines[i].QuantityIssued)
		if !looksLikeUUID(input.Lines[i].ItemID) {
			httpx.Error(w, http.StatusBadRequest, "invalid_item_id", "line item_id must be a UUID")
			return
		}
		if !looksLikeUUID(input.Lines[i].LocationID) {
			httpx.Error(w, http.StatusBadRequest, "invalid_location_id", "line location_id must be a UUID")
			return
		}
		if _, exists := seenItemIDs[input.Lines[i].ItemID]; exists {
			httpx.Error(w, http.StatusBadRequest, "duplicate_item_id", "each item_id may appear only once per sales issue")
			return
		}
		seenItemIDs[input.Lines[i].ItemID] = struct{}{}
		if !validPositiveDecimal(input.Lines[i].QuantityIssued) {
			httpx.Error(w, http.StatusBadRequest, "invalid_quantity_issued", "line quantity_issued must be a positive decimal")
			return
		}
	}

	userID, _ := auth.UserIDFromContext(r.Context())
	issue, err := a.sales.CreateIssue(r.Context(), tenantID, userID, input)
	switch {
	case errors.Is(err, sales.ErrIssueSalesOrderNotFulfillable):
		httpx.Error(w, http.StatusBadRequest, "sales_order_not_fulfillable", "sales order must be confirmed or partially fulfilled and belong to the tenant")
	case errors.Is(err, sales.ErrIssueItemNotOnOrder):
		httpx.Error(w, http.StatusBadRequest, "issue_item_not_on_order", "each issue item must exist on the sales order")
	case errors.Is(err, sales.ErrIssueQuantityExceedsRemaining):
		httpx.Error(w, http.StatusConflict, "issue_quantity_exceeds_remaining", "issue quantity exceeds the sales order remaining quantity")
	case errors.Is(err, sales.ErrIssueDuplicateItem):
		httpx.Error(w, http.StatusBadRequest, "duplicate_item_id", "each item_id may appear only once per sales issue")
	case errors.Is(err, inventory.ErrIssueItemOrLocationNotFound):
		httpx.Error(w, http.StatusBadRequest, "issue_stock_reference_not_found", "each item and location must be active and belong to the tenant")
	case errors.Is(err, inventory.ErrIssueInsufficientStock):
		httpx.Error(w, http.StatusConflict, "insufficient_stock", "insufficient stock at the requested location")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "sales_issue_create_failed", "failed to post sales issue")
	default:
		httpx.JSON(w, http.StatusCreated, map[string]any{"tenant_id": tenantID, "issue": issue})
	}
}

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
	"time"
)

func (a *app) inventoryReservationsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.requirePermission(w, r, "inventory.reservation.read") {
			return
		}
		tid, _ := tenancy.TenantIDFromContext(r.Context())
		v, e := a.inventory.ListReservations(r.Context(), tid)
		if e != nil {
			httpx.Error(w, 500, "inventory_reservations_query_failed", "failed to load reservations")
			return
		}
		httpx.JSON(w, 200, map[string]any{"tenant_id": tid, "reservations": v})
	case http.MethodPost:
		if !a.requirePermission(w, r, "inventory.reservation.write") {
			return
		}
		var in inventory.CreateReservationInput
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			httpx.Error(w, 400, "invalid_json", "request body must be valid JSON")
			return
		}
		in.ItemID = strings.TrimSpace(in.ItemID)
		in.LocationID = strings.TrimSpace(in.LocationID)
		in.Quantity = strings.TrimSpace(in.Quantity)
		if !looksLikeUUID(in.ItemID) {
			httpx.Error(w, 400, "invalid_item_id", "item_id must be a UUID")
			return
		}
		if !looksLikeUUID(in.LocationID) {
			httpx.Error(w, 400, "invalid_location_id", "location_id must be a UUID")
			return
		}
		if !validPositiveDecimal(in.Quantity) {
			httpx.Error(w, 400, "invalid_quantity", "quantity must be positive")
			return
		}
		if in.ExpiresAt != nil && !in.ExpiresAt.After(time.Now()) {
			httpx.Error(w, 400, "invalid_expires_at", "expires_at must be in the future")
			return
		}
		tid, _ := tenancy.TenantIDFromContext(r.Context())
		uid, _ := auth.UserIDFromContext(r.Context())
		v, e := a.inventory.CreateReservation(r.Context(), tid, uid, in)
		switch {
		case errors.Is(e, inventory.ErrReservationReferenceNotFound):
			httpx.Error(w, 400, "reservation_reference_not_found", "item and location must be active and tenant-owned")
		case errors.Is(e, inventory.ErrReservationInsufficientAvailable):
			httpx.Error(w, 409, "insufficient_available_stock", "insufficient stock available to reserve")
		case e != nil:
			httpx.Error(w, 500, "inventory_reservation_create_failed", "failed to create reservation")
		default:
			httpx.JSON(w, 201, map[string]any{"tenant_id": tid, "reservation": v})
		}
	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}
func (a *app) inventoryReservationActionHandler(w http.ResponseWriter, r *http.Request) {
	const p = "/api/v1/inventory/reservations/"
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, p), "/"), "/")
	if len(parts) != 2 || !looksLikeUUID(parts[0]) || parts[1] != "release" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w, http.MethodPost)
		return
	}
	if !a.requirePermission(w, r, "inventory.reservation.write") {
		return
	}
	tid, _ := tenancy.TenantIDFromContext(r.Context())
	uid, _ := auth.UserIDFromContext(r.Context())
	v, e := a.inventory.ReleaseReservation(r.Context(), tid, uid, parts[0])
	switch {
	case errors.Is(e, inventory.ErrReservationNotFound):
		httpx.Error(w, 404, "reservation_not_found", "reservation not found")
	case errors.Is(e, inventory.ErrReservationNotActive):
		httpx.Error(w, 409, "reservation_not_active", "reservation is not active")
	case e != nil:
		httpx.Error(w, 500, "inventory_reservation_release_failed", "failed to release reservation")
	default:
		httpx.JSON(w, 200, map[string]any{"tenant_id": tid, "reservation": v})
	}
}
func (a *app) inventoryAvailabilityHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}
	if !a.requirePermission(w, r, "inventory.reservation.read") {
		return
	}
	tid, _ := tenancy.TenantIDFromContext(r.Context())
	v, e := a.inventory.ListAvailability(r.Context(), tid)
	if e != nil {
		httpx.Error(w, 500, "inventory_availability_query_failed", "failed to load availability")
		return
	}
	httpx.JSON(w, 200, map[string]any{"tenant_id": tid, "availability": v})
}

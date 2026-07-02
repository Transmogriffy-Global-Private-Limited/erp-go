package main

import "net/http"

func (a *app) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/auth/login", a.tenantLoginHandler)
	mux.HandleFunc("/api/v1/auth/me", a.tenantMeHandler)
	mux.HandleFunc("/api/v1/auth/logout", a.tenantLogoutHandler)
	mux.HandleFunc("/healthz", a.healthHandler)
	mux.HandleFunc("/healthz/db", a.dbHealthHandler)
	mux.Handle("/api/v1/modules", tenantMiddleware(http.HandlerFunc(a.enabledModulesHandler)))
	mux.HandleFunc("/api/v1/inventory/items", a.inventoryItemsSessionHandler)
	mux.Handle("/api/v1/inventory/units", a.erpSessionMiddleware(a.requireModule("inventory", http.HandlerFunc(a.inventoryUnitsHandler))))
	mux.Handle("/api/v1/inventory/locations", a.erpSessionMiddleware(a.requireModule("inventory", http.HandlerFunc(a.inventoryLocationsHandler))))

	return mux
}

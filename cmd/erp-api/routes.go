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
	mux.Handle("/api/v1/inventory/stock-movements", a.erpSessionMiddleware(a.requireModule("inventory", http.HandlerFunc(a.inventoryStockMovementsHandler))))
	mux.Handle("/api/v1/inventory/stock-balances", a.erpSessionMiddleware(a.requireModule("inventory", http.HandlerFunc(a.inventoryStockBalancesHandler))))
	mux.Handle("/api/v1/inventory/transfers", a.erpSessionMiddleware(a.requireModule("inventory", http.HandlerFunc(a.inventoryTransfersHandler))))
	mux.Handle("/api/v1/inventory/stock-counts", a.erpSessionMiddleware(a.requireModule("inventory", http.HandlerFunc(a.inventoryStockCountsHandler))))
	mux.Handle("/api/v1/inventory/reservations", a.erpSessionMiddleware(a.requireModule("inventory", http.HandlerFunc(a.inventoryReservationsHandler))))
	mux.Handle("/api/v1/inventory/reservations/", a.erpSessionMiddleware(a.requireModule("inventory", http.HandlerFunc(a.inventoryReservationActionHandler))))
	mux.Handle("/api/v1/inventory/availability", a.erpSessionMiddleware(a.requireModule("inventory", http.HandlerFunc(a.inventoryAvailabilityHandler))))
	mux.Handle("/api/v1/inventory/cost-layers", a.erpSessionMiddleware(a.requireModule("inventory", http.HandlerFunc(a.inventoryCostLayersHandler))))
	mux.Handle("/api/v1/inventory/reports/valuation", a.erpSessionMiddleware(a.requireModule("inventory", http.HandlerFunc(a.inventoryValuationReportHandler))))
	mux.Handle("/api/v1/inventory/reports/summary", a.erpSessionMiddleware(a.requireModule("inventory", http.HandlerFunc(a.inventorySummaryReportHandler))))
	mux.Handle("/api/v1/inventory/reports/stock-ledger", a.erpSessionMiddleware(a.requireModule("inventory", http.HandlerFunc(a.inventoryLedgerReportHandler))))
	mux.Handle("/api/v1/purchase/suppliers", a.erpSessionMiddleware(a.requireModule("purchase", http.HandlerFunc(a.purchaseSuppliersHandler))))
	mux.Handle("/api/v1/purchase/orders", a.erpSessionMiddleware(a.requireModule("purchase", http.HandlerFunc(a.purchaseOrdersHandler))))
	mux.Handle("/api/v1/purchase/orders/", a.erpSessionMiddleware(a.requireModule("purchase", http.HandlerFunc(a.purchaseOrderActionHandler))))
	mux.Handle("/api/v1/purchase/receipts", a.erpSessionMiddleware(a.requireModule("purchase", http.HandlerFunc(a.purchaseReceiptsHandler))))
	mux.Handle("/api/v1/purchase/receipts/", a.erpSessionMiddleware(a.requireModule("purchase", http.HandlerFunc(a.purchaseReceiptActionHandler))))
	mux.Handle("/api/v1/sales/customers", a.erpSessionMiddleware(a.requireModule("sales", http.HandlerFunc(a.salesCustomersHandler))))
	mux.Handle("/api/v1/sales/orders", a.erpSessionMiddleware(a.requireModule("sales", http.HandlerFunc(a.salesOrdersHandler))))
	mux.Handle("/api/v1/sales/orders/", a.erpSessionMiddleware(a.requireModule("sales", http.HandlerFunc(a.salesOrderActionHandler))))
	mux.Handle("/api/v1/sales/issues", a.erpSessionMiddleware(a.requireModule("sales", http.HandlerFunc(a.salesIssuesHandler))))
	mux.Handle("/api/v1/sales/issues/", a.erpSessionMiddleware(a.requireModule("sales", http.HandlerFunc(a.salesIssueActionHandler))))

	return mux
}

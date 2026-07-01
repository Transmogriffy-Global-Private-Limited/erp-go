package sales

// Package sales will own customers, quotations, sales orders, invoices,
// and sales lifecycle state.
//
// Sales may request stock reservations from Inventory.
// Sales must not directly mutate Inventory stock tables.
// Sales emits events that Accounting can consume.

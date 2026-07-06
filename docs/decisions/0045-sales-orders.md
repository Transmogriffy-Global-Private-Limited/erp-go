# ADR 0045: Sales Orders

## Status

Accepted

## Context

Sales needs a tenant-owned commercial order that references stable Sales Customers and Inventory items before fulfillment can safely reduce stock.

Creating or confirming an order must not mutate Inventory. An order records commercial intent; a later fulfillment command will own the stock availability checks and negative Inventory movement.

## Decision

Add tenant-scoped Sales Orders and Sales Order Lines.

New tables:

- sales.orders
- sales.order_lines

New permissions:

- sales.order.read
- sales.order.create
- sales.order.approve

New endpoints:

- GET /api/v1/sales/orders
- POST /api/v1/sales/orders
- POST /api/v1/sales/orders/{sales_order_id}/confirm

Orders begin in draft and may be confirmed once. Customers and items must be active and belong to the authenticated tenant. Duplicate items within an order are rejected.

Creation and confirmation write tenant-scoped audit and outbox records atomically.

## Consequences

Sales Orders provide a durable commercial lifecycle without violating Inventory ownership.

Neither draft creation nor confirmation creates stock movements. A separate Sales fulfillment step must command Inventory to post negative movements and track partial fulfillment.

# ADR 0040: Purchase Orders

## Status

Accepted

## Context

The Purchase module now owns supplier master data and Purchase Receipts can post incoming stock. A purchase commitment must exist before receipts can be linked to an approved order.

Creating or approving a Purchase Order must not change Inventory stock. Stock remains owned by Inventory and changes only when a receipt posts a stock movement.

## Decision

Add tenant-scoped Purchase Orders and lines.

New tables:

- purchase.purchase_orders
- purchase.purchase_order_lines

New endpoints:

- GET /api/v1/purchase/orders
- POST /api/v1/purchase/orders
- POST /api/v1/purchase/orders/{purchase_order_id}/approve

Purchase Orders:

- reference an active tenant supplier
- contain active tenant Inventory items
- store positive quantities and non-negative unit prices
- are created as draft
- may transition once from draft to approved
- do not create Inventory stock movements

Create and approve transitions each write tenant audit and outbox records atomically.

## Consequences

The platform now records approved purchasing commitments without violating Inventory ownership.

Purchase Receipts can later link to approved Purchase Orders and enforce ordered-versus-received quantities.

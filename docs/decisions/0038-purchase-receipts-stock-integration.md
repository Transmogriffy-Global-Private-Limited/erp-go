# ADR 0038: Purchase Receipts Create Stock Movements

## Status

Accepted

## Context

Inventory now has Units of Measure, Items linked to base Units, Locations, Stock Movements, and derived Stock Balances.

The next ERP bridge is receiving purchased goods. Purchase receiving must affect inventory through the stock ledger rather than mutating item/location rows directly.

## Decision

Add posted Purchase Receipts.

New tables:

- purchase.receipts
- purchase.receipt_lines

New permissions:

- purchase.receipt.read
- purchase.receipt.write

New endpoints:

- GET /api/v1/purchase/receipts
- POST /api/v1/purchase/receipts

Creating a purchase receipt also creates:

- inventory.stock_movements with movement_type = receipt
- inventory.stock_movement_lines with positive quantity deltas

The receipt stores stock_movement_id for traceability.

Successful receipt creation writes:

- audit action: purchase.receipt.create
- outbox event: purchase.receipt.created.v1

## Consequences

Purchased goods now enter inventory through the stock ledger.

Future work can add suppliers, purchase orders, draft/post lifecycles, three-way matching, landed cost, and payable/accounting integration.

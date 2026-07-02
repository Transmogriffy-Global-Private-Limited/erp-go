# ADR 0037: Inventory Stock Ledger

## Status

Accepted

## Context

Inventory now has Units of Measure, Items linked to base Units, and Locations. The next foundation is stock itself.

Mutable quantity columns on item or location rows are unsafe for ERP accounting because they hide history and make reconciliation difficult.

## Decision

Add append-only posted stock movements.

New tables:

- inventory.stock_movements
- inventory.stock_movement_lines

New permissions:

- inventory.stock_movement.read
- inventory.stock_movement.write
- inventory.stock_balance.read

New endpoints:

- GET /api/v1/inventory/stock-movements
- POST /api/v1/inventory/stock-movements
- GET /api/v1/inventory/stock-balances

Stock movement lines store signed quantity deltas.

Stock balances are derived by summing movement lines grouped by item and location.

The first implementation supports posted movements only.

## Consequences

Inventory stock now has an auditable ledger foundation.

Future work can add movement documents, draft/post lifecycle, transfer pairs, reservations, costing, purchase receipts, sales issues, and accounting integration.

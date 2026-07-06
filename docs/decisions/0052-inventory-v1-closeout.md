# ADR 0052: Inventory v1 Closeout

## Status

Accepted

## Decision

Inventory v1 uses `inventory.locations` as its stock-node abstraction. A separate warehouse table is deferred until hierarchy or enterprise routing requirements justify it.

The module manifest now reflects implemented tables, permissions, routes, events, and active status. Consolidated summary and stock-ledger reports are exposed.

Direct stock movement creation is restricted to adjustments. Purchase Receipts, Sales Issues, reversals, and transfers use their owning document commands.

Endpoints:

- GET /api/v1/inventory/reports/summary
- GET /api/v1/inventory/reports/stock-ledger

## Consequences

Inventory v1 has inbound, outbound, reversal, transfer, count, allocation, valuation, and reporting paths with append-only stock truth.

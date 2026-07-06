# ADR 0048: Inventory Transfers

## Status

Accepted

## Decision

Inventory transfers are posted documents with one immutable `transfer` movement. Each item creates an equal negative source line and positive destination line, so the movement nets to zero.

Source and destination must be distinct active tenant locations. Source available stock is locked and checked before posting. Active reservations reduce transferable quantity.

Endpoints:

- GET /api/v1/inventory/transfers
- POST /api/v1/inventory/transfers

## Consequences

Internal movement preserves tenant-wide quantity while retaining location-level history and availability enforcement.

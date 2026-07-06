# ADR 0051: Inventory Cost History and Valuation

## Status

Accepted

## Decision

Inventory cost inputs are append-only item cost layers with unit cost, currency, source, and effective timestamp. Current valuation uses the latest effective layer for each item multiplied by derived on-hand quantity.

Endpoints:

- GET/POST /api/v1/inventory/cost-layers
- GET /api/v1/inventory/reports/valuation

## Consequences

Cost history remains auditable. This v1 uses latest-effective costing; FIFO, landed cost, and accounting journal integration remain future enhancements.

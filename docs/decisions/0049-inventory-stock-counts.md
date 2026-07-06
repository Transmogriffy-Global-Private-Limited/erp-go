# ADR 0049: Inventory Stock Counts

## Status

Accepted

## Decision

Posted stock counts capture system quantity, physical counted quantity, and variance for one location. Variances append an `adjustment` movement; existing ledger rows are never rewritten.

Endpoints:

- GET /api/v1/inventory/stock-counts
- POST /api/v1/inventory/stock-counts

## Consequences

Physical reconciliation is auditable and preserves the append-only stock truth model.

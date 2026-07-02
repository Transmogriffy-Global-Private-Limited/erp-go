# ADR 0036: Inventory Locations

## Status

Accepted

## Context

Inventory items and Units of Measure are now available, but stock cannot be modeled safely until the system knows where stock exists.

ERP inventory needs tenant-scoped locations or warehouses before stock ledgers, movement documents, reservations, availability, purchasing receipts, and sales fulfillment can be modeled.

## Decision

Add tenant-scoped Inventory Locations.

New table:

- inventory.locations

New permissions:

- inventory.location.read
- inventory.location.write

New endpoints:

- GET /api/v1/inventory/locations
- POST /api/v1/inventory/locations

Location codes are normalized to uppercase.

Successful location creation writes:

- audit action: inventory.location.create
- outbox event: inventory.location.created.v1

## Consequences

Inventory now has the core location master data needed before stock movements and stock ledger entries.

Future stock movements should reference both inventory.items and inventory.locations.

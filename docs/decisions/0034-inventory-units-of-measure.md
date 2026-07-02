# ADR 0034: Inventory Units of Measure

## Status

Accepted

## Context

Inventory items need master data before stock movements, purchasing, sales, and accounting flows can be modeled reliably.

Units of Measure are foundational ERP master data.

## Decision

Add tenant-scoped Inventory Units of Measure.

New table:

- inventory.units

New permissions:

- inventory.unit.read
- inventory.unit.write

New endpoints:

- GET /api/v1/inventory/units
- POST /api/v1/inventory/units

Unit codes are normalized to uppercase.

Successful unit creation writes:

- audit action: inventory.unit.create
- outbox event: inventory.unit.created.v1

## Consequences

Inventory now has a real master-data surface beyond items.

Future work can link inventory.items to inventory.units.

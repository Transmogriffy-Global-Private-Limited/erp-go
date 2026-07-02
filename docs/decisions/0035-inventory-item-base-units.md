# ADR 0035: Inventory Item Base Units

## Status

Accepted

## Context

Inventory items must know their base Unit of Measure before reliable stock movements, purchasing, sales, costing, and future unit conversions can be modeled.

The previous Inventory item model stored SKU/name/status only, while Units of Measure existed as separate tenant-scoped master data.

## Decision

Link tenant-scoped Inventory items to tenant-scoped active Units of Measure using:

- `inventory.items.base_unit_id`

Database behavior:

- `base_unit_id` references `inventory.units` using the composite tenant-safe key `(tenant_id, id)`.
- The column is nullable at the database layer for this migration so existing dev/test rows can survive without a backfill.
- New item creation requires `base_unit_id` at the API layer.

API behavior:

- `POST /api/v1/inventory/items` now requires `base_unit_id`.
- The referenced unit must belong to the same tenant and have `status = 'active'`.
- `GET /api/v1/inventory/items` returns both `base_unit_id` and a compact `base_unit` projection.

Successful item creation now includes base-unit information in:

- audit metadata
- outbox payload

## Consequences

Inventory item master data now has enough unit context for future stock ledgers and movements.

Existing legacy rows may temporarily have no `base_unit_id`; future work should either backfill them or enforce `NOT NULL` once migration-safe defaults are known.

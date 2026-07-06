# ADR 0050: Inventory Reservations and ATP

## Status

Accepted

## Decision

Reservations allocate item quantity at a location without changing on-hand stock. Available-to-promise is derived as on-hand minus active, unexpired reservations and is clamped at zero for reporting.

Transfers and Sales Issues enforce available quantity. Reserved stock must be explicitly released before those commands can consume it.

Endpoints:

- GET/POST /api/v1/inventory/reservations
- POST /api/v1/inventory/reservations/{reservation_id}/release
- GET /api/v1/inventory/availability

## Consequences

Reservations protect planned demand while keeping physical and allocated quantities distinct.

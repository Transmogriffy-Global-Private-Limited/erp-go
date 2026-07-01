# ADR 0029: ERP Route Wiring Regression Guard

## Status

Accepted

## Context

The ERP Inventory session migration exposed route wiring fragility.

Several failures came from Inventory accidentally reaching the old direct handler path or bypassing module entitlement checks.

## Decision

Add a static route-wiring verification script:

scripts/verify-erp-route-wiring.ps1

It checks that:

- ERP auth routes are registered.
- Inventory route points to inventoryItemsSessionHandler.
- Inventory is not directly routed to inventoryItemsHandler.
- Inventory session handler resolves X-ERP-Session.
- Inventory session handler applies requireModule("inventory").
- Inventory session handler does not internally inject X-User-ID.
- main.go no longer owns route registration.

## Consequences

verify-all.ps1 now catches route wiring regressions before runtime verification becomes confusing.

Future ERP route changes should update this script deliberately.

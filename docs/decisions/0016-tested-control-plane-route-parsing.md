# ADR 0016: Tested Control Plane Route Parsing

## Status

Accepted

## Context

Control-plane subresource routes are path-shaped and were previously parsed inside handler files.

Examples:

- /control/v1/tenants/{tenant_id}/modules
- /control/v1/tenants/{tenant_id}/modules/{module_id}/enable
- /control/v1/tenants/{tenant_id}/subscription
- /control/v1/plans/{plan_id}/modules/{module_id}/enable

Route parsing mistakes can silently send requests to 404 or the wrong handler.

## Decision

Move route parsing helpers into:

internal/platform/controlapi/routes.go

Add unit tests in:

internal/platform/controlapi/routes_test.go

## Consequences

Route parsing behavior is now isolated and tested.

Future control-plane subresource routes should add tests before handler implementation.

No external behavior change is intended.

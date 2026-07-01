# ADR 0008: ERP Module Entitlement Enforcement

## Status

Accepted

## Context

The control plane can enable and disable modules for tenants.

ERP runtime module visibility already reads from tenant entitlements.

However, hiding a module from /api/v1/modules is not enough. Module-specific ERP APIs must also be blocked when the module is disabled.

## Decision

ERP module APIs must be guarded by module entitlement checks.

The ERP API uses a module guard before serving module-specific endpoints.

Example:

- /api/v1/inventory/items requires the inventory module to be enabled for the tenant.

If the module is disabled, the endpoint returns 403 with code module_not_enabled.

## Consequences

Module visibility and module access are both controlled by tenant entitlements.

Future module routes must be wrapped with a module entitlement guard.

This is the licensing wall for ERP module APIs.

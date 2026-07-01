# ADR 0007: Control Plane Module Entitlements

## Status

Accepted

## Context

Tenants subscribe to modules through the control plane.

The ERP runtime should not hardcode module access.

Module access is represented by control.tenant_enabled_modules.

## Decision

The control plane exposes module entitlement endpoints:

GET /control/v1/tenants/{tenant_id}/modules

POST /control/v1/tenants/{tenant_id}/modules/{module_id}/enable

POST /control/v1/tenants/{tenant_id}/modules/{module_id}/disable

The ERP runtime reads enabled tenant modules from the database through the module store.

## Consequences

Superadmin/control-plane workflows can enable and disable tenant module access.

ERP APIs can later check module entitlements before allowing module-specific actions.

Disabling a module sets disabled_at instead of deleting the entitlement row.

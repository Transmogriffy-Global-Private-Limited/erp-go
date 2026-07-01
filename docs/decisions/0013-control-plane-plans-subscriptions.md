# ADR 0013: Control Plane Plans and Tenant Subscriptions

## Status

Accepted

## Context

The ERP platform needs SaaS licensing primitives.

Manual tenant module toggles are useful, but plans/subscriptions are the higher-level product model.

## Decision

Add control-plane APIs for plans and tenant subscriptions.

Plans own plan modules.

Tenant subscription assignment:

- creates a tenant subscription
- cancels previous active-like subscriptions
- enables all modules included in the assigned plan

## Consequences

Module access can now be managed through product plans.

Manual tenant module enable/disable APIs still exist for overrides and development.

ERP runtime continues to read enabled modules from control.tenant_enabled_modules.

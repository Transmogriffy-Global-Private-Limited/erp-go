# ADR 0009: ERP Permission Enforcement

## Status

Accepted

## Context

Tenant and module access are not enough for ERP security.

A tenant may have Inventory enabled, but individual tenant users still need specific permissions.

Inventory currently has permissions such as:

- inventory.item.read
- inventory.item.write

## Decision

ERP module APIs must enforce tenant-user permissions.

Temporary local development identity uses:

- X-User-ID

This is a placeholder until real authentication is introduced.

Inventory item APIs enforce:

- GET /api/v1/inventory/items requires inventory.item.read
- POST /api/v1/inventory/items requires inventory.item.write

Permission checks use:

- core.tenant_users
- core.user_roles
- core.role_permissions
- core.permissions

Permission queries run inside tenant-scoped transactions so RLS sees app.tenant_id.

## Consequences

Future ERP module APIs must define required permissions before implementation.

X-User-ID must later be replaced by real authenticated identity.

No module-specific ERP API should rely only on X-Tenant-ID.

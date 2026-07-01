# Native Local Development

This project does not use Docker or containers for local development.

## Environment files

Use .env.example as the committed template.

Use .env as the local machine-specific file.

Never commit .env.

## Database URLs

Three database URLs are currently used:

- MIGRATION_DATABASE_URL
- CONTROL_PLANE_DATABASE_URL
- ERP_DATABASE_URL

MIGRATION_DATABASE_URL is used by local migration scripts.

CONTROL_PLANE_DATABASE_URL is used by control-plane-api.

ERP_DATABASE_URL is used by erp-api and erp-worker.

## Database roles

Local development currently expects:

- erp_owner
- erp_app

erp_owner owns the local database and applies migrations.

erp_app is the application runtime role.

## Temporary local identity

ERP APIs currently use temporary headers:

- X-Tenant-ID
- X-User-ID

X-User-ID is only a placeholder until real authentication is introduced.

## Scripts

Check DB connectivity:

.\scripts\db-check.ps1

Seed the default local dev tenant:

.\scripts\seed-dev-tenant.ps1

Seed local dev RBAC users/roles:

.\scripts\seed-dev-rbac.ps1

Verify tenant isolation through the ERP API:

.\scripts\verify-tenant-isolation.ps1

Verify module entitlement enforcement through control-plane and ERP APIs:

.\scripts\verify-module-entitlement.ps1

Verify RBAC permission enforcement:

.\scripts\verify-rbac.ps1

Verify audit/outbox writes:

.\scripts\verify-audit-outbox.ps1

Verify outbox worker:

.\scripts\verify-outbox-worker.ps1

Run control plane API:

.\scripts\run-control-plane-api.ps1

Run ERP API:

.\scripts\run-erp-api.ps1

Run ERP worker once:

.\scripts\run-erp-worker.ps1 -Once -Limit 100

Run ERP worker continuously:

.\scripts\run-erp-worker.ps1

## Dev tenants

Default local dev tenant:

00000000-0000-0000-0000-000000000001

Second local dev tenant used for isolation checks:

00000000-0000-0000-0000-000000000002

## Dev users

Default allowed dev user:

11111111-1111-1111-1111-111111111111

Default no-access dev user:

22222222-2222-2222-2222-222222222222

## Mutation verification

Inventory item creation should create:

- inventory.items row
- audit.audit_log row
- core.outbox_events row

Use:

.\scripts\verify-audit-outbox.ps1

## Worker verification

The DB-backed worker should claim pending outbox rows and mark them published.

Use:

.\scripts\verify-outbox-worker.ps1

## Migration ledger

The native migration script uses:

public.schema_migrations

This prevents accidentally rerunning migrations that have already been applied.

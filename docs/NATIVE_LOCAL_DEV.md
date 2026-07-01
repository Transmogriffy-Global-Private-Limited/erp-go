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

ERP_DATABASE_URL is used by erp-api.

## Database roles

Local development currently expects:

- erp_owner
- erp_app

erp_owner owns the local database and applies migrations.

erp_app is the application runtime role.

## Scripts

Check DB connectivity:

.\scripts\db-check.ps1

Apply migration:

.\scripts\apply-migration.ps1 -Name 000002_runtime_role_grants -Direction up

Rollback migration:

.\scripts\apply-migration.ps1 -Name 000002_runtime_role_grants -Direction down

Mark an already-applied migration in the local migration ledger:

.\scripts\mark-migration-applied.ps1 -Name 000001_platform_foundation

Seed the default local dev tenant:

.\scripts\seed-dev-tenant.ps1

Verify tenant isolation through the ERP API:

.\scripts\verify-tenant-isolation.ps1

Verify module entitlement enforcement through control-plane and ERP APIs:

.\scripts\verify-module-entitlement.ps1

Run control plane API:

.\scripts\run-control-plane-api.ps1

Run ERP API:

.\scripts\run-erp-api.ps1

## Dev tenants

Default local dev tenant:

00000000-0000-0000-0000-000000000001

Second local dev tenant used for isolation checks:

00000000-0000-0000-0000-000000000002

The seed script enables all currently registered modules for each seeded tenant.

## Migration ledger

The native migration script uses:

public.schema_migrations

This prevents accidentally rerunning migrations that have already been applied.

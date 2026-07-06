# Native Local Development

This project does not use Docker or containers for local development.

## Environment files

Use .env.example as the committed template.

Use .env as the local machine-specific file.

Never commit .env.

## Temporary local identity

Control-plane APIs currently require:

- X-Platform-Session

Protected ERP business APIs currently require:

- X-Tenant-ID
- X-ERP-Session

Session tokens are obtained through the platform or tenant login endpoints.

## Run services

Run control plane API:

.\scripts\run-control-plane-api.ps1

Run ERP API:

.\scripts\run-erp-api.ps1

Run ERP worker once:

.\scripts\run-erp-worker.ps1 -Once -Limit 100

Run ERP worker continuously:

.\scripts\run-erp-worker.ps1

## Verification

Run all checks:

.\scripts\verify-all.ps1

Individual checks:

.\scripts\db-check.ps1
.\scripts\verify-control-plane-auth.ps1
.\scripts\verify-tenant-isolation.ps1
.\scripts\verify-module-entitlement.ps1
.\scripts\verify-rbac.ps1
.\scripts\verify-audit-outbox.ps1
.\scripts\verify-outbox-worker.ps1
.\scripts\verify-purchase-suppliers.ps1
.\scripts\verify-purchase-orders.ps1
.\scripts\verify-sales-customers.ps1
.\scripts\verify-sales-orders.ps1
.\scripts\verify-sales-issues.ps1
.\scripts\verify-inventory-v1.ps1

## Database URLs

Three database URLs are currently used:

- MIGRATION_DATABASE_URL
- CONTROL_PLANE_DATABASE_URL
- ERP_DATABASE_URL

MIGRATION_DATABASE_URL is used by local migration scripts.

CONTROL_PLANE_DATABASE_URL is used by control-plane-api.

ERP_DATABASE_URL is used by erp-api and erp-worker.

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

## Migration ledger

The native migration script uses:

public.schema_migrations

This prevents accidentally rerunning migrations that have already been applied.


# AI Changelog

This file records meaningful AI-assisted changes.

Every future AI agent must update this file after changing code, docs, architecture, migrations, or project structure.

## 2026-07-01

### Added AI/project-memory layer

Created the first repository memory layer before application code.

Added:

- AGENTS.md
- docs/PROJECT_STATE.md
- docs/ARCHITECTURE_BASELINE.md
- docs/AI_CHANGELOG.md
- docs/HANDOFF_TEMPLATE.md
- docs/decisions/0001-two-plane-saas.md
- docs/decisions/0002-build-own-platform.md
- docs/decisions/0003-tenant-isolation-encryption.md
- .github/copilot-instructions.md

Reason:

- Future AI/dev agents should be able to take over without losing architectural context.
- The project needs durable memory before code begins.
- The repo should preserve decisions around SaaS, superadmin, licensing, modules, and tenant isolation.

Current next step:

- Create minimal Go repo spine and bootable APIs.

## 2026-07-01

### Added minimal Go API spine

Added the first bootable Go backend structure:

- go.mod
- .editorconfig
- README.md
- cmd/control-plane-api
- cmd/erp-api
- internal/platform/httpx
- internal/platform/tenancy

Reason:

- Establish the first executable backend foundation.
- Keep control plane and ERP plane separate from the beginning.
- Provide a tenant-context placeholder before real auth is added.

Verification:

- Run gofmt.
- Run go test ./...
- Run both APIs and hit health endpoints.

Current next step:

- Add platform directory structure for auth, RBAC, licensing, modules, events, outbox, files, audit, and workflow.

## 2026-07-01

### Added platform package map and initial module manifests

Added package documentation files for platform concerns:

- auth
- rbac
- licensing
- modules
- events
- outbox
- files
- audit
- workflow

Added package documentation files for initial business modules:

- inventory
- sales
- purchase
- accounting

Added initial module manifests for:

- inventory
- sales
- purchase
- accounting

Reason:

- Make platform/module boundaries explicit before implementing business logic.
- Give future AI/dev agents stable package intent.
- Establish manifest-driven module contracts early.

Verification:

- Run gofmt.
- Run go test ./...

Current next step:

- Add base database migrations for control/core/outbox and tenant isolation foundations.

## 2026-07-01

### Added database foundation migration

Added initial SQL migrations:

- migrations/README.md
- migrations/000001_platform_foundation.up.sql
- migrations/000001_platform_foundation.down.sql

Added ADR:

- docs/decisions/0004-database-foundation-rls.md

The migration establishes:

- control plane schemas/tables
- tenant subscription/module entitlement tables
- ERP tenant user/RBAC base tables
- document metadata table
- audit log table
- outbox events table
- sample inventory.items tenant-owned table
- core.current_tenant_id()
- Row Level Security policies for tenant-owned tables

Reason:

- Tenant isolation must exist at the data layer early.
- The system needs control plane and ERP plane database foundations before real modules.

Verification:

- Run go test ./...
- SQL execution will be verified after adding local Postgres/dev migration runner.

Current next step:

- Add local development infrastructure using Docker Compose for PostgreSQL, Redis, NATS, and object storage.

## 2026-07-01

### Accepted no-container local development constraint

Added ADR:

- docs/decisions/0005-no-containers-local-dev.md

Decision:

- Do not use Docker, Docker Compose, or container-based local infrastructure unless explicitly reversed later.

Reason:

- The project should fit the current Windows 11 + PowerShell + VS Code development environment.
- Future agents must not keep suggesting container-based workflows.

Current next step:

- Add native database configuration and a PowerShell migration apply script using psql.

## 2026-07-01

### Added native environment and migration scripts

Added:

- .env.example
- scripts/Import-DotEnv.ps1
- scripts/db-check.ps1
- scripts/apply-migration.ps1
- docs/NATIVE_LOCAL_DEV.md

Updated:

- .gitignore

Reason:

- Local development must not use Docker or containers.
- The project needs a native PowerShell and psql workflow for database checks and migrations.
- .env.example is committed, while .env remains local-only.

Current next step:

- Create local PostgreSQL roles/database.
- Copy .env.example to .env.
- Run scripts/db-check.ps1.
- Apply the first migration.

## 2026-07-01

### Hardened native migration scripts

Updated:

- scripts/db-check.ps1
- scripts/apply-migration.ps1
- docs/NATIVE_LOCAL_DEV.md

Added:

- scripts/mark-migration-applied.ps1

Reason:

- Native PowerShell commands do not always stop on psql failure unless LASTEXITCODE is checked.
- The first migration was already applied manually before migration tracking existed.
- A migration ledger is needed to avoid accidentally rerunning already-applied migrations.

Current next step:

- Mark 000001_platform_foundation as applied in public.schema_migrations.
- Re-run apply-migration.ps1 and confirm it skips the already-applied migration.

## 2026-07-01

### Verified native database migration workflow

Verified locally:

- scripts/db-check.ps1 connects successfully to PostgreSQL
- 000001_platform_foundation is applied
- public.schema_migrations tracks the migration
- scripts/apply-migration.ps1 skips rerunning the already-applied migration
- commit 44a88d6 was pushed to origin/anubhab-work

Reason:

- The database foundation is no longer only SQL in the repo.
- The local native migration workflow is proven.
- The next safe implementation step is application runtime DB connectivity.

Current next step:

- Add shared Go database configuration/connection package.
- Add DB health endpoints.

## 2026-07-01

### Added runtime PostgreSQL connectivity and DB health endpoints

Added:

- internal/platform/db
- scripts/run-control-plane-api.ps1
- scripts/run-erp-api.ps1
- migrations/000002_runtime_role_grants.up.sql
- migrations/000002_runtime_role_grants.down.sql

Updated:

- cmd/control-plane-api/main.go
- cmd/erp-api/main.go
- scripts/db-check.ps1
- .env.example
- README.md
- docs/NATIVE_LOCAL_DEV.md

Reason:

- Both Go APIs now need runtime PostgreSQL connectivity.
- Health checks should include database health.
- Runtime DB role grants must be managed explicitly.
- Project docs must stay aligned with executable behavior.

Current next step:

- Apply 000002_runtime_role_grants.
- Run database checks.
- Start both APIs through scripts.
- Verify /healthz/db on both services.

## 2026-07-01

### Fixed migration script null-output handling

Updated:

- scripts/apply-migration.ps1

Reason:

- psql -At returns no output when a migration is not yet present in public.schema_migrations.
- PowerShell represented that empty output as null, causing .Trim() to fail.
- The script now safely handles empty migration ledger results.

Verified before this fix:

- db-check.ps1 successfully connected with MIGRATION_DATABASE_URL, CONTROL_PLANE_DATABASE_URL, and ERP_DATABASE_URL.
- control-plane-api /healthz/db returned db_ok.
- erp-api /healthz/db returned db_ok.

Current next step:

- Apply 000002_runtime_role_grants.
- Verify schema_migrations contains both 000001_platform_foundation and 000002_runtime_role_grants.

## 2026-07-01

### Made module endpoints database-backed

Added:

- internal/platform/modules/store.go
- scripts/seed-dev-tenant.ps1

Updated:

- cmd/control-plane-api/main.go
- cmd/erp-api/main.go
- README.md
- docs/NATIVE_LOCAL_DEV.md

Behavior:

- GET /control/v1/modules now reads from control.modules.
- GET /api/v1/modules now reads enabled modules from control.tenant_enabled_modules for the supplied tenant.
- Local dev tenant 00000000-0000-0000-0000-000000000001 can be seeded through scripts/seed-dev-tenant.ps1.

Reason:

- Remove hardcoded module lists from APIs.
- Start using the database as the source of truth for module registry and tenant entitlements.

Current next step:

- Seed local dev tenant.
- Verify both module endpoints return DB-backed data.

## 2026-07-01

### Added tenant-scoped DB transactions and Inventory items API

Added:

- internal/platform/db/tenant.go
- internal/modules/inventory/store.go
- docs/decisions/0006-tenant-scoped-db-transactions.md

Updated:

- cmd/erp-api/main.go
- README.md

Behavior:

- Tenant-owned DB operations now have a required helper for setting app.tenant_id inside a transaction.
- ERP API now supports GET /api/v1/inventory/items.
- ERP API now supports POST /api/v1/inventory/items.

Reason:

- PostgreSQL RLS policies depend on app.tenant_id.
- Tenant isolation must be enforced in runtime code, not only in SQL migrations.
- Inventory items are the first real tenant-owned ERP resource.

Current next step:

- Verify inventory item create/list through erp-api.
- Confirm tenant-scoped access works through RLS.

## 2026-07-01

### Added tenant isolation verification script

Updated:

- scripts/seed-dev-tenant.ps1
- README.md
- docs/NATIVE_LOCAL_DEV.md

Added:

- scripts/verify-tenant-isolation.ps1

Behavior:

- seed-dev-tenant.ps1 can now seed arbitrary dev tenants.
- verify-tenant-isolation.ps1 seeds tenant A and tenant B, creates an inventory item under tenant A, confirms tenant A can see it, and confirms tenant B cannot see it.

Reason:

- Tenant isolation must be verified through the actual ERP API path.
- The first tenant-owned Inventory API should prove RLS-backed separation between tenants.

Current next step:

- Run verify-tenant-isolation.ps1 while erp-api is running.
- Commit verification tooling if successful.

## 2026-07-01

### Verified tenant isolation through ERP API

Verified:

- scripts/verify-tenant-isolation.ps1 passed
- tenant A can see its own inventory item
- tenant B cannot see tenant A inventory item

Result:

- Runtime tenant isolation path is working through the actual ERP API.

This validates:

- tenant middleware
- Go tenant context
- internal/platform/db.WithTenantTx
- PostgreSQL app.tenant_id
- Row Level Security on inventory.items

Current next step:

- Make control-plane tenant APIs database-backed.

## 2026-07-01

### Made control-plane tenant APIs database-backed

Added:

- internal/platform/tenancy/store.go

Updated:

- cmd/control-plane-api/main.go
- README.md

Behavior:

- GET /control/v1/tenants reads from control.tenants.
- POST /control/v1/tenants creates a tenant in control.tenants.
- Tenant creation validates slug, legal_name, display_name, and status.

Reason:

- The control-plane tenant endpoint was still a placeholder.
- Tenant lifecycle must become database-backed before licensing/module enablement APIs.

Current next step:

- Verify GET and POST /control/v1/tenants through control-plane-api.
- Then add control-plane module enable/disable endpoint for tenants.

## 2026-07-01

### Added control-plane tenant module entitlement APIs

Added:

- docs/decisions/0007-control-plane-module-entitlements.md

Updated:

- internal/platform/modules/store.go
- cmd/control-plane-api/main.go
- README.md

Behavior:

- GET /control/v1/tenants/{tenant_id}/modules lists enabled modules for a tenant.
- POST /control/v1/tenants/{tenant_id}/modules/{module_id}/enable enables a tenant module.
- POST /control/v1/tenants/{tenant_id}/modules/{module_id}/disable disables a tenant module.

Reason:

- Module licensing/entitlements must be controlled through the control plane.
- ERP runtime module visibility should reflect database entitlements.

Current next step:

- Verify disabling a module through control-plane removes it from ERP /api/v1/modules.
- Verify enabling it restores it.

## 2026-07-01

### Verified control-plane module entitlement behavior

Verified:

- GET /control/v1/tenants/{tenant_id}/modules lists enabled modules.
- POST /control/v1/tenants/{tenant_id}/modules/inventory/disable disables Inventory.
- ERP GET /api/v1/modules reflects Inventory removal.
- POST /control/v1/tenants/{tenant_id}/modules/inventory/enable re-enables Inventory.
- ERP GET /api/v1/modules reflects Inventory restoration.

Result:

- Control-plane tenant module entitlements are connected to ERP runtime module visibility.

Current next step:

- Add ERP-side module entitlement enforcement so module-specific APIs are blocked when the module is disabled.

## 2026-07-01

### Added ERP-side module entitlement enforcement

Added:

- scripts/verify-module-entitlement.ps1
- docs/decisions/0008-erp-module-entitlement-enforcement.md

Updated:

- internal/platform/modules/store.go
- cmd/erp-api/main.go
- README.md
- docs/NATIVE_LOCAL_DEV.md

Behavior:

- ERP module APIs can now be guarded by tenant module entitlements.
- /api/v1/inventory/items requires inventory to be enabled for the tenant.
- If inventory is disabled, /api/v1/inventory/items returns 403 module_not_enabled.

Reason:

- Module visibility is not enough.
- Module-specific ERP APIs must enforce licensing/module entitlement rules.

Current next step:

- Run verify-module-entitlement.ps1 with both APIs running.
- Commit after verification succeeds.

## 2026-07-01

### Verified ERP module entitlement enforcement

Verified:

- scripts/verify-module-entitlement.ps1 passed
- Inventory API works while inventory is enabled
- Inventory API returns 403 while inventory is disabled
- Inventory API works again after re-enable

Result:

- ERP-side licensing/module entitlement wall is working.

Current next step:

- Add tenant user/RBAC foundation and protect Inventory item APIs with permissions.

## 2026-07-01

### Added ERP-side RBAC permission enforcement

Added:

- internal/platform/auth/context.go
- internal/platform/rbac/store.go
- scripts/seed-dev-rbac.ps1
- scripts/verify-rbac.ps1
- docs/decisions/0009-erp-permission-enforcement.md

Updated:

- cmd/erp-api/main.go
- scripts/verify-tenant-isolation.ps1
- scripts/verify-module-entitlement.ps1
- README.md
- docs/NATIVE_LOCAL_DEV.md

Behavior:

- Inventory item APIs now require X-User-ID.
- GET /api/v1/inventory/items requires inventory.item.read.
- POST /api/v1/inventory/items requires inventory.item.write.
- RBAC checks run through tenant-scoped DB transactions.
- Existing verification scripts now seed/use RBAC identity.

Reason:

- Tenant/module access is not sufficient for ERP security.
- Module-specific ERP APIs need user-level permission enforcement.

Current next step:

- Run verify-rbac.ps1.
- Re-run verify-tenant-isolation.ps1 and verify-module-entitlement.ps1.
- Commit after all verification passes.

## 2026-07-01

### Added audit and outbox writes for Inventory item creation

Added:

- internal/platform/audit/audit.go
- internal/platform/outbox/outbox.go
- scripts/verify-audit-outbox.ps1
- docs/decisions/0010-audit-outbox-mutations.md

Updated:

- internal/modules/inventory/store.go
- cmd/erp-api/main.go
- scripts/seed-dev-rbac.ps1
- README.md
- docs/NATIVE_LOCAL_DEV.md

Behavior:

- POST /api/v1/inventory/items now writes inventory.items, audit.audit_log, and core.outbox_events in one tenant-scoped transaction.
- Inventory item creation emits durable outbox event inventory.item.created.v1.
- Inventory item creation writes audit action inventory.item.create.
- seed-dev-rbac.ps1 now displays seeded users/roles correctly under RLS.

Reason:

- ERP mutations must be accountable and event-capable from the beginning.
- Business row, audit row, and outbox event must commit together.

Current next step:

- Run verify-audit-outbox.ps1.
- Re-run verify-rbac.ps1.
- Commit after verification succeeds.

## 2026-07-01

### Fixed audit/outbox verification under RLS

Updated:

- scripts/verify-audit-outbox.ps1

Reason:

- audit.audit_log is protected by RLS.
- The verification query must set app.tenant_id in the same database session before counting audit rows.
- The mutation path was working, but the verification query was not RLS-aware enough.

Current next step:

- Commit audit/outbox mutation support after verify-audit-outbox.ps1 and verify-rbac.ps1 pass.

## 2026-07-01

### Added DB-backed outbox worker v1

Added:

- cmd/erp-worker/main.go
- scripts/run-erp-worker.ps1
- scripts/verify-outbox-worker.ps1
- docs/decisions/0011-db-backed-outbox-worker-first.md

Updated:

- internal/platform/outbox/outbox.go
- README.md
- docs/NATIVE_LOCAL_DEV.md

Behavior:

- erp-worker claims pending/failed outbox events.
- Claimed events are marked publishing.
- Worker logs the event payload.
- Worker marks processed events published.
- verify-outbox-worker.ps1 creates an inventory item, confirms pending outbox row, runs the worker once, and confirms the row becomes published.

Reason:

- Durable event processing should work before introducing any external event bus.
- The project has a no-container local development constraint, so DB-backed worker is the right first event-processing step.

Current next step:

- Run verify-outbox-worker.ps1.
- Re-run verify-audit-outbox.ps1 and verify-rbac.ps1.
- Commit after verification succeeds.

## 2026-07-01

### Added master verification script

Added:

- scripts/verify-all.ps1

Updated:

- README.md
- docs/NATIVE_LOCAL_DEV.md

Behavior:

- verify-all.ps1 runs go tests, DB checks, API health checks, RBAC verification, tenant isolation verification, module entitlement verification, audit/outbox verification, and outbox worker verification.

Reason:

- The project now has multiple critical invariants.
- A single command should verify the current ERP spine before future changes.

Current next step:

- Run verify-all.ps1 with control-plane-api and erp-api running.
- Commit after verification succeeds.

## 2026-07-01

### Added temporary control-plane auth guard

Added:

- scripts/verify-control-plane-auth.ps1
- docs/decisions/0012-temporary-control-plane-auth-guard.md

Updated:

- cmd/control-plane-api/main.go
- scripts/verify-all.ps1
- scripts/verify-rbac.ps1
- scripts/verify-module-entitlement.ps1
- scripts/verify-audit-outbox.ps1
- scripts/verify-outbox-worker.ps1
- README.md
- docs/NATIVE_LOCAL_DEV.md

Behavior:

- Health endpoints remain public.
- /control/v1/* endpoints require X-Platform-User-ID and X-Platform-Role: superadmin.
- Verification scripts now pass temporary superadmin headers when calling control-plane APIs.

Reason:

- Control-plane APIs manage platform-level state and must not remain open.

Current next step:

- Restart control-plane-api.
- Run verify-all.ps1.
- Commit after verification succeeds.

## 2026-07-01

### Verified temporary control-plane auth guard

Verified:

- public health endpoints still work
- control-plane endpoints reject missing platform identity
- control-plane endpoints reject non-superadmin role
- control-plane endpoints allow temporary superadmin headers
- verify-all.ps1 passed

Result:

- Temporary control-plane auth guard is working.

## 2026-07-01

### Added control-plane plans and tenant subscriptions

Added:

- internal/platform/licensing/store.go
- scripts/verify-plans-subscriptions.ps1
- docs/decisions/0013-control-plane-plans-subscriptions.md

Updated:

- cmd/control-plane-api/main.go
- scripts/verify-all.ps1

Behavior:

- GET /control/v1/plans lists plans.
- POST /control/v1/plans creates plans.
- GET /control/v1/plans/{plan_id}/modules lists plan modules.
- POST /control/v1/plans/{plan_id}/modules/{module_id}/enable adds a module to a plan.
- POST /control/v1/tenants/{tenant_id}/subscription assigns a plan subscription to a tenant and enables plan modules for that tenant.

Reason:

- SaaS licensing needs plans/subscriptions, not only manual module toggles.

Current next step:

- Restart control-plane-api.
- Run verify-plans-subscriptions.ps1.
- Run verify-all.ps1.
- Commit after success.

## 2026-07-01

### Added control-plane platform audit logging

Added:

- migrations/000003_control_plane_audit.up.sql
- migrations/000003_control_plane_audit.down.sql
- internal/platform/auth/platform_context.go
- internal/platform/audit/platform_audit.go
- scripts/verify-control-plane-audit.ps1
- docs/decisions/0014-control-plane-platform-audit-log.md

Updated:

- cmd/control-plane-api/main.go
- internal/platform/tenancy/store.go
- internal/platform/modules/store.go
- internal/platform/licensing/store.go
- scripts/verify-all.ps1

Behavior:

- control.platform_audit_log records platform/superadmin actions.
- Tenant creation writes platform audit.
- Tenant module enable/disable writes platform audit.
- Plan creation writes platform audit.
- Plan module enable writes platform audit.
- Tenant subscription assignment writes platform audit.

Reason:

- Control-plane actions are platform-powerful and need accountability.
- Platform audit should be separate from tenant ERP business audit.

Current next step:

- Apply migration 000003_control_plane_audit.
- Restart control-plane-api.
- Run verify-control-plane-audit.ps1.
- Run verify-all.ps1.
- Commit after success.

## 2026-07-01

### Refactored control-plane API into internal package

Added:

- internal/platform/controlapi/server.go
- internal/platform/controlapi/auth.go
- internal/platform/controlapi/tenants.go
- internal/platform/controlapi/modules.go
- internal/platform/controlapi/plans.go
- docs/decisions/0015-split-control-plane-api-package.md

Updated:

- cmd/control-plane-api/main.go

Behavior:

- No intended behavior change.
- cmd/control-plane-api/main.go is now process boot only.
- Control-plane routes, handlers, and auth middleware live under internal/platform/controlapi.

Reason:

- The control-plane main file had become too large.
- The next control-plane features should not keep expanding a monolithic main.go.

Current next step:

- Restart control-plane-api.
- Run verify-all.ps1.
- Commit if verification passes.

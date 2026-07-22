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

## 2026-07-01

### Cleaned and tested control-plane route parsing

Added:

- internal/platform/controlapi/routes.go
- internal/platform/controlapi/routes_test.go
- docs/decisions/0016-tested-control-plane-route-parsing.md

Updated:

- internal/platform/controlapi/tenants.go
- internal/platform/controlapi/plans.go

Behavior:

- No intended external behavior change.
- Tenant module path parsing moved into route helpers.
- Tenant subscription path parsing moved into route helpers.
- Plan module path parsing moved into route helpers.
- Route parsing now has unit tests.

Reason:

- Control-plane route parsing should not be buried inside handler files.
- Future route additions need tested parsing behavior.

Current next step:

- Run go test ./...
- Restart control-plane-api.
- Run verify-all.ps1.
- Commit after success.

## 2026-07-01

### Added DB-backed platform superadmin check

Added:

- migrations/000004_platform_users.up.sql
- migrations/000004_platform_users.down.sql
- internal/platform/auth/platform_store.go
- scripts/seed-platform-superadmin.ps1
- docs/decisions/0017-db-backed-platform-superadmin-check.md

Updated:

- internal/platform/controlapi/server.go
- internal/platform/controlapi/auth.go
- scripts/verify-control-plane-auth.ps1
- scripts/verify-all.ps1

Behavior:

- X-Platform-User-ID remains the temporary platform identity carrier.
- X-Platform-Role is no longer trusted for authorization.
- Control-plane authorization checks control.platform_users for active superadmin.
- verify-control-plane-auth.ps1 deliberately sends X-Platform-Role = viewer for the DB-backed superadmin request to prove the role header is ignored.

Reason:

- Control-plane APIs should not trust caller-supplied role headers.

Current next step:

- Apply migration 000004_platform_users.
- Seed platform superadmin.
- Restart control-plane-api.
- Run verify-control-plane-auth.ps1.
- Run verify-all.ps1.
- Commit after success.

## 2026-07-01

### Added platform login and session groundwork

Added:

- migrations/000005_platform_sessions.up.sql
- migrations/000005_platform_sessions.down.sql
- internal/platform/controlapi/login.go
- scripts/verify-platform-session.ps1
- docs/decisions/0018-platform-login-sessions.md

Updated:

- internal/platform/auth/platform_store.go
- internal/platform/controlapi/server.go
- internal/platform/controlapi/auth.go
- scripts/seed-platform-superadmin.ps1
- scripts/verify-all.ps1

Behavior:

- POST /control/v1/auth/login returns a platform session token for an active superadmin.
- Protected control-plane routes accept X-Platform-Session.
- X-Platform-User-ID remains as a temporary fallback.
- X-Platform-Role continues to be ignored for authorization.

Reason:

- Platform auth needs real login/session groundwork.
- Direct X-Platform-User-ID should eventually be removed.

Current next step:

- Apply migration 000005_platform_sessions.
- Seed platform superadmin with password.
- Restart control-plane-api.
- Run verify-platform-session.ps1.
- Run verify-all.ps1.
- Commit after success.

## 2026-07-01

### Moved control-plane verification scripts to platform sessions

Added:

- scripts/Get-PlatformSessionHeaders.ps1
- docs/decisions/0019-control-plane-verification-uses-platform-sessions.md

Updated:

- scripts/verify-control-plane-auth.ps1
- scripts/verify-rbac.ps1
- scripts/verify-module-entitlement.ps1
- scripts/verify-audit-outbox.ps1
- scripts/verify-outbox-worker.ps1
- scripts/verify-plans-subscriptions.ps1
- scripts/verify-control-plane-audit.ps1

Behavior:

- Verification scripts now use X-Platform-Session for control-plane API calls.
- X-Platform-User-ID fallback still exists in server code for this step.
- verify-control-plane-auth.ps1 checks missing auth, invalid session, and valid DB-backed session.

Reason:

- Scripts must prove the session path before the temporary platform user header fallback is removed.

Current next step:

- Run verify-all.ps1.
- Commit after success.

## 2026-07-01

### Removed temporary X-Platform-User-ID fallback from control-plane auth

Added:

- docs/decisions/0020-control-plane-requires-platform-sessions.md

Updated:

- internal/platform/controlapi/auth.go
- scripts/verify-control-plane-auth.ps1

Behavior:

- Protected /control/v1/* endpoints now require X-Platform-Session.
- X-Platform-User-ID is no longer accepted as a fallback.
- verify-control-plane-auth.ps1 now proves the old fallback is rejected.

Reason:

- Platform login/session support exists.
- Direct platform user identity headers should not authorize control-plane access.

Current next step:

- Restart control-plane-api.
- Run verify-control-plane-auth.ps1.
- Run verify-all.ps1.
- Commit after success.

## 2026-07-01

### Added platform logout/session revoke

Added:

- scripts/verify-platform-logout.ps1
- docs/decisions/0021-platform-logout-revokes-sessions.md

Updated:

- internal/platform/auth/platform_store.go
- internal/platform/controlapi/login.go
- internal/platform/controlapi/server.go
- scripts/verify-all.ps1

Behavior:

- POST /control/v1/auth/logout revokes the current platform session.
- Revoked sessions are rejected on future control-plane requests.
- verify-platform-logout.ps1 proves a session works before logout and fails after logout.

Reason:

- Platform sessions need explicit lifecycle control.

Current next step:

- Restart control-plane-api.
- Run verify-platform-logout.ps1.
- Run verify-all.ps1.
- Commit after success.

## 2026-07-01

### Added platform login/logout audit logging

Added:

- scripts/verify-platform-auth-audit.ps1
- docs/decisions/0022-platform-auth-login-logout-audit.md

Updated:

- internal/platform/auth/platform_store.go
- scripts/verify-all.ps1

Behavior:

- Successful platform login writes control.platform_auth.login to control.platform_audit_log.
- Successful platform logout writes control.platform_auth.logout to control.platform_audit_log.
- verify-platform-auth-audit.ps1 proves both audit counts increase.

Reason:

- Platform auth lifecycle events are security-relevant control-plane events.

Current next step:

- Restart control-plane-api.
- Run verify-platform-auth-audit.ps1.
- Run verify-all.ps1.
- Commit after success.

## 2026-07-01

### Added failed platform login audit and lockout groundwork

Added:

- migrations/000006_platform_login_security.up.sql
- migrations/000006_platform_login_security.down.sql
- scripts/verify-platform-login-security.ps1
- docs/decisions/0023-failed-platform-login-audit-lockout.md

Updated:

- internal/platform/auth/platform_store.go
- internal/platform/controlapi/login.go
- scripts/seed-platform-superadmin.ps1
- scripts/verify-all.ps1

Behavior:

- Failed platform login writes control.platform_auth.login_failed audit row.
- Platform users track failed_login_count, last_failed_login_at, and locked_until.
- Five failed logins temporarily lock the platform user.
- Locked login attempts return HTTP 423.
- The dev superadmin seed script resets local lockout state.

Reason:

- Platform auth should have basic brute-force protection groundwork.

Current next step:

- Apply migration 000006_platform_login_security.
- Restart control-plane-api.
- Run verify-platform-login-security.ps1.
- Run verify-all.ps1.
- Commit after success.

## 2026-07-01

### Added platform session management

Added:

- scripts/verify-platform-session-management.ps1
- docs/decisions/0024-platform-session-management.md

Updated:

- internal/platform/auth/platform_store.go
- internal/platform/controlapi/login.go
- internal/platform/controlapi/server.go
- scripts/verify-all.ps1

Behavior:

- GET /control/v1/auth/sessions lists active sessions for the current platform user.
- POST /control/v1/auth/sessions/revoke-all revokes all active sessions for the current platform user.
- POST /control/v1/auth/sessions/cleanup-expired revokes expired active sessions.
- verify-platform-session-management.ps1 proves two sessions can be listed and then revoked.

Reason:

- Platform auth needs basic session visibility and cleanup/revoke controls.

Current next step:

- Restart control-plane-api.
- Run verify-platform-session-management.ps1.
- Run verify-all.ps1.
- Commit after success.

## 2026-07-01

### Added tenant-user ERP session foundation

Added:

- migrations/000007_tenant_user_sessions.up.sql
- migrations/000007_tenant_user_sessions.down.sql
- internal/platform/auth/tenant_session_store.go
- cmd/erp-api/auth_handlers.go
- scripts/verify-tenant-session.ps1
- docs/decisions/0025-tenant-user-erp-sessions.md

Updated:

- scripts/seed-dev-rbac.ps1
- cmd/erp-api/main.go
- scripts/verify-all.ps1

Behavior:

- POST /api/v1/auth/login authenticates a tenant user and returns session_token.
- GET /api/v1/auth/me resolves X-Tenant-ID + X-ERP-Session.
- POST /api/v1/auth/logout revokes the ERP session.
- Existing business APIs still use X-User-ID for this step.

Reason:

- ERP APIs need tenant-user login/session groundwork before removing temporary X-User-ID headers.

Current next step:

- Apply migration 000007_tenant_user_sessions.
- Restart erp-api.
- Run verify-tenant-session.ps1.
- Run verify-all.ps1.
- Commit after success.

## 2026-07-01

### Migrated Inventory APIs to ERP session auth

Added:

- cmd/erp-api/session_middleware.go
- scripts/Get-TenantSessionHeaders.ps1
- scripts/verify-inventory-session-auth.ps1
- docs/decisions/0026-inventory-apis-require-erp-sessions.md

Updated:

- cmd/erp-api/main.go
- scripts/verify-rbac.ps1
- scripts/verify-module-entitlement.ps1
- scripts/verify-audit-outbox.ps1
- scripts/verify-outbox-worker.ps1
- scripts/verify-tenant-isolation.ps1
- scripts/verify-all.ps1

Behavior:

- Inventory APIs require X-Tenant-ID and X-ERP-Session.
- X-User-ID is rejected for Inventory APIs.
- ERP session resolves tenant user and feeds RBAC/audit context.

Reason:

- Business APIs should not trust caller-supplied user IDs.

Current next step:

- Restart erp-api.
- Run verify-inventory-session-auth.ps1.
- Run verify-all.ps1.
- Commit after success.

## 2026-07-01

### Added automatic local API restart for verification

Added:

- scripts/restart-local-apis.ps1
- scripts/ensure-local-apis.ps1
- docs/decisions/0027-local-verification-restarts-apis.md

Updated:

- scripts/verify-all.ps1
- scripts/verify-inventory-session-auth.ps1
- scripts/verify-rbac.ps1
- scripts/verify-tenant-isolation.ps1

Behavior:

- verify-all.ps1 restarts control-plane-api and erp-api before verification.
- Key individual verification scripts also restart APIs when run directly.
- Restart kills stale processes on ports 8080 and 8081, starts fresh API windows, and waits for health/db health.

Reason:

- Avoid stale-process failures after code changes.

## 2026-07-01

### Fixed Inventory ERP session entitlement path

Updated:

- cmd/erp-api/inventory_session_handler.go
- scripts/verify-module-entitlement.ps1

Behavior:

- Inventory API still requires X-Tenant-ID and X-ERP-Session.
- The session-backed Inventory handler now preserves requireModule("inventory") entitlement enforcement.
- Module entitlement verification now uses Invoke-WebRequest -SkipHttpErrorCheck so failures report exact HTTP status and body.

Reason:

- The first session-backed Inventory route bypassed module entitlement checks.

## 2026-07-01

### Added ERP route wiring regression guard

Added:

- scripts/verify-erp-route-wiring.ps1
- docs/decisions/0029-erp-route-wiring-regression-guard.md

Updated:

- scripts/verify-all.ps1

Behavior:

- verify-all.ps1 now checks ERP route wiring statically.
- The check ensures Inventory is routed through session auth and module entitlement.
- The check prevents reintroducing internal X-User-ID injection in the Inventory session handler.

Reason:

- Inventory session migration exposed route wiring fragility.

## 2026-07-01

### Extracted ERP middleware and guards

Added:

- cmd/erp-api/middleware.go
- docs/decisions/0030-erp-middleware-extracted.md

Updated:

- cmd/erp-api/main.go

Behavior:

- No intended behavior change.
- requireModule, requirePermission, tenantMiddleware, and userMiddleware moved out of main.go.

Reason:

- Keep cmd/erp-api/main.go closer to process boot only.

## 2026-07-01

### Extracted ERP health handlers

Added:

- cmd/erp-api/health_handlers.go
- docs/decisions/0032-erp-health-handlers-extracted.md

Updated:

- cmd/erp-api/main.go

Behavior:

- No intended behavior change.
- healthHandler and dbHealthHandler moved out of main.go.

Reason:

- Keep cmd/erp-api/main.go closer to process boot only.

## 2026-07-01

### Extracted ERP app construction

Added:

- cmd/erp-api/app.go
- docs/decisions/0033-erp-app-construction-extracted.md

Updated:

- cmd/erp-api/main.go

Behavior:

- No intended behavior change.
- app struct and newApp constructor now live in app.go.
- main.go is closer to process boot only.

Reason:

- Keep ERP API boot, wiring, and handlers separated.

## 2026-07-02

### Added Inventory Units of Measure

Added:

- migrations/000008_inventory_units.up.sql
- migrations/000008_inventory_units.down.sql
- internal/modules/inventory/units.go
- cmd/erp-api/inventory_units_handlers.go
- scripts/verify-inventory-units.ps1
- docs/decisions/0034-inventory-units-of-measure.md

Updated:

- cmd/erp-api/routes.go
- scripts/seed-dev-rbac.ps1
- scripts/verify-all.ps1

Behavior:

- GET /api/v1/inventory/units lists tenant units.
- POST /api/v1/inventory/units creates tenant units.
- Unit codes are normalized to uppercase.
- Unit create writes audit and outbox records.

Reason:

- Units of Measure are foundational ERP master data needed before stock movements and item/unit links.

## 2026-07-02

### Linked Inventory Items to Base Units of Measure

Added:

- migrations/000009_inventory_item_base_units.up.sql
- migrations/000009_inventory_item_base_units.down.sql
- scripts/verify-inventory-item-units.ps1
- docs/decisions/0035-inventory-item-base-units.md

Updated:

- internal/modules/inventory/store.go
- cmd/erp-api/business_handlers.go
- scripts/verify-all.ps1
- docs/PROJECT_STATE.md
- docs/AI_CHANGELOG.md

Behavior:

- Inventory items now have base_unit_id support.
- New item creation requires base_unit_id.
- The base unit must be tenant-owned and active.
- Item list/create responses include base_unit_id and compact base_unit details.
- Item create audit/outbox records include base-unit information.
- verify-all.ps1 now includes inventory item/base unit verification.

Reason:

- Inventory items need a base Unit of Measure before stock movement, purchase, sales, costing, and future conversion workflows can be modeled safely.

### Fixed Inventory Session Auth Verifier Tenant Headers

Updated:

- scripts/verify-inventory-session-auth.ps1

Behavior:

- The verifier now creates its temporary Unit of Measure using tenant ERP session headers.
- The session-auth item creation test still uses the intended X-ERP-Session path.

Reason:

- The temporary unit creation request is tenant-scoped and must include X-Tenant-ID.

### Updated RBAC Verification for Inventory Item Base Units

Updated:

- scripts/verify-rbac.ps1

Behavior:

- RBAC verification now creates an active Unit of Measure before creating an inventory item.
- The allowed-user inventory item payload now includes base_unit_id.

Reason:

- Inventory item creation now requires base_unit_id after linking items to Units of Measure.

### Updated Tenant Isolation Verification for Inventory Item Base Units

Updated:

- scripts/verify-tenant-isolation.ps1

Behavior:

- Tenant isolation verification now creates an active Unit of Measure before creating an inventory item.
- The tenant-scoped inventory item payload now includes base_unit_id.

Reason:

- Inventory item creation now requires base_unit_id after linking items to Units of Measure.

### Updated Audit Outbox Verification for Inventory Item Base Units

Updated:

- scripts/verify-audit-outbox.ps1

Behavior:

- Audit/outbox verification now creates an active Unit of Measure before creating an inventory item.
- The inventory item payload now includes base_unit_id.

Reason:

- Inventory item creation now requires base_unit_id after linking items to Units of Measure.

### Swept Inventory Item Verification Scripts for Base Units

Updated:

- scripts/verify-inventory-session-auth.ps1
- scripts/verify-rbac.ps1
- scripts/verify-tenant-isolation.ps1
- scripts/verify-audit-outbox.ps1

Behavior:

- Inventory item verification scripts now create an active Unit of Measure before creating inventory items.
- Inventory item verification payloads now include base_unit_id.

Reason:

- Inventory item creation now requires base_unit_id after linking items to Units of Measure.
- Existing verification scripts still used the pre-base-UOM item payload shape.

### Updated Outbox Worker Verification for Inventory Item Base Units

Updated:

- scripts/verify-outbox-worker.ps1

Behavior:

- Outbox worker verification now creates an active Unit of Measure before creating an inventory item.
- The inventory item payload now includes base_unit_id.

Reason:

- Inventory item creation now requires base_unit_id after linking items to Units of Measure.

## 2026-07-02

### Added Inventory Locations

Added:

- migrations/000010_inventory_locations.up.sql
- migrations/000010_inventory_locations.down.sql
- internal/modules/inventory/locations.go
- cmd/erp-api/inventory_locations_handlers.go
- scripts/verify-inventory-locations.ps1
- docs/decisions/0036-inventory-locations.md

Updated:

- cmd/erp-api/routes.go
- scripts/seed-dev-rbac.ps1
- scripts/verify-all.ps1
- docs/PROJECT_STATE.md
- docs/AI_CHANGELOG.md

Behavior:

- GET /api/v1/inventory/locations lists tenant inventory locations.
- POST /api/v1/inventory/locations creates tenant inventory locations.
- Location codes are normalized to uppercase.
- Location create writes audit and outbox records.
- Dev RBAC seed now grants inventory.location.read and inventory.location.write to the allowed dev user.
- verify-all.ps1 now includes inventory location verification.

Reason:

- Inventory locations are required before stock movements, stock ledgers, availability, receiving, and fulfillment can be modeled.

## 2026-07-02

### Added Inventory Stock Ledger

Added:

- migrations/000011_inventory_stock_movements.up.sql
- migrations/000011_inventory_stock_movements.down.sql
- internal/modules/inventory/stock.go
- cmd/erp-api/inventory_stock_handlers.go
- scripts/verify-inventory-stock.ps1
- docs/decisions/0037-inventory-stock-ledger.md

Updated:

- cmd/erp-api/routes.go
- scripts/seed-dev-rbac.ps1
- scripts/verify-all.ps1
- docs/PROJECT_STATE.md
- docs/AI_CHANGELOG.md

Behavior:

- POST /api/v1/inventory/stock-movements creates posted stock movements.
- GET /api/v1/inventory/stock-movements lists recent posted stock movements.
- GET /api/v1/inventory/stock-balances returns balances derived from stock movement lines.
- Stock movement creation writes audit and outbox records.
- Dev RBAC seed now grants stock movement and stock balance permissions.
- verify-all.ps1 now includes inventory stock verification.

Reason:

- Inventory stock must be ledger-based and auditable before purchase receipts, sales issues, availability, costing, and accounting integration can be modeled.

## 2026-07-02

### Added Purchase Receipts Stock Integration

Added:

- migrations/000012_purchase_receipts.up.sql
- migrations/000012_purchase_receipts.down.sql
- internal/modules/purchase/receipts.go
- cmd/erp-api/purchase_receipts_handlers.go
- scripts/verify-purchase-receipts.ps1
- docs/decisions/0038-purchase-receipts-stock-integration.md

Updated:

- cmd/erp-api/app.go
- cmd/erp-api/routes.go
- scripts/seed-dev-rbac.ps1
- scripts/verify-all.ps1
- docs/PROJECT_STATE.md
- docs/AI_CHANGELOG.md

Behavior:

- POST /api/v1/purchase/receipts creates posted purchase receipts.
- Purchase receipt creation also creates inventory stock movements with movement_type = receipt.
- Receipt lines become positive inventory stock movement lines.
- GET /api/v1/purchase/receipts lists recent tenant receipts.
- Dev RBAC seed now grants purchase.receipt.read and purchase.receipt.write to the allowed dev user.
- verify-all.ps1 now includes purchase receipt verification.

Reason:

- Purchased goods must enter inventory through the stock ledger before purchase orders, suppliers, payables, and accounting integration can be modeled safely.

## 2026-07-03

### Added Purchase Suppliers

Added:

- migrations/000013_purchase_suppliers.up.sql
- migrations/000013_purchase_suppliers.down.sql
- internal/modules/purchase/suppliers.go
- cmd/erp-api/purchase_suppliers_handlers.go
- scripts/verify-purchase-suppliers.ps1
- docs/decisions/0039-purchase-suppliers.md

Updated:

- cmd/erp-api/routes.go
- manifests/purchase.module.yaml
- scripts/seed-dev-rbac.ps1
- scripts/verify-erp-route-wiring.ps1
- scripts/verify-all.ps1
- README.md
- docs/NATIVE_LOCAL_DEV.md
- docs/PROJECT_STATE.md
- docs/AI_CHANGELOG.md

Behavior:

- GET /api/v1/purchase/suppliers lists tenant suppliers.
- POST /api/v1/purchase/suppliers creates tenant suppliers.
- Supplier codes are normalized to uppercase and unique per tenant.
- Duplicate supplier codes return HTTP 409 with supplier_code_exists.
- Supplier APIs require ERP sessions, Purchase entitlement, and purchase.supplier RBAC permissions.
- Supplier creation writes audit and outbox records in the same tenant-scoped transaction.
- The focused verifier proves tenant isolation and allows the same supplier code in different tenants.
- README and native-development identity notes now reflect platform and ERP session headers.

Reason:

- Purchase Orders and later payable workflows need stable tenant-owned supplier master data.

Current next step:

- Apply migration 000013_purchase_suppliers.
- Restart local APIs.
- Run scripts/verify-purchase-suppliers.ps1.
- Run scripts/verify-all.ps1.

## 2026-07-03

### Added Purchase Orders

Added:

- migrations/000014_purchase_orders.up.sql
- migrations/000014_purchase_orders.down.sql
- internal/modules/purchase/orders.go
- cmd/erp-api/purchase_orders_handlers.go
- scripts/verify-purchase-orders.ps1
- docs/decisions/0040-purchase-orders.md

Updated:

- cmd/erp-api/routes.go
- manifests/purchase.module.yaml
- scripts/seed-dev-rbac.ps1
- scripts/verify-erp-route-wiring.ps1
- scripts/verify-all.ps1
- docs/PROJECT_STATE.md
- docs/AI_CHANGELOG.md

Behavior:

- Purchase Orders can be listed, created as drafts, and approved.
- Orders reference tenant suppliers and Inventory items through tenant-safe foreign keys.
- Approval is a one-way draft-to-approved transition.
- Create and approve write audit and outbox records atomically.
- Purchase Orders do not mutate Inventory stock.
- Focused verification covers validation, RBAC, tenant isolation, lifecycle conflicts, audit/outbox, and absence of stock movements.
- Fixed run-erp-worker.ps1 so successful worker execution returns to its caller instead of terminating verify-all.ps1 early.

Current next step:

- Apply migration 000014_purchase_orders.
- Run scripts/verify-purchase-orders.ps1.
- Run scripts/verify-all.ps1.

## 2026-07-03

### Required Approved Purchase Orders for Receipts

Added:

- migrations/000015_purchase_receipt_order_link.up.sql
- migrations/000015_purchase_receipt_order_link.down.sql
- docs/decisions/0041-purchase-receipts-require-approved-orders.md

Updated:

- internal/modules/purchase/receipts.go
- cmd/erp-api/purchase_receipts_handlers.go
- scripts/verify-purchase-receipts.ps1
- README.md
- docs/PROJECT_STATE.md
- docs/AI_CHANGELOG.md

Behavior:

- New receipts require purchase_order_id.
- The order must be approved and tenant-owned.
- Supplier data is derived from the linked order.
- Only ordered items can be received.
- Cumulative receipts cannot exceed ordered quantity.
- Purchase Order locking serializes concurrent remaining-quantity checks.
- Receipt, stock movement, audit, and outbox writes remain atomic.

Current next step:

- Apply migration 000015_purchase_receipt_order_link.
- Restart the APIs.
- Run scripts/verify-purchase-receipts.ps1.
- Run scripts/verify-all.ps1.

## 2026-07-06

### Added Purchase Order Receipt Progress

Added:

- migrations/000016_purchase_order_receipt_progress.up.sql
- migrations/000016_purchase_order_receipt_progress.down.sql
- docs/decisions/0042-purchase-order-receipt-progress.md

Updated:

- internal/modules/purchase/orders.go
- internal/modules/purchase/receipts.go
- cmd/erp-api/purchase_orders_handlers.go
- cmd/erp-api/purchase_receipts_handlers.go
- manifests/purchase.module.yaml
- scripts/verify-purchase-orders.ps1
- scripts/verify-purchase-receipts.ps1
- README.md
- docs/PROJECT_STATE.md
- docs/AI_CHANGELOG.md

Behavior:

- Purchase Order lines return received_quantity and remaining_quantity.
- Receipt transactions move orders to partially_received or received.
- Lifecycle transitions write audit and outbox records atomically.
- Duplicate item lines are rejected during Purchase Order creation.
- Purchase Orders accept additional receipts only while approved or partially_received.

Current next step:

- Apply migration 000016_purchase_order_receipt_progress.
- Restart the APIs.
- Run scripts/verify-purchase-orders.ps1.
- Run scripts/verify-purchase-receipts.ps1.
- Run scripts/verify-all.ps1.

## 2026-07-06

### Added Purchase Receipt Reversal

Added:

- migrations/000017_purchase_receipt_reversal.up.sql
- migrations/000017_purchase_receipt_reversal.down.sql
- internal/modules/purchase/reversals.go
- docs/decisions/0043-purchase-receipt-reversal.md

Updated:

- internal/modules/purchase/receipts.go
- internal/modules/purchase/orders.go
- cmd/erp-api/purchase_receipts_handlers.go
- cmd/erp-api/routes.go
- manifests/purchase.module.yaml
- scripts/seed-dev-rbac.ps1
- scripts/verify-purchase-receipts.ps1
- scripts/verify-erp-route-wiring.ps1
- README.md
- docs/PROJECT_STATE.md
- docs/AI_CHANGELOG.md

Behavior:

- Posted receipts can be reversed once with a required reason.
- Reversal appends a compensating receipt_reversal Inventory movement.
- Original receipt and stock ledger facts remain intact.
- Reversed receipts no longer contribute to Purchase Order receipt progress.
- Purchase Orders reopen as partially_received or approved.
- Reversal and reopening write tenant audit and outbox records atomically.

Follow-up fix:

- Cast derived Purchase Order received_quantity and remaining_quantity values to NUMERIC(18, 3) before text serialization.
- This preserves the API quantity contract as 0.000 instead of 0 after all receipts are reversed.

Current next step:

- Apply migration 000017_purchase_receipt_reversal.
- Restart the APIs.
- Run scripts/verify-purchase-receipts.ps1.
- Run scripts/verify-all.ps1.

## 2026-07-06

### Added Sales Customers

Added:

- migrations/000018_sales_customers.up.sql
- migrations/000018_sales_customers.down.sql
- internal/modules/sales/customers.go
- cmd/erp-api/sales_customers_handlers.go
- scripts/verify-sales-customers.ps1
- docs/decisions/0044-sales-customers.md

Updated:

- cmd/erp-api/app.go
- cmd/erp-api/routes.go
- manifests/sales.module.yaml
- scripts/seed-dev-rbac.ps1
- scripts/verify-erp-route-wiring.ps1
- scripts/verify-all.ps1
- README.md
- docs/NATIVE_LOCAL_DEV.md
- docs/PROJECT_STATE.md
- docs/AI_CHANGELOG.md
- AGENTS.md

Behavior:

- Sales Customers can be listed and created through session-authenticated Sales APIs.
- Customer codes are normalized and tenant-scoped.
- Duplicate customer codes return customer_code_exists.
- Customer creation writes audit and outbox records atomically.
- Focused verification covers tenant isolation, RBAC, entitlement, validation, audit, and outbox behavior.

Workflow:

- Human-facing PowerShell residue scans no longer assume rg is installed.
- Agents must not modify the human environment to install rg without explicit permission.

Current next step:

- Apply migration 000018_sales_customers.
- Restart the APIs.
- Run scripts/verify-sales-customers.ps1.
- Run scripts/verify-all.ps1.

## 2026-07-06

### Added Sales Orders

Added:

- migrations/000019_sales_orders.up.sql
- migrations/000019_sales_orders.down.sql
- internal/modules/sales/orders.go
- cmd/erp-api/sales_orders_handlers.go
- scripts/verify-sales-orders.ps1
- docs/decisions/0045-sales-orders.md

Updated:

- cmd/erp-api/routes.go
- manifests/sales.module.yaml
- scripts/seed-dev-rbac.ps1
- scripts/verify-erp-route-wiring.ps1
- scripts/verify-all.ps1
- README.md
- docs/NATIVE_LOCAL_DEV.md
- docs/PROJECT_STATE.md
- docs/AI_CHANGELOG.md

Behavior:

- Sales Orders can be listed, created as drafts, and confirmed once.
- Orders reference tenant Sales Customers and Inventory items without directly writing Inventory tables.
- Creation and confirmation write audit and outbox records atomically.
- Focused verification covers validation, tenant isolation, RBAC, lifecycle conflicts, audit/outbox records, and absence of stock mutation.

Current next step:

- Apply migration 000019_sales_orders.
- Restart the APIs.
- Run scripts/verify-sales-orders.ps1.
- Run scripts/verify-all.ps1.

## 2026-07-06

### Added Posted Sales Issues

Added:

- migrations/000020_sales_issues.up.sql
- migrations/000020_sales_issues.down.sql
- internal/modules/inventory/issue.go
- internal/modules/sales/issues.go
- cmd/erp-api/sales_issues_handlers.go
- scripts/verify-sales-issues.ps1
- docs/decisions/0046-sales-issues.md

Updated:

- internal/modules/sales/orders.go
- cmd/erp-api/routes.go
- manifests/inventory.module.yaml
- manifests/sales.module.yaml
- scripts/seed-dev-rbac.ps1
- scripts/verify-erp-route-wiring.ps1
- scripts/verify-all.ps1
- README.md
- docs/NATIVE_LOCAL_DEV.md
- docs/PROJECT_STATE.md
- docs/AI_CHANGELOG.md

Behavior:

- Confirmed Sales Orders can be fulfilled with posted Sales Issues.
- Sales validates ordered and remaining quantities.
- Inventory owns location-level availability checks and negative ledger writes.
- Order lines derive issued and remaining quantities.
- Orders transition to partially_fulfilled and fulfilled.
- Sales, Inventory, audit, outbox, and progress writes commit atomically.
- The focused verifier compares audit and outbox text identifiers without incompatible UUID casts.

Current next step:

- Apply migration 000020_sales_issues.
- Restart the APIs.
- Run scripts/verify-sales-issues.ps1.
- Run scripts/verify-all.ps1.

## 2026-07-06

### Added Sales Issue Reversal

Added:

- migrations/000021_sales_issue_reversal.up.sql
- migrations/000021_sales_issue_reversal.down.sql
- internal/modules/inventory/issue_reversal.go
- internal/modules/sales/reversals.go
- docs/decisions/0047-sales-issue-reversal.md

Updated:

- internal/modules/sales/issues.go
- internal/modules/sales/orders.go
- cmd/erp-api/sales_issues_handlers.go
- cmd/erp-api/routes.go
- manifests/sales.module.yaml
- scripts/seed-dev-rbac.ps1
- scripts/verify-erp-route-wiring.ps1
- scripts/verify-sales-issues.ps1
- README.md
- docs/PROJECT_STATE.md
- docs/AI_CHANGELOG.md

Behavior:

- Posted Sales Issues can be reversed once with a required reason.
- Inventory appends a compensating positive issue_reversal movement.
- Original issue and stock ledger facts remain immutable.
- Reversed issues no longer contribute to fulfillment progress.
- Sales Orders reopen to partially_fulfilled or confirmed.
- Reversal and reopening write audit/outbox records atomically.

Current next step:

- Apply migration 000021_sales_issue_reversal.
- Restart the APIs.
- Run scripts/verify-sales-issues.ps1.
- Run scripts/verify-all.ps1.

## 2026-07-06

### Completed Inventory v1 — Steps 48–52

Added:

- migrations/000022_inventory_transfers
- migrations/000023_inventory_stock_counts
- migrations/000024_inventory_reservations
- migrations/000025_inventory_cost_layers
- migrations/000026_inventory_v1_closeout
- Inventory transfer, stock count, reservation, costing, valuation, summary, and ledger APIs
- scripts/verify-inventory-v1.ps1
- ADRs 0048 through 0052

Updated:

- Inventory RBAC seed, routes, route guard, module manifest, package documentation, README, native-development guide, project state, changelog, and verify-all integration
- Sales Issue and transfer availability now subtract active, unexpired reservations
- Direct stock movement creation now accepts adjustments only

Behavior:

- Transfers create balanced source/destination movement pairs.
- Counts append physical variances.
- Reservations change ATP without changing on-hand stock and can be released once.
- Cost layers are append-only and drive latest-effective valuation.
- Reports expose location valuation, item summary, and stock ledger.
- The Inventory manifest is active and aligned with the implemented location-based v1 model.

Current next step:

- Apply migrations 000022 through 000025.
- Restart the APIs.
- Run scripts/verify-inventory-v1.ps1.
- Run scripts/verify-all.ps1.

## 2026-07-06

### Fixed Ctrl+C shutdown for native servers

Changed:

- control-plane-api and erp-api now react to interrupt cancellation with bounded graceful HTTP shutdown.
- A second Ctrl+C regains default force-termination behavior during shutdown.
- API and worker PowerShell launchers build and invoke direct local executables instead of `go run` wrappers.
- `.local/` build output is ignored.
- Added static shutdown-wiring verification and verify-all integration.

Verification:

- Run scripts/verify-server-shutdown-wiring.ps1.
- Start each continuous launcher and press Ctrl+C once to confirm it exits.

## 2026-07-22

### Added one-command FE local integration setup

Added:

- `scripts/setup-fe-dev.ps1`
- `scripts/verify-fe-dev-setup.ps1`
- `docs/decisions/0054-fe-local-integration-bootstrap.md`

Changed:

- `scripts/apply-migration.ps1` now composes safely inside an aggregate setup process when a migration is already applied.
- `scripts/restart-local-apis.ps1` and `scripts/ensure-local-apis.ps1` now forward `-EnvFile` into the API launchers.
- Every focused verifier that invokes `ensure-local-apis.ps1` now forwards its own `-EnvFile` value.
- `scripts/verify-all.ps1` now includes the focused FE setup-helper contract check and forwards its environment file to API startup.
- README and native local-development documentation now describe the FE handoff workflow.

Behavior:

- One command checks databases, applies pending migrations, seeds local identities and entitlements, starts both APIs, validates login flows, and emits `.local/fe-integration.json`.
- The manifest returns base URLs, tenant and user IDs, test credentials, auth endpoints, required headers, and module lists.
- Temporary validation sessions are revoked unless `-IncludeSessionTokens` is explicitly supplied.
- Existing `.env` files are not overwritten and generated credential artifacts remain gitignored.

Verification:

- Run `scripts/verify-fe-dev-setup.ps1` for the static helper contract.
- Run `scripts/setup-fe-dev.ps1` for the live bootstrap.
- Run `scripts/verify-all.ps1` for the complete suite.

## 2026-07-22

### Added granular frontend integration teaching guide

Added:

- `docs/FE_INTEGRATION_GUIDE.md`
- `scripts/verify-fe-integration-docs.ps1`

Updated:

- README and native local-development documentation now direct frontend developers to the guide after bootstrap.
- `scripts/verify-all.ps1` now checks FE documentation coverage.
- Project state records the frontend integration contract and its known boundaries.

The guide combines architecture explanation with the live contracts for tenant/platform sessions, Vite proxying, TypeScript clients, session restoration, route/module/permission guards, all current endpoints, write DTOs, document lifecycles, error UX, and security review.

It now opens with progressive explanations for the same reader acting as student, product consumer, and developer. It defines platform superadmin authority versus tenant/module authority and traces the tenant ID from the setup parameter through seeding, live validation, terminal output, returned PowerShell object, handoff manifest, login request, and protected-request headers.

It explicitly records current gaps instead of presenting future behavior as available: there is no permission-discovery endpoint, browser CORS middleware, Accounting business API, or external HRMS/custom-SaaS handoff exchange yet.

Verification:

- Run `scripts/verify-fe-integration-docs.ps1`.
- Run `scripts/verify-all.ps1`.

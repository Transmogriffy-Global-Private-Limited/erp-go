# Project State

## Repository

Name: erp-go

Remote: https://github.com/Transmogriffy-Global-Private-Limited/erp-go.git

Local path:

C:\Users\AnubhabDey\Programs\My_Programs\erp-go

## Development environment

Human development machine:

- Windows 11
- PowerShell 7+
- VS Code

## Product direction

We are building a professional, fully online, multi-tenant ERP SaaS platform.

The current direction is fully online SaaS ERP with superadmin, licensing, tenant plans, and module subscriptions.

The earlier installable/offline module idea is not the current base, but module boundaries should remain clean enough that future packaging changes are possible.

## Important conversation history

### Initial vision

The first architectural idea was an installable ERP runtime where modules could run offline/local at the customer site.

That model had:

- local ERP node
- local database
- local filestore
- local event bus
- module packages
- cloud licensing/update server

### Direction changed

The requirement changed to a completely online ERP.

The architecture was rebased to:

- cloud-first SaaS
- central ERP runtime
- control plane
- superadmin panel
- tenant/module licensing
- fully online operation

### Licensing and superadmin confirmed

The licensing server/control-plane idea remains.

There will be a superadmin panel to manage:

- tenants
- plans
- module access
- subscriptions
- limits
- support access
- platform users
- usage
- billing hooks

### Tenant separation concern

Strong tenant data separation is mandatory.

The current stance:

- shared DB with tenant_id for standard tenants
- PostgreSQL Row Level Security for tenant tables
- tenant-scoped cache/events/files/jobs/logs
- optional dedicated DB/bucket/key for enterprise tenants
- selective E2EE/envelope encryption for sensitive files/fields
- no full ERP-wide E2EE promise

### FOSS ERP review

FOSS ERPs were considered as references only.

Suggested systems to study:

- ERPNext for product feel, metadata/docs/workflows
- Odoo Community for module/addon architecture
- iDempiere for enterprise seriousness and multi-org thinking
- Dolibarr for simplicity and SME UX

Decision: build our own platform, do not fork a FOSS ERP.

## Current technical direction

Backend: Go

Frontend later: React + TypeScript

Data/storage:

- PostgreSQL
- Redis
- NATS JetStream
- S3-compatible object storage

Initial backend shape: modular monolith

Future extraction: module services only when justified

## Current planes

### Control Plane

Owns platform-level concerns:

- tenants
- plans
- subscriptions
- licensing
- enabled modules
- module registry
- limits
- support access
- platform audit
- superadmin users

### ERP Plane

Owns tenant business concerns:

- tenant users
- roles
- permissions
- org/branches
- inventory
- sales
- purchase
- accounting
- documents
- workflow
- notifications
- reports

## Current module candidates

Initial platform modules:

- core
- auth
- tenancy
- RBAC
- licensing
- module registry
- documents
- audit
- workflow
- notifications
- reports

Initial business modules:

- inventory
- sales
- purchase
- accounting

Later modules:

- CRM
- HR
- payroll
- assets
- maintenance
- manufacturing
- projects
- integrations

## Current next step

After this AI/project-memory layer is created, next step should be:

Create minimal Go repo spine:

- README.md
- go.mod
- cmd/control-plane-api
- cmd/erp-api
- internal/platform/httpx
- internal/platform/tenancy

Do not start business modules before the platform spine boots.

## 2026-07-01 update

The minimal Go API spine was added.

New backend entrypoints:

- cmd/control-plane-api
- cmd/erp-api

New shared platform packages:

- internal/platform/httpx
- internal/platform/tenancy

Current verification target:

- gofmt ./cmd ./internal
- go test ./...
- go run ./cmd/control-plane-api
- go run ./cmd/erp-api
- Invoke-RestMethod http://localhost:8081/healthz
- Invoke-RestMethod http://localhost:8080/healthz
- Invoke-RestMethod http://localhost:8080/api/v1/modules -Headers @{ "X-Tenant-ID" = "00000000-0000-0000-0000-000000000001" }

Next recommended step:

- Add platform package directories and module manifests.

## 2026-07-01 update

Platform package map and initial module manifests were added.

New platform packages:

- internal/platform/auth
- internal/platform/rbac
- internal/platform/licensing
- internal/platform/modules
- internal/platform/events
- internal/platform/outbox
- internal/platform/files
- internal/platform/audit
- internal/platform/workflow

New business module packages:

- internal/modules/inventory
- internal/modules/sales
- internal/modules/purchase
- internal/modules/accounting

New manifests:

- manifests/inventory.module.yaml
- manifests/sales.module.yaml
- manifests/purchase.module.yaml
- manifests/accounting.module.yaml

Next recommended step:

- Add base database migrations for control/core/outbox and tenant isolation foundations.

## 2026-07-01 update

Database foundation migration was added.

New files:

- migrations/README.md
- migrations/000001_platform_foundation.up.sql
- migrations/000001_platform_foundation.down.sql
- docs/decisions/0004-database-foundation-rls.md

The migration creates schemas, control plane tables, tenant user/RBAC base, documents, audit, outbox, and sample inventory.items with RLS.

Next recommended step:

- Add Docker Compose infrastructure for local PostgreSQL, Redis, NATS JetStream, and S3-compatible object storage.

## 2026-07-01 update

A new development constraint was accepted:

- no Docker
- no Docker Compose
- no container-based local development

Future setup should use native Go tooling, PowerShell, native PostgreSQL or externally provided PostgreSQL, and non-container service options when needed.

Next recommended step:

- Add native database configuration and a PowerShell migration apply script using psql.

## 2026-07-01 update

Native local development workflow was added.

New files:

- .env.example
- scripts/Import-DotEnv.ps1
- scripts/db-check.ps1
- scripts/apply-migration.ps1
- docs/NATIVE_LOCAL_DEV.md

Updated:

- .gitignore

The workflow uses PowerShell and psql. No Docker or containers.

Next recommended step:

- Create local PostgreSQL roles/database.
- Copy .env.example to .env.
- Run database check and first migration.

## 2026-07-01 update

Native migration scripts were hardened.

Updated scripts now check native psql exit codes.

Added migration ledger support using:

- public.schema_migrations

Added helper:

- scripts/mark-migration-applied.ps1

Next recommended step:

- Mark the already-applied 000001_platform_foundation migration as applied.
- Verify apply-migration.ps1 skips it on rerun.

## 2026-07-01 update

Native migration workflow has been verified locally.

Verified state:

- PostgreSQL connection works through scripts/db-check.ps1
- 000001_platform_foundation was applied to local database
- public.schema_migrations exists
- 000001_platform_foundation is marked as applied
- scripts/apply-migration.ps1 -Direction up safely skips the already-applied migration
- branch anubhab-work is clean and pushed after commit 44a88d6

Current next recommended step:

- Connect Go services to PostgreSQL using ERP_DATABASE_URL.
- Add shared DB package.
- Add /healthz/db endpoint for erp-api and control-plane-api.
- Ensure runtime DB role is treated separately from migration owner role.

## 2026-07-01 update

Runtime PostgreSQL connectivity was added to both Go APIs.

New runtime DB package:

- internal/platform/db

New service health endpoints:

- GET /healthz/db on control-plane-api
- GET /healthz/db on erp-api

New scripts:

- scripts/run-control-plane-api.ps1
- scripts/run-erp-api.ps1

New migration:

- 000002_runtime_role_grants

Updated docs:

- README.md
- docs/NATIVE_LOCAL_DEV.md

Next recommended step:

- Apply 000002_runtime_role_grants.
- Run scripts/db-check.ps1.
- Start both APIs and verify /healthz/db.

## 2026-07-01 update

Migration script null-output handling was fixed.

Problem:

- Applying 000002_runtime_role_grants failed because scripts/apply-migration.ps1 called .Trim() on null when the migration was not yet in public.schema_migrations.

Fix:

- Capture psql output as an array.
- Join output safely.
- Trim the resulting string.

Verified before this fix:

- Database connectivity works for migration and runtime URLs.
- Both APIs return db_ok on /healthz/db.

Next recommended step:

- Apply 000002_runtime_role_grants and verify migration ledger.

## 2026-07-01 update

Module APIs are now database-backed.

New store:

- internal/platform/modules/store.go

Updated endpoints:

- GET /control/v1/modules reads from control.modules.
- GET /api/v1/modules reads from enabled tenant modules.

New local helper:

- scripts/seed-dev-tenant.ps1

Local dev tenant:

- 00000000-0000-0000-0000-000000000001

Next recommended step:

- Run seed-dev-tenant.ps1.
- Verify module endpoints from both services.

## 2026-07-01 update

Tenant-scoped DB execution foundation was added.

New helper:

- internal/platform/db.WithTenantTx

New ADR:

- docs/decisions/0006-tenant-scoped-db-transactions.md

New Inventory APIs:

- GET /api/v1/inventory/items
- POST /api/v1/inventory/items

Important rule:

- Tenant-owned module stores must use WithTenantTx or equivalent before touching tenant-owned tables.

Next recommended step:

- Verify Inventory item create/list using dev tenant.
- Then add a second tenant seed/check to prove RLS isolation.

## 2026-07-01 update

Tenant isolation verification tooling was added.

Updated:

- scripts/seed-dev-tenant.ps1

Added:

- scripts/verify-tenant-isolation.ps1

The verification script uses the ERP API to prove that an item created under tenant A is not visible under tenant B.

Next recommended step:

- Run scripts/verify-tenant-isolation.ps1.
- Commit the verification tooling after success.

## 2026-07-01 update

Tenant isolation verification was successfully run through the ERP API.

Verified flow:

- Seeded tenant A: 00000000-0000-0000-0000-000000000001
- Seeded tenant B: 00000000-0000-0000-0000-000000000002
- Created an inventory item under tenant A
- Confirmed tenant A can see its own item
- Confirmed tenant B cannot see tenant A item

Result:

- Tenant isolation verification passed.

This proves the current runtime path:

X-Tenant-ID
Go tenant context
WithTenantTx
PostgreSQL app.tenant_id
RLS policy
inventory.items

Current next recommended step:

- Make control-plane tenant APIs database-backed.

## 2026-07-01 update

Control-plane tenant APIs are now database-backed.

New store:

- internal/platform/tenancy/store.go

Updated endpoint behavior:

- GET /control/v1/tenants lists tenants from control.tenants.
- POST /control/v1/tenants creates tenants in control.tenants.

Next recommended step:

- Verify tenant list/create endpoints.
- Then add tenant module enable/disable endpoints.

## 2026-07-01 update

Control-plane tenant module entitlement APIs were added.

New endpoints:

- GET /control/v1/tenants/{tenant_id}/modules
- POST /control/v1/tenants/{tenant_id}/modules/{module_id}/enable
- POST /control/v1/tenants/{tenant_id}/modules/{module_id}/disable

New ADR:

- docs/decisions/0007-control-plane-module-entitlements.md

Next recommended step:

- Verify entitlement disable/enable behavior through both control-plane and ERP module endpoints.

## 2026-07-01 update

Control-plane tenant module entitlement APIs were verified.

Verified flow:

- Listed enabled modules for tenant 00000000-0000-0000-0000-000000000001
- Disabled inventory through control-plane API
- Confirmed inventory disappeared from control-plane tenant modules
- Confirmed inventory disappeared from ERP /api/v1/modules
- Re-enabled inventory through control-plane API
- Confirmed inventory reappeared in ERP /api/v1/modules

Result:

- Control-plane module entitlements now affect ERP module visibility.

Important next concern:

- Module-specific ERP APIs are not yet entitlement-gated.
- The next implementation step should prevent access to /api/v1/inventory/* when inventory is disabled for the tenant.

## 2026-07-01 update

ERP-side module entitlement enforcement was added.

New guard behavior:

- /api/v1/inventory/items requires inventory entitlement.
- Disabled module access returns 403 with code module_not_enabled.

New verification script:

- scripts/verify-module-entitlement.ps1

New ADR:

- docs/decisions/0008-erp-module-entitlement-enforcement.md

Next recommended step:

- Run verify-module-entitlement.ps1.
- Commit after verification succeeds.

## 2026-07-01 update

ERP-side module entitlement enforcement was successfully verified.

Verified flow:

- Ensured inventory module was enabled for tenant 00000000-0000-0000-0000-000000000001
- Confirmed /api/v1/inventory/items worked while inventory was enabled
- Disabled inventory through control-plane API
- Confirmed /api/v1/inventory/items returned 403 while inventory was disabled
- Re-enabled inventory
- Confirmed /api/v1/inventory/items worked again

Result:

- Module entitlement enforcement verification passed.

This proves:

- Control-plane entitlement changes affect ERP API access.
- ERP module APIs are now protected by module enablement, not merely hidden from /api/v1/modules.

Current next recommended step:

- Add tenant users, roles, permissions, and basic ERP permission enforcement.

## 2026-07-01 update

ERP-side RBAC permission enforcement was added.

New temporary identity context:

- X-User-ID

New permission store:

- internal/platform/rbac/store.go

New scripts:

- scripts/seed-dev-rbac.ps1
- scripts/verify-rbac.ps1

Updated verification scripts:

- scripts/verify-tenant-isolation.ps1
- scripts/verify-module-entitlement.ps1

New ADR:

- docs/decisions/0009-erp-permission-enforcement.md

Next recommended step:

- Verify RBAC, tenant isolation, and module entitlement scripts all pass.

## 2026-07-01 update

Audit and outbox writes were added to Inventory item creation.

New platform helpers:

- internal/platform/audit.Insert
- internal/platform/outbox.Insert

New verification script:

- scripts/verify-audit-outbox.ps1

New ADR:

- docs/decisions/0010-audit-outbox-mutations.md

Updated behavior:

- POST /api/v1/inventory/items writes inventory item, audit log, and outbox event in the same transaction.

Next recommended step:

- Verify audit/outbox and RBAC scripts.

## 2026-07-01 update

Audit/outbox verification script was fixed for RLS.

Problem:

- verify-audit-outbox.ps1 saw 0 audit rows because the audit count query did not preserve app.tenant_id for the RLS-protected audit.audit_log read.

Fix:

- The script now sets app.tenant_id in the same psql session before counting audit rows.

Next recommended step:

- Re-run audit/outbox and RBAC verification.
- Commit the audit/outbox mutation work.

## 2026-07-01 update

DB-backed outbox worker v1 was added.

New executable:

- cmd/erp-worker

New scripts:

- scripts/run-erp-worker.ps1
- scripts/verify-outbox-worker.ps1

New ADR:

- docs/decisions/0011-db-backed-outbox-worker-first.md

Updated outbox package:

- internal/platform/outbox/outbox.go

Current behavior:

- Worker claims pending/failed outbox events.
- Worker logs event payloads.
- Worker marks events published.

Next recommended step:

- Verify outbox worker.
- Re-run audit/outbox and RBAC checks.

## 2026-07-01 update

Master verification script was added.

New script:

- scripts/verify-all.ps1

It verifies:

- go test ./...
- database connectivity
- control-plane health
- ERP API health
- RBAC
- tenant isolation
- module entitlement enforcement
- audit/outbox mutation writes
- DB-backed outbox worker

Next recommended step:

- Run scripts/verify-all.ps1.
- Commit after success.

## 2026-07-01 update

Temporary control-plane auth guard was added.

New temporary headers:

- X-Platform-User-ID
- X-Platform-Role: superadmin

Protected route family:

- /control/v1/*

Public route family:

- /healthz
- /healthz/db

New verification script:

- scripts/verify-control-plane-auth.ps1

Next recommended step:

- Run verify-all.ps1.
- Commit after success.

## 2026-07-01 update

Temporary control-plane auth guard was successfully verified.

Verified behavior:

- /healthz remains public.
- /control/v1/modules without X-Platform-User-ID returns 401.
- /control/v1/modules with wrong X-Platform-Role returns 403.
- /control/v1/modules with X-Platform-Role: superadmin returns 200.
- verify-all.ps1 passed after updating verification scripts.

Result:

- Control-plane APIs are no longer open in local development.
- Temporary superadmin guard is working.

## 2026-07-01 update

Control-plane plans/subscriptions foundation was added.

New store:

- internal/platform/licensing/store.go

New endpoints:

- GET /control/v1/plans
- POST /control/v1/plans
- GET /control/v1/plans/{plan_id}/modules
- POST /control/v1/plans/{plan_id}/modules/{module_id}/enable
- POST /control/v1/tenants/{tenant_id}/subscription

New verification script:

- scripts/verify-plans-subscriptions.ps1

Next recommended step:

- Verify plans/subscriptions.
- Run full verification.

## 2026-07-01 update

Control-plane platform audit logging was added.

New table:

- control.platform_audit_log

New migration:

- 000003_control_plane_audit

New platform audit helper:

- internal/platform/audit.InsertPlatform

New verification script:

- scripts/verify-control-plane-audit.ps1

Audited actions include:

- tenant creation
- tenant module enable/disable
- plan creation
- plan module enable
- tenant subscription assignment

Next recommended step:

- Apply migration 000003.
- Verify control-plane audit.
- Run full verification.

## 2026-07-01 update

Control-plane API was refactored into a dedicated internal package.

New package:

- internal/platform/controlapi

Main boot file:

- cmd/control-plane-api/main.go

New ADR:

- docs/decisions/0015-split-control-plane-api-package.md

No behavior change intended.

Next recommended step:

- Run full verification and commit.

## 2026-07-01 update

Control-plane route parsing was cleaned and tested.

New files:

- internal/platform/controlapi/routes.go
- internal/platform/controlapi/routes_test.go

New ADR:

- docs/decisions/0016-tested-control-plane-route-parsing.md

No behavior change intended.

Next recommended step:

- Run full verification and commit.

## 2026-07-01 update

DB-backed platform superadmin authorization was added.

New table:

- control.platform_users

New migration:

- 000004_platform_users

New store:

- internal/platform/auth/platform_store.go

New seed script:

- scripts/seed-platform-superadmin.ps1

Updated behavior:

- control-plane auth checks DB role/status.
- X-Platform-Role is ignored for authorization.

Next recommended step:

- Apply migration 000004.
- Verify full project.

## 2026-07-01 update

Platform login/session groundwork was added.

New endpoint:

- POST /control/v1/auth/login

New table:

- control.platform_sessions

New script:

- scripts/verify-platform-session.ps1

New ADR:

- docs/decisions/0018-platform-login-sessions.md

Temporary state:

- X-Platform-Session is supported.
- X-Platform-User-ID fallback is still supported.
- X-Platform-Role remains ignored.

Next recommended step:

- Apply migration 000005.
- Verify sessions and full project.

## 2026-07-01 update

Control-plane verification scripts were moved to platform sessions.

New helper:

- scripts/Get-PlatformSessionHeaders.ps1

Updated scripts now call control-plane APIs with:

- X-Platform-Session

Temporary fallback still exists:

- X-Platform-User-ID

Next recommended step:

- Run full verification.
- Remove X-Platform-User-ID fallback in the following step.

## 2026-07-01 update

Control-plane auth now requires platform sessions.

Removed fallback:

- X-Platform-User-ID

Required protected control-plane header:

- X-Platform-Session

Updated verification:

- scripts/verify-control-plane-auth.ps1 confirms old X-Platform-User-ID fallback is rejected.

Next recommended step:

- Run full verification and commit.

## 2026-07-01 update

Platform logout/session revoke was added.

New endpoint:

- POST /control/v1/auth/logout

New verification script:

- scripts/verify-platform-logout.ps1

Updated behavior:

- Logout revokes the current platform session.
- Revoked session tokens are rejected.

Next recommended step:

- Verify and commit.

## 2026-07-01 update

Platform login/logout audit logging was added.

New audited actions:

- control.platform_auth.login
- control.platform_auth.logout

New verification script:

- scripts/verify-platform-auth-audit.ps1

Next recommended step:

- Verify and commit.

## 2026-07-01 update

Failed platform login audit and lockout groundwork was added.

New migration:

- 000006_platform_login_security

New audited action:

- control.platform_auth.login_failed

New verification script:

- scripts/verify-platform-login-security.ps1

Updated behavior:

- Five failed platform login attempts lock the platform user for fifteen minutes.
- Locked logins return HTTP 423.

Next recommended step:

- Apply migration 000006.
- Verify and commit.

## 2026-07-01 update

Platform session management was added.

New endpoints:

- GET /control/v1/auth/sessions
- POST /control/v1/auth/sessions/revoke-all
- POST /control/v1/auth/sessions/cleanup-expired

New verification script:

- scripts/verify-platform-session-management.ps1

Next recommended step:

- Verify and commit.

## 2026-07-01 update

Tenant-user ERP session foundation was added.

New endpoints:

- POST /api/v1/auth/login
- GET /api/v1/auth/me
- POST /api/v1/auth/logout

New session table:

- core.tenant_user_sessions

New verification script:

- scripts/verify-tenant-session.ps1

Temporary state:

- ERP auth session exists.
- Existing business APIs still accept X-User-ID.

Next recommended step:

- Verify tenant sessions.
- Then migrate protected ERP APIs to X-ERP-Session.

## 2026-07-01 update

Inventory APIs were migrated to ERP session auth.

Required headers:

- X-Tenant-ID
- X-ERP-Session

Rejected old header path:

- X-User-ID

New verification:

- scripts/verify-inventory-session-auth.ps1

Next recommended step:

- Verify and commit.

## 2026-07-01 update

Local verification now restarts APIs automatically.

New scripts:

- scripts/restart-local-apis.ps1
- scripts/ensure-local-apis.ps1

This prevents stale running API processes from causing false verification failures.

Next recommended step:

- Run full verification and commit.

## 2026-07-01 update

Inventory ERP session handler now preserves module entitlement enforcement.

Fix:

- session auth resolves tenant/user
- internal Inventory handler compatibility remains
- requireModule("inventory") is now applied before Inventory handler execution

Updated verification:

- scripts/verify-module-entitlement.ps1 now reports exact HTTP status/body.

## 2026-07-01 update

ERP route wiring regression guard was added.

New script:

- scripts/verify-erp-route-wiring.ps1

It prevents Inventory route/session/module-entitlement regressions.

Next recommended step:

- Run full verification and commit.

## 2026-07-01 update

ERP middleware and guard functions were extracted.

New file:

- cmd/erp-api/middleware.go

Moved functions:

- requireModule
- requirePermission
- tenantMiddleware
- userMiddleware

Next recommended step:

- Run full verification and commit.

## 2026-07-01 update

ERP health handlers were extracted.

New file:

- cmd/erp-api/health_handlers.go

Moved functions:

- healthHandler
- dbHealthHandler

Next recommended step:

- Run full verification and commit.

## 2026-07-01 update

ERP app construction was extracted.

New file:

- cmd/erp-api/app.go

New constructor:

- newApp(pool)

cmd/erp-api/main.go is now mostly process boot.

## 2026-07-02 update

Inventory Units of Measure were added.

New endpoints:

- GET /api/v1/inventory/units
- POST /api/v1/inventory/units

New table:

- inventory.units

New verification:

- scripts/verify-inventory-units.ps1

Next recommended step:

- Link inventory items to units after this passes.

## 2026-07-02 update

Inventory items were linked to base Units of Measure.

New migration:

- migrations/000009_inventory_item_base_units.up.sql
- migrations/000009_inventory_item_base_units.down.sql

Updated behavior:

- POST /api/v1/inventory/items now requires base_unit_id.
- base_unit_id must reference an active unit in the same tenant.
- GET /api/v1/inventory/items now returns base_unit_id and base_unit details.
- Inventory item create audit/outbox payloads include base-unit information.

New verification:

- scripts/verify-inventory-item-units.ps1

Note:

- inventory.items.base_unit_id is nullable at the database layer for now so existing rows do not break during migration.
- New API-created items require a base unit.

Next recommended step:

- Apply migration 000009.
- Run scripts/verify-inventory-item-units.ps1.
- Then run scripts/verify-all.ps1.

### Follow-up verification header fix

Inventory session auth verifier unit creation now uses tenant ERP session headers.

Updated:

- scripts/verify-inventory-session-auth.ps1

Reason:

- The temporary Unit of Measure creation request requires X-Tenant-ID before the verifier can create an item with base_unit_id.

### Follow-up RBAC verification fix

RBAC verification was updated for inventory item base_unit_id.

Updated:

- scripts/verify-rbac.ps1

Reason:

- The verifier still used the pre-base-UOM inventory item payload.
- It now creates an active Unit of Measure first and injects base_unit_id into the allowed inventory item create request.

### Follow-up tenant isolation verification fix

Tenant isolation verification was updated for inventory item base_unit_id.

Updated:

- scripts/verify-tenant-isolation.ps1

Reason:

- The verifier still used the pre-base-UOM inventory item payload.
- It now creates an active Unit of Measure first and injects base_unit_id into the tenant-scoped inventory item create request.

### Follow-up audit/outbox verification fix

Audit/outbox verification was updated for inventory item base_unit_id.

Updated:

- scripts/verify-audit-outbox.ps1

Reason:

- The verifier still used the pre-base-UOM inventory item payload.
- It now creates an active Unit of Measure first and injects base_unit_id into the inventory item create request.

### Follow-up verification sweep

Inventory item verification scripts were swept for base_unit_id.

Updated:

- scripts/verify-inventory-session-auth.ps1
- scripts/verify-rbac.ps1
- scripts/verify-tenant-isolation.ps1
- scripts/verify-audit-outbox.ps1

Reason:

- Multiple verifiers still created inventory items with the old payload.
- They now create active Units of Measure and inject base_unit_id before item creation.

### Follow-up outbox worker verification fix

Outbox worker verification was updated for inventory item base_unit_id.

Updated:

- scripts/verify-outbox-worker.ps1

Reason:

- The verifier still used the pre-base-UOM inventory item payload.
- It now creates an active Unit of Measure first and injects base_unit_id into the inventory item create request.

## 2026-07-02 update

Inventory Locations were added.

New endpoints:

- GET /api/v1/inventory/locations
- POST /api/v1/inventory/locations

New table:

- inventory.locations

New permissions:

- inventory.location.read
- inventory.location.write

New verification:

- scripts/verify-inventory-locations.ps1

Updated verification:

- scripts/verify-all.ps1 now includes Inventory locations verification.

Next recommended step:

- Add inventory stock movement / stock ledger foundation.

## 2026-07-02 update

Inventory Stock Ledger was added.

New endpoints:

- GET /api/v1/inventory/stock-movements
- POST /api/v1/inventory/stock-movements
- GET /api/v1/inventory/stock-balances

New tables:

- inventory.stock_movements
- inventory.stock_movement_lines

New permissions:

- inventory.stock_movement.read
- inventory.stock_movement.write
- inventory.stock_balance.read

New verification:

- scripts/verify-inventory-stock.ps1

Updated verification:

- scripts/verify-all.ps1 now includes Inventory stock verification.

Design note:

- Stock balances are derived from append-only movement lines.
- Item/location rows do not store mutable stock quantity.

Next recommended step:

- Add movement lifecycle or begin purchase receipt integration.

## 2026-07-02 update

Purchase Receipts Stock Integration was added.

New endpoints:

- GET /api/v1/purchase/receipts
- POST /api/v1/purchase/receipts

New tables:

- purchase.receipts
- purchase.receipt_lines

New permissions:

- purchase.receipt.read
- purchase.receipt.write

Behavior:

- Purchase receipt creation creates inventory stock movements with movement_type = receipt.
- Purchase receipt lines create positive inventory stock movement lines.
- Stock balances update through the existing inventory stock ledger.

New verification:

- scripts/verify-purchase-receipts.ps1

Updated verification:

- scripts/verify-all.ps1 now includes Purchase receipts verification.

Next recommended step:

- Add suppliers and purchase orders, or add sales issue integration into stock movements.

## 2026-07-03 update

Purchase Suppliers were added.

New endpoints:

- GET /api/v1/purchase/suppliers
- POST /api/v1/purchase/suppliers

New table:

- purchase.suppliers

New permissions:

- purchase.supplier.read
- purchase.supplier.write

Behavior:

- Supplier codes are normalized to uppercase and unique within each tenant.
- Supplier list/create uses ERP session auth, Purchase module entitlement, and tenant-user RBAC.
- Supplier rows use tenant-scoped transactions and PostgreSQL RLS.
- Supplier creation writes purchase.supplier.create audit and purchase.supplier.created.v1 outbox records atomically.
- Existing Purchase Receipts retain their free-text supplier_name contract for now.

New verification:

- scripts/verify-purchase-suppliers.ps1
- scripts/verify-all.ps1 now includes Purchase suppliers verification.
- The focused verifier covers validation, duplicate handling, RBAC, module entitlement, tenant isolation, tenant-scoped code uniqueness, audit, and outbox behavior.

Next recommended step:

- Add Purchase Orders linked to purchase.suppliers, with order lines that reference Inventory items but do not change stock until receipt posting.

## 2026-07-03 update

Purchase Orders were added.

New endpoints:

- GET /api/v1/purchase/orders
- POST /api/v1/purchase/orders
- POST /api/v1/purchase/orders/{purchase_order_id}/approve

New tables:

- purchase.purchase_orders
- purchase.purchase_order_lines

Behavior:

- Orders reference active tenant suppliers and active tenant Inventory items.
- Orders are created as draft and can transition once to approved.
- Quantities must be positive; unit prices must be non-negative.
- Currency codes are normalized and validated as three uppercase letters.
- Create and approve write audit/outbox records atomically.
- Draft and approved orders do not create stock movements.

New verification:

- scripts/verify-purchase-orders.ps1
- scripts/verify-all.ps1 now includes Purchase Order verification.
- The worker runner now returns normally on success so verify-all can complete its final outbox and suite assertions.

Next recommended step:

- Link Purchase Receipts to approved Purchase Orders and enforce supplier/item/order consistency while keeping Inventory stock posting inside the receipt transaction.

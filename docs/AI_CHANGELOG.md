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

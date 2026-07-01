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

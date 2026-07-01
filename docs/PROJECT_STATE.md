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

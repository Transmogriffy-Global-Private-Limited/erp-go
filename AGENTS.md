# AGENTS.md

## Project

Repository: erp-go

Remote: https://github.com/Transmogriffy-Global-Private-Limited/erp-go.git

This repository is for building a professional, fully online, multi-tenant ERP SaaS platform in Go.

The product includes:

- control plane
- superadmin panel
- tenant licensing
- module subscriptions
- ERP runtime
- strong tenant data isolation
- event-heavy module communication
- future enterprise isolation options

## Non-negotiable agent rules

### 1. Do not commit or push

Do not run git commit, git push, git reset --hard, or git clean -fd unless the human explicitly asks for that exact action.

Prefer providing patches, commands, and exact file edits.

### 2. One step at a time

Work in small, verifiable steps.

After every meaningful change, ask the human to run verification commands and paste output.

### 3. Maintain repo memory

After every meaningful architecture or code change, update:

- docs/PROJECT_STATE.md
- docs/AI_CHANGELOG.md
- relevant ADR under docs/decisions/
- AGENTS.md if agent rules change

If a future agent changes code but does not update project memory, that is a process bug.

### 4. Preserve the two-plane architecture

The platform has two major planes:

1. Control Plane
2. ERP Plane

The control plane manages:

- tenants
- plans
- subscriptions
- module entitlements
- licensing
- superadmin users
- billing hooks
- support access
- usage limits

The ERP plane manages:

- tenant users
- roles
- permissions
- inventory
- sales
- purchase
- accounting
- documents
- workflows
- reports

Do not blur these boundaries casually.

### 5. Strong tenant isolation is mandatory

Tenant separation must exist at every layer:

- API
- database
- cache
- event bus
- object storage
- background jobs
- realtime broadcasts
- audit logs
- support access

Never trust tenant_id supplied directly by the client.

Tenant context must eventually come from authenticated identity, trusted domain/subdomain routing, session, or another trusted server-side mechanism.

### 6. Superadmin is not God-mode business access

Superadmin users manage the platform.

They should not casually read or mutate tenant business data.

Support access must be:

- explicit
- scoped
- time-limited
- reason-required
- audited

### 7. ERP module ownership

Modules own their data.

Examples:

- Inventory owns stock.
- Accounting owns ledgers and journals.
- Documents owns files.
- Workflow owns approvals.
- Notification owns delivery.

One module must not directly write another module's tables.

Cross-module work happens through commands, queries, and events.

### 8. Command/event rule

APIs perform commands.
Events announce facts.
UI broadcasts invalidate views.
PostgreSQL stores truth.
Object storage stores files.

### 9. Prefer boring, reliable technology

Current intended stack:

- Go backend
- React + TypeScript frontend later
- PostgreSQL
- Redis
- NATS JetStream
- S3-compatible object storage
- Docker for local/dev deployment

### 10. Start modular monolith, not microservices

Initial backend should be a modular monolith with clean internal module boundaries.

Future extraction into services is allowed only when justified.

### 11. Security posture

Default security posture:

- TLS in transit
- encryption at rest
- tenant-scoped data everywhere
- PostgreSQL Row Level Security for shared tenant-owned tables
- least-privileged DB roles
- selective field/file encryption for sensitive data
- optional enterprise dedicated DB/bucket/key later

Do not promise full ERP-wide E2EE.

### 12. Update documentation before moving on

When making a change, include:

- what changed
- why it changed
- how to verify it
- what remains next

Update docs/AI_CHANGELOG.md with every meaningful change.

## Current human environment

Development machine:

- Windows 11
- PowerShell 7+
- VS Code

Repo path:

C:\Users\AnubhabDey\Programs\My_Programs\erp-go

## Current repo state

At the time this file was created:

- GitHub repository exists.
- Default GitHub .gitignore exists.
- Repo is cloned locally.
- No application code has been added yet.
- First intentional repo content is this AI/project-memory layer.

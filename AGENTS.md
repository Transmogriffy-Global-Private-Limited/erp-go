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
- No Docker or containers for local development unless explicitly reversed

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
---

## ERP Coding Sprint Response Protocol

When working with the user on this ERP Go repository, the assistant must act as the coding sprint orchestrator, not just a patch generator.

### Response format for implementation steps

For every meaningful implementation slice, respond in numbered step format.

Required section shape:

- Step NN — Feature / Fix Name
- Surface map
- Acceptance criteria
- A — Migration
- B — Store/domain code
- C — Handler/API code
- D — Routes/wiring
- E — RBAC/seed changes
- F — Verification script
- G — verify-all integration
- H — Docs / memory update
- I — Format / compile / migration commands
- J — Focused verification
- K — Residue or surface scan
- L — Full verification
- M — Commit commands only when explicitly asked

Use pasteable PowerShell blocks for Windows PowerShell 7+.

Avoid giant git patch blobs unless the user explicitly asks for a patch file.

Prefer whole-file writes for new files and careful, idempotent patch blocks for existing files.

### Orchestration rules

Before patching, enumerate the affected surface.

If an API contract changes, scan and patch every verification script, seed script, handler, docs entry, and integration surface affected by that contract change before asking the user to run verify-all.

Do not let verification become whack-a-mole. If one verifier fails due to a repeated contract change, immediately scan for all similar call sites and patch the whole class.

For endpoint payload changes, search all scripts/verify*.ps1 files for that endpoint and update every POST body that needs the new contract.

For new endpoints, always include:

- route wiring
- permission creation in migration
- seed-dev-rbac permissions
- focused verification script
- verify-all entry
- docs/PROJECT_STATE.md update
- docs/AI_CHANGELOG.md update
- ADR under docs/decisions when behavior/design changes

### Verification closeout

Every implementation slice must end with commands for:

- gofmt -w ./cmd ./internal
- go test ./...
- git diff --check
- .\scripts\apply-migration.ps1 -Name <migration_name> -Direction up -EnvFile .env
- .\scripts\restart-local-apis.ps1
- .\scripts\<focused-verifier>.ps1 -EnvFile .env
- .\scripts\verify-all.ps1 -EnvFile .env
- git status --short

Include a residue/surface scan command before declaring the slice complete.

If git diff --check reports trailing blank lines, provide a targeted EOF cleanup block.

### Commit discipline

Do not commit or push unless the user explicitly asks.

When the user asks to commit, provide:

- branch guard
- git diff --check
- go test ./...
- .\scripts\verify-all.ps1 -EnvFile .env
- explicit git add file list
- git diff --cached --check
- git diff --cached --stat
- commit command
- push command only if the user asked to push or implied the feature is ready to push

### Communication style during sprints

Be direct and operational.

If a mistake is found, name the exact failed assumption, fix the whole class of issue, and avoid defensive explanations.

The user is relying on the assistant to orchestrate the sprint. Missing affected files, verification scripts, or contract surfaces is considered a process failure.

### Human-facing PowerShell scan commands

Do not assume `rg` is installed in the human's PowerShell environment.

For commands the human must run, use native PowerShell such as:

- `Get-ChildItem -Recurse -File | Select-String -Pattern '<pattern>'`

Do not install, configure, or modify the human PowerShell environment to add `rg` unless the human explicitly asks.

Agents may still use available search tools internally.

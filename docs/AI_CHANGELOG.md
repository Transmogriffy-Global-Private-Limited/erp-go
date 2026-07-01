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

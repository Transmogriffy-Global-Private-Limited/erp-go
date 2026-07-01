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

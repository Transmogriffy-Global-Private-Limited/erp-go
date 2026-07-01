# GitHub Copilot / AI Instructions

This repository builds a fully online multi-tenant ERP SaaS platform in Go.

Read these files before suggesting major changes:

- AGENTS.md
- docs/PROJECT_STATE.md
- docs/ARCHITECTURE_BASELINE.md
- docs/AI_CHANGELOG.md

Core architecture:

- Control Plane: superadmin, tenants, licensing, plans, module entitlements
- ERP Plane: tenant business modules and users
- Strong tenant isolation everywhere
- Modular monolith first
- Event-heavy internally
- PostgreSQL stores truth
- Object storage stores files

Rules:

- Do not mix superadmin/platform logic into tenant business modules casually.
- Do not let modules write each other's tables.
- Do not trust tenant_id from client input.
- Do not propose full ERP-wide E2EE.
- Do not introduce microservices unless justified.
- Do not skip documentation updates for architecture changes.

Preferred stack:

- Go backend
- PostgreSQL
- Redis
- NATS JetStream
- S3-compatible object storage
- React + TypeScript frontend later

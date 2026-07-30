# AI-assisted changelog

This file records meaningful agent-assisted repository changes. It reports
actual work and verification, not planned future behavior.

## 2026-07-30 — Step 00 planning baseline

Status: Verified

### Why

The prior ERP implementation was intentionally reset. The accepted KISS-first
rebuild direction needed durable repository memory so implementation would not
depend on chat history.

### Changed

- Added repository-specific operating and safety rules.
- Added the repository and documentation entry points.
- Added the living development plan, current project state, and this changelog.
- Added the detailed ERP rebuild and HRMS-provider plan.
- Added the planned house-HRMS integration boundary.
- Added ADRs for the modular monolith, provider authority, stable workforce
  identity, configuration/session security, and OpenAPI-first contracts.

### Compatibility and migration impact

None. This slice creates documentation only. There is no application, API,
database schema, migration, configuration, or deployment to migrate.

### Verification

Passed on 2026-07-30:

- all 14 required documentation surfaces existed;
- only Markdown documentation was added;
- all local Markdown file links resolved;
- sharing-safety and credential-value scans passed;
- whitespace, final-newline, and placeholder-marker checks passed;
- required architecture and implementation-state statements were present;
- `git diff --check` passed;
- final `git status --short` showed only the untracked Step 00 documentation.

### Remaining work

- Obtain separate approval before Step 01 implementation.

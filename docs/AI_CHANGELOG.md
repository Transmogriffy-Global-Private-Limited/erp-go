# AI-assisted changelog

This file records meaningful agent-assisted repository changes. It reports
actual work and verification, not planned future behavior.

## 2026-07-30 — Step 01A bootable documented API

Status: Verified

### Why

Step 00 was verified. The human approved the smallest first implementation
slice: a real loopback API process with lifecycle, canonical errors,
authoritative OpenAPI, environment-controlled embedded Swagger UI, and one full
verification entry point.

### Approved scope

- Go module and one `erp-api` process;
- typed loopback configuration;
- `/healthz` and canonical JSON not-found behavior;
- graceful shutdown and explicit HTTP timeouts;
- specification-first OpenAPI;
- `/docs` and `/openapi.yaml` behind `API_DOCS_ENABLED`;
- embedded Swagger assets with no runtime CDN or Node requirement;
- focused tests and `scripts/verify-all.ps1`.

### Excluded

- PostgreSQL, readiness, migrations, identity, tenancy, RBAC, HRMS providers,
  deployment, and all later slices.

### Changed

- Added the Go module, locked dependencies, and one `erp-api` entry point.
- Added typed loopback-only configuration with actionable startup errors.
- Added an HTTP server with explicit timeouts and graceful shutdown.
- Added `GET /healthz`, canonical JSON errors, and method handling.
- Added the specification-first embedded OpenAPI contract.
- Added environment-controlled raw OpenAPI and embedded Swagger UI routes.
- Added focused Go tests, the full PowerShell verifier, local-development
  guidance, and the human-readable HTTP contract.

### Compatibility and migration impact

This is the first application foundation after the reset. It creates no
database, migration, external connection, identity, tenant state, or deployment
artifact. API documentation defaults to disabled and requires a restart after
the toggle changes.

### Verification

Passed on 2026-07-30:

- Go formatting inspection;
- `go test ./...`;
- `go vet ./...`;
- `go build -o .local/bin/erp-api.exe ./cmd/erp-api`;
- invalid wildcard-host startup rejection;
- built-binary loopback smoke tests with documentation disabled and enabled;
- exact embedded raw-contract and local Swagger-asset checks;
- OpenAPI parsing, validation, and route inventory checks;
- `scripts/verify-all.ps1` end to end;
- local Markdown links, sharing-safety language, credential values,
  placeholder markers, stale Step 01A claims, whitespace, and final newlines;
- `git diff --check`.

Nothing was committed, pushed, deployed, or applied to a database for Step 01A.

### Remaining work

- Step 01B and later implementation remain unapproved.

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

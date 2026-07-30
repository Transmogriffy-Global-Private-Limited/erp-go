# ERP project state

Last verified: 2026-07-30

## Current repository reality

The repository has completed Step 00 and locally verified the approved Step 01A
implementation slice.

The application was intentionally reset before this rebuild. The reset baseline
is commit `2a5f2f3bd9bfaa8d54e3b519909c347c81698dc2` on `anubhab-work`.

Step 00 was later committed and pushed only to `anubhab-work`. Step 01A is
currently a verified local worktree change and has not been committed or
pushed.

## Implemented behavior

Step 01A implements:

- a Go 1.25 module and one `cmd/erp-api` entry point;
- typed `HTTP_HOST`, `HTTP_PORT`, and `API_DOCS_ENABLED` configuration;
- a `127.0.0.1:8080` default and rejection of wildcard/non-loopback hosts;
- explicit HTTP timeouts and graceful process shutdown;
- `GET /healthz` as process liveness only;
- canonical JSON `not_found` and `method_not_allowed` errors;
- the authoritative embedded OpenAPI contract at
  `api/openapi/v1/openapi.yaml`;
- `/openapi.yaml`, `/docs`, `/docs/`, and locally embedded Swagger assets only
  when `API_DOCS_ENABLED=true`;
- Go tests for configuration, lifecycle, routing, toggle behavior, exact raw
  schema serving, Swagger assets, OpenAPI validation, and route inventory;
- `scripts/verify-all.ps1` as the canonical native Windows verification entry
  point;
- local development and HTTP contract documentation.

Step 01A intentionally implements no PostgreSQL connection, `/readyz`, schema,
migration, ERP identity, tenant, authentication, authorization, HRMS provider,
external integration, or deployment behavior.

## Documented direction

The repository now records the planned:

- KISS-first Go modular monolith;
- single initial `erp-api` process;
- PostgreSQL source-of-truth boundaries;
- ERP identity and trusted tenant context;
- minimal RBAC and one-provider-per-tenant rule;
- canonical HRMS API boundary;
- native and house-HRMS providers;
- stable ERP `workforce_id`;
- encrypted server-side external sessions;
- OpenAPI-first REST contract and environment-controlled documentation routes;
- loopback-only, non-admin native Windows workflow;
- separately approved vertical implementation sequence.

Documented direction is not implemented behavior.

## Existing external system observation

A local checkout of the standalone house HRMS exists conventionally at
`..\HRMS-2.0`.

Observed during read-only inspection on 2026-07-30:

- Node.js, Fastify, Prisma, and PostgreSQL;
- separate Employee, HR, and Admin authentication paths;
- employee login uses `assignedEmail` and password;
- employee 2FA is optional;
- employee sessions are Bearer credentials with an effective seven-day
  lifetime and database-backed active-session state;
- the employee login method can return a 2FA-required result while the route
  still assumes a token exists;
- employee 2FA completion returns the eventual Bearer credential through the
  response authorization header.

These observations guide the planned adapter but do not prove a stable external
contract. The adapter implementation must revalidate actual behavior and fail
closed. No house-HRMS file, database, credential, or deployment was modified.

See [`integrations/HOUSE_HRMS.md`](integrations/HOUSE_HRMS.md).

## Operational state

- No local ERP service remains running after verification.
- The default listen address is `127.0.0.1:8080`; non-loopback hosts fail
  before listening.
- No ERP database has been created or modified by Step 00 or Step 01A.
- Go module dependencies are locked by `go.sum`; no global tool installation
  is required.
- No remote service or deployment has been changed.

## Verification state

Step 00 documentation verification passed on 2026-07-30:

- all 14 required documentation files existed;
- the Step 00 worktree contained Markdown documentation only;
- all local Markdown file links resolved;
- the sharing-safety scan found no contributor or work-style characterization;
- the credential-value scan found no credential-like values;
- whitespace, final-newline, and placeholder-marker checks passed;
- required architecture and implementation-state statements were present;
- `git diff --check` passed.

Step 01A verification passed on 2026-07-30:

- `gofmt` inspection found no unformatted Go files;
- `go test ./...` passed for configuration, app lifecycle, and HTTP behavior;
- `go vet ./...` passed;
- `go build -o .local/bin/erp-api.exe ./cmd/erp-api` passed;
- invalid wildcard-host startup failed with an actionable configuration error;
- the built binary passed loopback smoke tests with API documentation disabled;
- the built binary passed loopback raw-schema, Swagger UI, and embedded-asset
  smoke tests with API documentation enabled;
- OpenAPI 3.0 parsing, structural validation, and current route inventory
  checks passed;
- `scripts/verify-all.ps1` passed end to end.
- all local Markdown file links resolved;
- source/document final-newline and trailing-whitespace checks passed;
- sharing-safety, credential-value, placeholder-marker, and stale-claim scans
  passed;
- `git diff --check` passed.

## Known limitations and unresolved decisions

The living list is owned by
[`DEVELOPMENT_PLAN.md`](DEVELOPMENT_PLAN.md#risks-and-unresolved-decisions).

The most immediate decisions belong to later slices and include tenant
discovery, administrator bootstrap semantics, external provider naming,
canonical error details, migration tooling, and external-session key rotation.

## Next gate

No implementation is currently approved. Step 01B and all later work require a
new explicit approval.

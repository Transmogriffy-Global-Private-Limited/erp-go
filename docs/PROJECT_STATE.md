# ERP project state

Last verified: 2026-07-30

## Current repository reality

The repository has completed Step 00, a documentation-only planning slice.

The application was intentionally reset before this rebuild. The reset baseline
is commit `2a5f2f3bd9bfaa8d54e3b519909c347c81698dc2` on `anubhab-work`.

Step 00 added uncommitted repository-memory documents on top of that baseline.
No commit or push was part of Step 00.

## Implemented behavior

None.

Specifically, the rebuild currently has no:

- Go module;
- executable or running API;
- route registration;
- OpenAPI document;
- Swagger UI;
- configuration loader;
- PostgreSQL schema or migration;
- ERP identity or tenant record;
- authentication or authorization behavior;
- native HRMS provider;
- house-HRMS adapter;
- local setup or full-verification script;
- deployment artifact.

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

- No local ERP service is running from this rebuild.
- No port is assigned by implemented configuration.
- No ERP database has been created or modified by Step 00.
- No package or tool installation has occurred.
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

There is no application behavior available to compile, test, smoke-test, or
exercise end to end.

## Known limitations and unresolved decisions

The living list is owned by
[`DEVELOPMENT_PLAN.md`](DEVELOPMENT_PLAN.md#risks-and-unresolved-decisions).

The most immediate decisions belong to later slices and include tenant
discovery, administrator bootstrap semantics, external provider naming,
canonical error details, migration tooling, and external-session key rotation.

## Next gate

Step 00 is complete. Step 01 remains proposed and requires explicit human
approval before any application implementation begins.

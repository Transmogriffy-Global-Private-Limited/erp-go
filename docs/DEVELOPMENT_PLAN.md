# ERP development plan

Status: Active living plan

Last reconciled: 2026-07-30

## Project objective

Build a complete native ERP that is easy to understand, develop, integrate,
host, diagnose, recover, extend, and replace without weakening correctness,
security, durability, contracts, or verification.

The ERP includes a native HRMS. The first house deployment already has a
standalone HRMS, so that tenant will use a narrow external provider adapter
without replacing the standalone system's authentication or source-of-truth
responsibility.

## System boundaries

In scope:

- the ERP application and its native modules;
- ERP identity, tenancy, sessions, and RBAC;
- a stable ERP workforce identity;
- the native HRMS provider;
- one narrow adapter to the existing house HRMS;
- canonical REST contracts, OpenAPI, Swagger, verification, and documentation.

Out of scope unless separately proposed and approved:

- a generic integration hub;
- Shuttle, Food, GSH, amenity tracking, or unrelated standalone systems;
- arbitrary runtime plugins or mapping languages;
- microservices and speculative infrastructure;
- changing the standalone house HRMS implementation;
- general external-HRMS data synchronization in the first adapter slice.

## Approved architectural direction

- One Go modular monolith and one `erp-api` process initially.
- PostgreSQL owns durable ERP state.
- Plain Go, explicit SQL through `pgx`, typed configuration, and `log/slog` are
  preferred defaults.
- OpenAPI will be the authoritative REST contract.
- Every tenant has exactly one authoritative HRMS provider.
- Other ERP modules use a stable ERP `workforce_id`.
- External provider sessions remain server-side and encrypted.
- Local development is native PowerShell, non-admin, and loopback-only.
- Implementation proceeds through small, separately approved vertical slices.

The detailed plan is
[`plans/ERP_REBUILD_AND_HRMS_PROVIDER_PLAN.md`](plans/ERP_REBUILD_AND_HRMS_PROVIDER_PLAN.md).

## Permanent engineering invariants

- KISS means the simplest complete design, not omitted guarantees.
- Backend behavior is explicit and is the authority for domain rules,
  authorization, tenant scope, provider selection, and transitions.
- Public contracts remain direct and easy to consume even when internals contain
  necessary advanced logic.
- The safe, correct development path must also be the easiest path.
- One module owns each durable concept and its invariants.
- No dual writes or competing authorities.
- No fake capability or placeholder success.
- No provider-specific identity leakage outside the HRMS boundary.
- No client-selected trusted tenant scope.
- No plaintext external credentials or sessions in storage, logs, or responses.
- Documentation, contracts, verification, and implementation move together.
- No implementation, Git publication, deployment, or database mutation follows
  merely from appearing in this plan; the human must authorize the action.

## Development phases

### Phase 0 — Durable project direction

Objective: persist the accepted plan, boundaries, decisions, status, and next
approval gate without creating application behavior.

Completion enables safe implementation without relying on chat history.

### Phase 1 — Bootable and verifiable foundation

Objective: create the smallest production-shaped Go/PostgreSQL API foundation,
including truthful health/readiness, configuration validation, graceful
shutdown, canonical errors, OpenAPI, and documentation routes.

### Phase 2 — Trusted ERP identity and tenancy

Objective: establish ERP-owned tenants, users, password authentication, opaque
server-side sessions, trusted tenant context, isolation, audit, and first-admin
bootstrap/recovery.

### Phase 3 — Minimal authorization and provider binding

Objective: implement only the roles, permissions, module entitlement, and
tenant HRMS-provider binding needed by current behavior.

### Phase 4 — Canonical HRMS boundary

Objective: expose the first provider-neutral HRMS capabilities, connection
state, and current-user profile contracts without fake native behavior.

### Phase 5 — House employee vertical slice

Objective: link an ERP user to the employee persona of the house HRMS, including
optional 2FA, encrypted sessions, stable workforce mapping, restart recovery,
profile retrieval, disconnect, expiry, and reconnect behavior.

### Phase 6 — Additional house personas

Objective: inspect and implement the HR and Admin personas one workflow at a
time without assuming that employee authentication semantics apply.

### Phase 7 — Native HRMS employee foundation

Objective: implement the first genuine ERP-owned employee record and canonical
profile so native and house-provider tenants use the same ERP contract.

### Phase 8 — HRMS workflows by business priority

Objective: add complete native workflows and truthful external mappings in an
approved order, tentatively employee administration, leave, approvals,
attendance, payroll prerequisites, and payroll.

## Feature registry

### Feature: Step 00 repository memory

Status: Verified

Phase: 0

Objective:

Persist the approved architecture and planning context in authoritative,
navigable repository documents.

Scope:

- repository-specific operating rules;
- documentation index;
- living plan, project state, and changelog;
- detailed rebuild/provider plan;
- house-HRMS integration boundary;
- accepted architecture decisions.

Non-goals:

- application code;
- OpenAPI content for nonexistent routes;
- database schema or migrations;
- dependencies, scripts, deployment, or runtime configuration.

Acceptance criteria:

- a future contributor can recover the approved scope without chat history;
- all documents distinguish planned from implemented behavior;
- links and statuses are internally consistent;
- the worktree contains documentation only;
- no secret or real credential is recorded;
- documentation and Git hygiene checks pass.

Verification:

- inspect every new documentation surface;
- verify local Markdown links;
- scan for prohibited implementation claims and sensitive-value patterns;
- run `git diff --check`;
- inspect `git diff --stat` and `git status --short`.

### Feature: Step 01 minimal bootable ERP foundation

Status: Proposed

Phase: 1

Depends on:

- verified Step 00;
- explicit human approval for Step 01.

Objective:

Create one loopback-safe Go API with typed configuration, PostgreSQL
connectivity, health/readiness, graceful lifecycle, canonical JSON errors, an
authoritative OpenAPI document, and environment-controlled Swagger/raw schema.

Acceptance criteria and detailed affected surfaces are defined in the detailed
rebuild plan. No implementation is currently authorized.

### Feature: Step 02 ERP identity and tenancy

Status: Proposed

Phase: 2

Depends on:

- verified Step 01;
- decisions about tenant discovery and administrator bootstrap semantics;
- explicit approval.

Objective:

Provide ERP login and sessions that remain available independently of HRMS
providers while enforcing trusted tenant isolation.

### Feature: Step 03 minimal RBAC and HRMS provider binding

Status: Proposed

Phase: 3

Depends on:

- verified identity and tenancy;
- explicit approval.

Objective:

Authorize implemented behavior and bind each tenant to one authoritative HRMS
provider without building speculative permissions.

### Feature: Step 04 canonical HRMS provider boundary

Status: Proposed

Phase: 4

Depends on:

- verified RBAC/provider binding;
- final initial capability and error contracts;
- explicit approval.

Objective:

Expose provider-neutral capabilities, connection state, and current-user
profile contracts with shared provider tests.

### Feature: Step 05 house employee HRMS vertical slice

Status: Proposed

Phase: 5

Depends on:

- verified canonical HRMS boundary;
- confirmed house-HRMS employee contracts;
- selected encryption/key-rotation design;
- explicit approval.

Objective:

Implement the complete employee-persona connection lifecycle through a narrow,
fail-closed adapter.

### Feature: Step 06 house HR and Admin personas

Status: Proposed

Phase: 6

Depends on:

- verified employee adapter slice;
- separate inspection of each persona;
- workflow-level approval.

Objective:

Add only the externally supported HR and Admin operations required by the ERP.

### Feature: Step 07 native HRMS employee foundation

Status: Proposed

Phase: 7

Depends on:

- verified canonical HRMS boundary and stable workforce identity;
- approved native employee lifecycle;
- explicit approval.

Objective:

Provide a genuine ERP-owned employee record and profile under the same
canonical contract used by the house adapter.

### Feature: Step 08 and later HRMS workflows

Status: Proposed

Phase: 8

Objective:

Add HRMS workflows one complete vertical slice at a time according to actual
business priority.

The tentative order is not implementation authorization.

## Current execution

Current phase:

- Phase 0 — Durable project direction

Active feature:

- None

Current implementation slice:

- None. Step 00 is complete and no application slice is authorized.

Last completed slice:

- Step 00 repository memory, verified on 2026-07-30.

Next expected slice:

- Step 01 minimal bootable ERP foundation, only after explicit human approval.

Blocked by:

- No technical blocker recorded.
- All implementation is gated by a new explicit approval.

## Next approved work

None. Step 00 is verified. Step 01 remains proposed and no application
implementation is currently approved.

## Risks and unresolved decisions

Resolve before the dependent implementation slice:

- the stable external provider kind name (`house_hrms` versus a versioned name);
- tenant discovery during ERP login;
- whether first-admin bootstrap refers to tenant administration, platform
  administration, or two distinct workflows;
- canonical error codes and error envelope details;
- the initial HRMS capability representation and connection command paths;
- the migration tool and migration-verification mechanism;
- external-session encryption envelope, key identification, rotation, and
  invalidation behavior;
- provider-switch migration and reconciliation rules;
- audit retention and sensitive metadata policy;
- the final implemented defaults and routes for health, readiness, Swagger, and
  raw OpenAPI serving.

These are recorded decisions to make, not permission to improvise during an
unrelated slice.

## Verification strategy

Every implemented feature must use the smallest relevant checks first, then the
broadest available repository verification. The intended mature sequence is:

1. formatter;
2. focused unit tests;
3. package tests;
4. `go test ./...`;
5. build;
6. migration validation where applicable;
7. focused loopback integration verification;
8. OpenAPI validation and route/contract drift checking;
9. Swagger and raw-schema enabled/disabled checks;
10. residue and affected-surface scan;
11. `scripts/verify-all.ps1` after Step 01 creates it;
12. `git diff --check`;
13. `git status --short`.

Do not claim a script or check exists until `PROJECT_STATE.md` records it.

## Project completion criteria

The project is not complete until its approved ERP modules work coherently
across authentication, authorization, tenant scope, durable state, external
integration, failure, recovery, contracts, frontend consumption, operations,
verification, and documentation.

Individual features become Verified only when their stated focused and full
verification passes. Writing code or documentation alone is not sufficient.

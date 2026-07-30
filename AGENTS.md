# AGENTS.md

## Purpose and scope

These rules apply to the complete `erp-go` repository. They supplement the
workspace-level agent instructions and may not weaken their safety, Git,
documentation, or verification requirements.

The repository is a KISS-first rebuild of a native ERP. Its minimal Step 01A
HTTP foundation is implemented and verified. Do not infer that a database
schema, business module, migration, or integration exists unless
`docs/PROJECT_STATE.md` states that it is implemented and identifies its
verification evidence.

## Required project memory

Before planning or changing this repository, read:

- `README.md`
- `docs/README.md`
- `docs/DEVELOPMENT_PLAN.md`
- `docs/PROJECT_STATE.md`
- `docs/AI_CHANGELOG.md`
- the relevant detailed plan under `docs/plans/`
- the relevant ADRs under `docs/decisions/`
- the relevant integration guide under `docs/integrations/`

Approved plans are binding implementation guidance. Do not silently redesign,
bypass, or materially expand them. Record discoveries that invalidate a plan
and obtain human approval before a material architecture or scope change.

## Current implementation gate

Step 00 and Step 01A are complete. No later implementation slice is currently
approved.

Step 01A does not authorize PostgreSQL, readiness, migrations, identity,
tenancy, RBAC, HRMS providers, deployment, or any later slice. Do not expand
application code, dependencies, runtime configuration, or operational assets
beyond the active approved slice.

## Product boundary

This repository will become a complete native ERP, including a native HRMS.
It will also support a narrow adapter to an existing house HRMS for the tenant
that already uses that system.

It is not a generic integration hub. Do not bring Shuttle, Food, GSH, amenity
tracking, or unrelated standalone systems into the ERP integration design.

## Permanent architecture invariants

- Begin with one Go modular monolith and one deployable `erp-api` process.
- PostgreSQL is the durable source of truth for ERP-owned state.
- Keep one authoritative owner for every business concept.
- Preserve module ownership even when modules share one process and database.
- Use plain Go, explicit control flow, explicit dependency construction, and
  explicit transaction boundaries.
- Do not add microservices, Redis, NATS, Kafka, gRPC, WebSockets, object
  storage, background workers, containers, or a dependency-injection framework
  without a current, approved requirement.
- Do not add speculative interfaces, repositories, plugins, mapping languages,
  generic proxies, or extension frameworks.
- An interface is justified by an actual boundary or a real second
  implementation. The native/external HRMS provider boundary is such a case.
- Keep advanced internal logic isolated. Ordinary module development and every
  frontend-facing contract must remain clear without requiring knowledge of
  unrelated backend internals.
- Backend code owns domain rules, authorization, tenant scope, provider
  selection, and state-transition validity. Frontends must not reconstruct
  these rules.
- Prefer explicit task-oriented commands for meaningful business actions.
- Make the correct path the easiest path through safe defaults, focused tests,
  actionable errors, examples, and one repeatable verification entry point.

## HRMS invariants

- Each tenant selects exactly one authoritative HRMS provider.
- The planned provider implementations are the native ERP HRMS and one narrow
  house-HRMS adapter.
- Never make both providers writable authorities for the same tenant.
- Never implement dual writes.
- Provider changes require an explicit migration and reconciliation workflow;
  they are not ordinary live configuration toggles.
- The ERP API and all other ERP modules consume canonical ERP-owned HRMS
  contracts. Provider-specific payloads must remain inside the adapter.
- Unsupported capabilities must be reported honestly. Never fake success or
  invent unavailable data.
- Other ERP modules refer to people through the stable ERP `workforce_id`, not
  raw provider identifiers.

## Identity, tenant, and security invariants

- ERP authentication is independent of HRMS authentication and availability.
- Derive tenant context from trusted server-side authentication or routing.
- Never trust an unvalidated client-supplied tenant identifier.
- Deny access by default and enforce authorization server-side.
- Platform administration must not silently bypass tenant business-data
  boundaries.
- Never persist or log external passwords or OTPs.
- Never return an external provider token or session to the browser.
- Encrypt external provider session material before PostgreSQL persistence.
- Secrets and encryption keys come from environment configuration or an
  established secret mechanism, never committed files.
- Keep external connections scoped by tenant, ERP user, provider, and persona
  where the provider distinguishes personas.
- Use explicit outbound timeouts, bounded response sizes, trusted host
  allowlists, TLS for remote providers, and secret-redacted errors.
- A house-HRMS outage must degrade the HRMS module, not ERP authentication or
  unrelated ERP modules.

## Required documentation surfaces

Canonical documentation index:

- `docs/README.md`

Living development plan:

- `docs/DEVELOPMENT_PLAN.md`

Present implementation state:

- `docs/PROJECT_STATE.md`

Agent-assisted changelog:

- `docs/AI_CHANGELOG.md`

Detailed plans:

- `docs/plans/`

Architecture decisions:

- `docs/decisions/`

Integration documentation:

- `docs/integrations/`

Educational guides:

- `docs/guides/`

Human-readable programmatic contracts:

- `docs/contracts/`

Authoritative REST contract:

- `api/openapi/v1/openapi.yaml`

The embedded application document, raw schema route, contract validation, and
Swagger UI use this same source. Do not manually maintain a second API schema.
Contract, implementation, examples, tests, and frontend guidance must change in
the same slice.

The local documentation routes are `/docs`, `/docs/`, and `/openapi.yaml`,
controlled by `API_DOCS_ENABLED`. The default is `false`; the enabled and
disabled states must remain covered by the full verifier.

Canonical verification command:

- `scripts/verify-all.ps1`

## Development workflow

For each meaningful slice:

1. Reconfirm the approved scope and current Git state.
2. Build an affected-surface map and acceptance criteria.
3. Implement the smallest complete vertical behavior.
4. Add focused verification and full-suite registration where applicable.
5. Update OpenAPI and human-readable contracts in the same slice.
6. Update the development plan, project state, changelog, detailed plan, and
   ADRs whose meaning changed.
7. Run formatting, focused checks, broader checks, a residue scan,
   `git diff --check`, and `git status --short`.
8. Report precisely what was and was not verified.

Do not begin a later slice while repository memory describes an outdated state.

## Developer usability rules

- A normal feature should have one obvious owning module and one traceable
  request-to-state-transition flow.
- Keep functions small enough to understand in one sitting. Decompose genuine
  algorithms into named, testable helpers rather than hiding complexity.
- Avoid clever Go, reflection, deep interface chains, and generic frameworks.
- Provide one established example for every repeated feature pattern.
- Use canonical typed errors and direct, copyable API examples.
- Return frontend-useful state such as truthful capabilities and allowed
  actions, while keeping the backend authoritative for authorization.
- A bounded feature must be implementable without understanding
  external-session encryption, every ERP module, or the whole provider
  architecture.
- Critical authentication, tenant isolation, authorization, encryption,
  transaction, and migration logic requires focused tests and close review.

## Local environment

- Primary environment: Windows 11 and PowerShell 7+.
- Development must work without administrator access.
- Prefer repository-local or user-scoped tooling.
- Native development is the default; do not require Docker or WSL.
- Local listeners must use `127.0.0.1`, `localhost`, or `::1` only.
- Never use `0.0.0.0` or `::` for local development unless the human explicitly
  changes the environment constraint.
- Human-facing commands must be directly pasteable in PowerShell.

## External house HRMS

The local house-HRMS checkout is conventionally a sibling repository at
`..\HRMS-2.0`. Treat it as a separate system and source of truth.

Do not modify that repository, its authentication, its data, or its deployment
without explicit authorization for work in that repository. Inspect its actual
schemas, payloads, query paths, and failure behavior before implementing an
adapter operation; do not infer behavior from filenames.

## Git, database, and deployment safety

- Do not stage, commit, push, merge, rebase, deploy, or modify a remote system
  without explicit permission for that action.
- Do not treat approval for one slice or one earlier commit as continuing
  authorization.
- Preserve unrelated and uncommitted human work.
- Do not execute destructive database operations.
- Never place real credentials, tokens, personal data, or secrets in code,
  examples, tests, logs, screenshots, or documentation.

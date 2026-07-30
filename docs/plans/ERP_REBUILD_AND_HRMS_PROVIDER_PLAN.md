# ERP rebuild and HRMS provider plan

Status: Approved architecture; implementation not authorized by this document

Last updated: 2026-07-30

## 1. Objective

Build a complete native ERP with a native HRMS while allowing a tenant that
already uses the standalone house HRMS to keep that system as its HRMS source
of truth.

The result must be simple to develop, understand, use, integrate, host,
diagnose, recover, extend, and replace. Simplicity may not remove necessary
security, correctness, durability, recovery, contracts, or verification.

## 2. Product model

1. The ERP is a native product, not an integration shell.
2. Customers without an HRMS use the ERP's native HRMS.
3. The house tenant uses the existing standalone HRMS through a narrow adapter.
4. Each tenant has exactly one authoritative HRMS provider.
5. Additional ERP-specific adapters are considered only for real requirements.
6. Provider changes are explicit migration and reconciliation operations.

The adaptability boundary currently applies to HRMS only. It does not include
Shuttle, Food, GSH, amenity tracking, or unrelated standalone systems.

## 3. Initial runtime architecture

```text
Client
  -> one erp-api process
     -> HTTP validation and canonical errors
     -> ERP identity and trusted tenant context
     -> authorization
     -> owning application module
        -> PostgreSQL transaction for ERP-owned state
        -> or selected HRMS provider for provider-owned behavior
     -> canonical ERP response
     -> audit record where required
```

Initial technology direction:

- Go;
- one modular monolith;
- one deployable API process;
- PostgreSQL;
- `net/http` with a small router such as `chi` when route grouping adds value;
- `pgx` and explicit SQL;
- `log/slog`;
- typed startup configuration;
- specification-first OpenAPI;
- Swagger UI using that same OpenAPI document;
- PowerShell setup and verification scripts;
- loopback-only native local development.

The first implementation does not add Redis, NATS, Kafka, object storage,
gRPC, WebSockets, background workers, Kubernetes, containers, a dependency
injection framework, or separate services per module.

## 4. Developer usability

Necessary complexity must be isolated behind cohesive ownership boundaries.

A normal feature should expose one visible flow:

```text
handler
-> typed request
-> application command or query
-> authorization
-> transaction or provider operation
-> typed result
```

Rules:

- use direct control flow and ordinary Go constructs;
- avoid reflection, hidden hooks, speculative generics, and deep interfaces;
- keep one obvious module owner for each workflow;
- make transaction boundaries visible;
- provide one established example for repeated patterns;
- make errors actionable and contracts copyable;
- keep frontend rules declarative and backend-authoritative;
- isolate provider encryption and translation from unrelated ERP modules;
- use focused tests as executable examples of important behavior.

## 5. Conceptual module ownership

These are ownership boundaries, not an approved directory scaffold.

| Module | Owns |
|---|---|
| Runtime | startup, shutdown, health, readiness, configuration wiring |
| Identity | ERP users, password verification, sessions, login/logout |
| Tenancy | tenants and trusted tenant context |
| Authorization | roles, permissions, assignments, enforcement |
| HRMS | canonical contracts, capabilities, provider selection |
| Native HRMS | ERP-owned employee and later HRMS workflow state |
| House HRMS adapter | provider transport, translation, external sessions |
| Workforce | stable `workforce_id` and provider identity mappings |
| Audit | security-sensitive and privileged action records |

One module must not casually write another module's internal tables merely
because the initial system shares one PostgreSQL database.

## 6. Source-of-truth map

| Concept | Authority |
|---|---|
| ERP tenant | ERP PostgreSQL |
| ERP user and session | ERP PostgreSQL |
| ERP role and permission | ERP PostgreSQL |
| Enabled ERP module | ERP PostgreSQL |
| Tenant HRMS-provider binding | ERP PostgreSQL |
| Stable `workforce_id` | ERP PostgreSQL |
| Provider-person mapping | ERP PostgreSQL |
| Encrypted external connection state | ERP PostgreSQL |
| Native HRMS records | ERP PostgreSQL |
| House HRMS records and external roles | Standalone house HRMS |
| Provider implementation registration and safe network facts | Deployment configuration |
| Encryption keys and other secrets | Environment or established secret mechanism |

## 7. Canonical HRMS boundary

The frontend and all other ERP modules consume ERP-owned request, response,
error, capability, and identity contracts.

The first query surface is expected to include:

- `GET /api/v1/hrms/capabilities`;
- `GET /api/v1/hrms/connection`;
- `GET /api/v1/hrms/me`.

External account linking, 2FA completion, disconnect, and reconnect commands
are also required, but their exact paths and schemas must be approved with the
Step 04/05 contract rather than invented here.

Provider-specific payloads, paths, tokens, and error text must not escape the
adapter.

Provider selection occurs once through a tenant-scoped provider resolver. Do
not spread provider-name conditions through handlers, frontends, or other ERP
modules.

Capabilities describe real provider behavior. A provider that cannot perform
an operation reports it as unsupported. A native provider with no implemented
employee profile must not return placeholder success.

## 8. Identity model

ERP authentication is independent from provider authentication.

For a native-HRMS tenant:

```text
ERP login
-> trusted tenant context
-> ERP authorization
-> native HRMS data
```

For a house-HRMS tenant:

```text
ERP login
-> trusted tenant context
-> ERP authorization
-> external account linking for a specific persona
-> encrypted server-side provider session
-> stable workforce mapping
-> canonical HRMS response
```

Other ERP modules store `workforce_id`. They never adopt raw provider IDs as
their domain identity.

The minimum external mapping concept includes:

- tenant;
- ERP user;
- stable `workforce_id`;
- provider;
- provider subject/person identifier;
- persona where relevant;
- connection status.

The first external adapter does not copy the complete house-HRMS database into
ERP PostgreSQL.

## 9. External authentication and session boundary

- Use external passwords and OTPs only for the immediate provider request.
- Never store or log passwords or OTPs.
- Encrypt the resulting provider session before durable storage.
- Never return provider session material to the browser.
- Scope external connections by tenant, ERP user, provider, and persona.
- Make expiry and reconnect requirements explicit.
- Rate-limit external authentication and OTP operations.
- Redact provider secrets from errors, logs, and audit metadata.
- Restore valid encrypted connection state after ERP restart.
- Never elevate an external persona based only on ERP role.

The encryption envelope, key identifier, rotation, and invalidation strategy
must be decided before Step 05 implementation.

## 10. Provider failure and recovery

If the house HRMS is unavailable:

- ERP login and unrelated ERP modules continue to work;
- the HRMS operation returns a canonical provider-unavailable result;
- no success is reported when the provider outcome is unknown;
- expired sessions result in a clear reconnect-required state;
- tenant and user isolation remain enforced.

Provider clients require:

- explicit connect and request timeouts;
- bounded response bodies;
- trusted configured host allowlisting;
- TLS for remote connections;
- structured, redacted errors;
- validation of every provider response before state changes.

Future external writes require workflow-specific idempotency, retry, duplicate,
partial-failure, and reconciliation rules. Blind retry of a non-idempotent write
is prohibited.

## 11. Configuration ownership

Deployment configuration may register:

- provider implementation name and kind;
- provider base URL;
- allowed host;
- timeouts;
- required TLS behavior.

ERP PostgreSQL owns tenant selection, mappings, connection state, and audit
records. Environment configuration owns encryption keys and secrets.

Configuration must not become a scripting or mapping language and must never
contain user passwords, OTPs, or provider sessions. Invalid startup
configuration must fail before the server begins accepting traffic.

## 12. HTTP contract and documentation

The planned authoritative source is:

```text
api/openapi/v1/openapi.yaml
```

It must cover every implemented HTTP endpoint, authentication scheme,
authorization expectation, trusted scope, request, response, validation,
canonical error, example, side effect, and relevant idempotency behavior.

Swagger UI and raw OpenAPI serving use that exact document. Step 01 will
implement and verify an `API_DOCS_ENABLED` toggle. When disabled, both routes
must be unavailable rather than merely hidden from navigation.

The planned local route names are `/docs` and `/openapi.yaml`; Step 01 owns the
final implementation and verification record.

## 13. Implementation sequence

### Step 00 — Repository memory

Create and verify the plan, decisions, project state, changelog, navigation, and
integration boundary. Do not create application behavior.

### Step 01 — Minimal bootable foundation

Scope:

- Go module and one API entry point;
- typed configuration and fail-fast validation;
- loopback-safe listener;
- PostgreSQL connectivity;
- health and readiness;
- graceful shutdown;
- canonical JSON errors;
- authoritative OpenAPI;
- Swagger and raw schema behind `API_DOCS_ENABLED`;
- idempotent PowerShell setup and full verification entry point.

Acceptance:

- invalid configuration prevents serving;
- liveness does not falsely depend on optional downstream systems;
- readiness reflects required PostgreSQL availability;
- shutdown closes listeners and resources cleanly;
- documentation routes work when enabled and are unavailable when disabled;
- route/schema drift checking passes;
- ordinary endpoints continue in both documentation modes;
- local verification uses loopback and no administrator privileges.

### Step 02 — ERP identity and tenancy

Scope:

- tenants and ERP users;
- password login;
- opaque server-side sessions;
- logout, expiry, and revocation;
- trusted tenant derivation and query scope;
- authentication audit;
- first-administrator bootstrap and recovery.

Acceptance:

- ERP authentication works without any HRMS provider;
- client input cannot establish unauthorized tenant context;
- password storage and session behavior fail closed;
- cross-tenant access is denied and verified;
- bootstrap/recovery is explicit, auditable, and safe to disable.

Tenant discovery and administrator semantics require an approved decision before
implementation.

### Step 03 — Minimal RBAC and provider binding

Scope:

- roles, permissions, assignments, and enforcement required by current routes;
- tenant module entitlement;
- one authoritative HRMS provider binding per tenant.

Acceptance:

- denied access returns canonical `403` behavior;
- backend authorization remains authoritative;
- no speculative permission catalog is created;
- a provider change cannot silently create competing authorities.

### Step 04 — Canonical HRMS provider boundary

Scope:

- provider contract and explicit registry;
- tenant-scoped provider resolution;
- capabilities, connection state, and current profile queries;
- shared provider contract tests;
- canonical errors and OpenAPI.

Acceptance:

- callers do not branch on provider-specific payloads;
- unsupported operations are truthful;
- unimplemented native behavior does not return success;
- tenant scope is retained through provider resolution.

### Step 05 — House employee vertical slice

Scope:

```text
ERP login
-> inspect HRMS connection
-> link employee account
-> optional 2FA
-> validate provider response
-> encrypt and persist provider session
-> create or confirm workforce mapping
-> retrieve canonical employee profile
-> restore after restart
-> disconnect
-> reconnect after expiry
```

Acceptance includes wrong password/OTP, expired or consumed challenge, missing
or malformed token, malformed JSON, expired session, timeout, downtime, logout
failure, cross-user denial, cross-tenant denial, and secret-redaction checks.

Deterministic verification uses a loopback mock before any separately approved
live smoke test with nonproduction credentials.

### Step 06 — House HR and Admin personas

Inspect and implement each persona and operation separately. Every operation
requires ERP permission, the matching external persona connection, tenant/user/
provider/persona scope, and explicit auditing.

### Step 07 — Native HRMS employee foundation

Scope:

- native employee record;
- stable workforce identity and ERP-user link;
- employment/lifecycle state;
- native current profile and capabilities;
- authorization and audit.

Acceptance includes one house-provider tenant and one native-provider tenant
using the same canonical ERP HRMS contract.

### Step 08 and later — HRMS workflows

Tentative order, subject to business approval:

1. employee directory and administration;
2. leave;
3. approval workflows;
4. attendance;
5. payroll prerequisites;
6. payroll.

Each workflow must be complete for the native provider. The external adapter
maps only behavior the house HRMS actually supports.

## 14. Verification model

Shared provider contract tests must enforce canonical behavior for native and
external implementations.

The loopback mock house HRMS must cover normal login, 2FA, credential failures,
challenge lifecycle, missing/malformed credentials, expired sessions, malformed
JSON, timeout, downtime, logout failure, restart restoration, redaction, and
isolation.

Every slice must update focused tests, full verification, OpenAPI, examples,
integration documentation, project state, development plan, and changelog as
applicable.

## 15. Non-goals

- generic arbitrary HTTP proxy;
- runtime plugin engine;
- JSONPath or authentication DSL;
- manifest-driven integration framework;
- provider hot loading;
- dual-write synchronization;
- speculative infrastructure;
- changing the standalone HRMS without separate authorization;
- implementing every future ERP module before a real requirement exists.

## 16. Approval gates

- Step 00 approval authorizes documentation only.
- Every later step requires separate explicit implementation approval.
- Commit does not imply push.
- Neither commit, push, deployment, database mutation, nor external-system
  modification is authorized by this plan.

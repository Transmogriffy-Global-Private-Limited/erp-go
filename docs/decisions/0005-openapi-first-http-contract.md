# ADR 0005: OpenAPI-first HTTP contract

Status: Accepted

Date: 2026-07-30

## Context

The ERP frontend and future integrations need an authoritative contract that
does not require reading backend code. A separately handwritten Swagger page or
informal examples would drift from route behavior.

The repository starts empty, so it can establish one contract source before the
first endpoint is implemented.

## Decision

Use a specification-first OpenAPI document as the authoritative REST contract:

```text
api/openapi/v1/openapi.yaml
```

For every implemented HTTP endpoint, the document describes:

- method, path, operation ID, purpose, and grouping;
- authentication and authorization expectations;
- trusted tenant or ownership scope;
- headers, path parameters, and query parameters;
- request and response schemas;
- required fields, nullability, enums, and validation;
- canonical errors and status codes;
- examples;
- side effects and idempotency where applicable;
- compatibility and deprecation semantics.

The same file drives:

- schema validation;
- raw OpenAPI serving;
- Swagger UI;
- route/contract drift detection;
- examples and frontend integration guidance.

Step 01A implemented local routes `/docs`, `/docs/`, and `/openapi.yaml` behind
`API_DOCS_ENABLED`. The default is `false`. When disabled, the UI, embedded
assets, and raw schema routes are unavailable. Focused tests and the full
PowerShell verifier cover both states on loopback.

Do not maintain a second independent schema and do not manually edit generated
artifacts if generation is later introduced.

## Consequences

Positive:

- consumers receive a complete contract before integration;
- review can detect contract changes explicitly;
- Swagger cannot silently describe a different schema;
- route coverage and breaking-change checks have a stable input;
- canonical errors and security schemes remain consistent.

Tradeoffs:

- every endpoint slice must update the specification and implementation
  together;
- response conformance still needs tests because a schema alone cannot prove
  runtime behavior;
- schema review is part of normal feature review.

## Alternatives rejected

### Documentation generated only from handlers

Rejected initially because important authorization, tenancy, failure,
idempotency, and compatibility semantics can be omitted by minimal annotations.

### Separate handwritten Swagger configuration

Rejected because it creates a second source of truth that can drift.

### Documentation after implementation

Rejected because frontend and integration behavior would depend on source-code
archaeology and undocumented assumptions.

## Revisit when

The repository may adopt checked code generation later if it preserves this
single-source contract, improves drift prevention, and does not add unnecessary
tooling complexity.

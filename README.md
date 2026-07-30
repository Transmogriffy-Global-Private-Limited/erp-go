# ERP Go

`erp-go` is a KISS-first rebuild of a native ERP product.

The product will provide its own native HRMS for customers that do not already
have one. A tenant that already uses the house HRMS will access that system
through a narrow ERP-owned adapter and the same canonical ERP HRMS API.

## Current status

The repository has completed **Step 00** and verified **Step 01A: the bootable,
documented API process**.

The implemented foundation provides:

- one Go `erp-api` process with loopback-only typed configuration;
- graceful shutdown and explicit HTTP timeouts;
- `GET /healthz` and canonical JSON errors;
- an authoritative OpenAPI contract;
- embedded Swagger UI and raw schema routes behind `API_DOCS_ENABLED`;
- focused tests and one PowerShell verification entry point.

It does not yet provide PostgreSQL, readiness, migrations, identity, tenancy,
RBAC, HRMS behavior, or a deployment.

## Local verification

From PowerShell at the repository root:

```powershell
.\scripts\verify-all.ps1
```

See [`docs/guides/LOCAL_DEVELOPMENT.md`](docs/guides/LOCAL_DEVELOPMENT.md) for
configuration and local run instructions.

## Architectural direction

- One Go modular monolith.
- One deployable `erp-api` process initially.
- PostgreSQL for durable ERP-owned state.
- Plain, explicit Go and small coherent modules.
- OpenAPI as the authoritative REST contract.
- Native Windows/PowerShell development on loopback.
- No speculative infrastructure.

Each tenant will use exactly one authoritative HRMS provider:

- the native ERP HRMS; or
- the existing house HRMS through a compiled Go adapter.

The frontend and other ERP modules will use canonical ERP contracts and a
stable ERP `workforce_id`. Provider-specific identities and payloads remain
inside the HRMS integration boundary.

## Documentation

Start with [`docs/README.md`](docs/README.md).

Important documents include:

- [`docs/DEVELOPMENT_PLAN.md`](docs/DEVELOPMENT_PLAN.md) — living sequence and
  approval state.
- [`docs/PROJECT_STATE.md`](docs/PROJECT_STATE.md) — what actually exists now.
- [`docs/plans/ERP_REBUILD_AND_HRMS_PROVIDER_PLAN.md`](docs/plans/ERP_REBUILD_AND_HRMS_PROVIDER_PLAN.md)
  — detailed rebuild and HRMS provider plan.
- [`docs/integrations/HOUSE_HRMS.md`](docs/integrations/HOUSE_HRMS.md) — planned
  boundary with the existing standalone HRMS.
- [`docs/decisions/README.md`](docs/decisions/README.md) — accepted architecture
  decisions.

## Development gate

Step 01A is verified. PostgreSQL, readiness, identity, tenancy, RBAC, HRMS, and
later implementation remain unapproved.

Nothing should be committed, pushed, deployed, or applied to a database without
separate explicit permission.

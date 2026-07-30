# ERP Go

`erp-go` is a KISS-first rebuild of a native ERP product.

The product will provide its own native HRMS for customers that do not already
have one. A tenant that already uses the house HRMS will access that system
through a narrow ERP-owned adapter and the same canonical ERP HRMS API.

## Current status

The repository has completed **Step 00: the verified planning and
repository-memory baseline**.

At this point:

- no Go module or application binary exists;
- no API route is implemented;
- no ERP database schema or migration exists;
- no provider adapter is implemented;
- no setup or verification script exists;
- no service has been started or deployed from this rebuild.

The documentation describes approved direction and proposed implementation
slices. It must not be read as evidence that those behaviors already exist.

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

Step 00 documentation is complete. Application implementation starts only after
the human explicitly approves Step 01.

Nothing should be committed, pushed, deployed, or applied to a database without
separate explicit permission.

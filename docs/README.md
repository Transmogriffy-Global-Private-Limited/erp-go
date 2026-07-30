# ERP documentation index

This index is the canonical navigation entry point for the ERP documentation.
Documents distinguish approved plans from implemented and verified behavior.

## Status vocabulary

- **Proposed** — shaped for discussion but not authorized for implementation.
- **Approved** — accepted by the human but not necessarily started.
- **Ready** — approved with sufficiently defined dependencies and acceptance
  criteria.
- **In Progress** — the active implementation or documentation slice.
- **Implemented** — planned changes exist but may not have completed all
  verification.
- **Verified** — all required verification for the slice passed.
- **Deferred** — intentionally postponed.

## Canonical project memory

| Question | Canonical document |
|---|---|
| What should be built and in which order? | [`DEVELOPMENT_PLAN.md`](DEVELOPMENT_PLAN.md) |
| What exists right now? | [`PROJECT_STATE.md`](PROJECT_STATE.md) |
| What did an agent change and verify? | [`AI_CHANGELOG.md`](AI_CHANGELOG.md) |
| What is the detailed rebuild design? | [`plans/ERP_REBUILD_AND_HRMS_PROVIDER_PLAN.md`](plans/ERP_REBUILD_AND_HRMS_PROVIDER_PLAN.md) |
| Why were major architecture choices made? | [`decisions/README.md`](decisions/README.md) |
| How will the house HRMS integrate? | [`integrations/HOUSE_HRMS.md`](integrations/HOUSE_HRMS.md) |
| How do I run and verify the current API locally? | [`guides/LOCAL_DEVELOPMENT.md`](guides/LOCAL_DEVELOPMENT.md) |
| What is the implemented HTTP contract? | [`contracts/HTTP_API.md`](contracts/HTTP_API.md) |

## Architecture decisions

- [`decisions/0001-modular-monolith-and-kiss.md`](decisions/0001-modular-monolith-and-kiss.md)
- [`decisions/0002-native-and-external-hrms-providers.md`](decisions/0002-native-and-external-hrms-providers.md)
- [`decisions/0003-stable-workforce-identity.md`](decisions/0003-stable-workforce-identity.md)
- [`decisions/0004-configuration-and-external-session-security.md`](decisions/0004-configuration-and-external-session-security.md)
- [`decisions/0005-openapi-first-http-contract.md`](decisions/0005-openapi-first-http-contract.md)

## Machine-readable contract

The authoritative REST contract is:

```text
api/openapi/v1/openapi.yaml
```

That document is embedded into the binary and is the single schema used for
validation, raw serving, Swagger UI, drift detection, examples, and future
frontend integration.

## Documentation ownership

- `DEVELOPMENT_PLAN.md` owns sequencing, approval, feature status, risks, and
  next work.
- `PROJECT_STATE.md` owns current implemented and operational reality.
- `AI_CHANGELOG.md` records meaningful agent-assisted changes and actual
  verification.
- `plans/` holds detailed feature and architecture plans linked from the living
  development plan.
- `decisions/` owns durable architectural decisions and their consequences.
- `integrations/` owns cross-system boundaries, contracts, security, failure,
  and verification expectations.
- `guides/` teaches implemented development and operational workflows.
- `contracts/` explains human-readable semantics for implemented programmatic
  boundaries while linking to the authoritative machine schema.

When documents overlap, link to the canonical owner instead of duplicating
facts that can drift.

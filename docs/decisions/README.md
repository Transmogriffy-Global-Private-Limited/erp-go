# Architecture decision index

Architecture Decision Records capture durable choices, their context, and their
consequences. They do not prove that the decided behavior is implemented.

| ADR | Status | Decision |
|---|---|---|
| [0001](0001-modular-monolith-and-kiss.md) | Accepted | Begin with a KISS-first Go modular monolith |
| [0002](0002-native-and-external-hrms-providers.md) | Accepted | Use one canonical HRMS boundary with one authoritative provider per tenant |
| [0003](0003-stable-workforce-identity.md) | Accepted | Give ERP modules a stable provider-independent workforce identity |
| [0004](0004-configuration-and-external-session-security.md) | Accepted | Separate configuration, durable connection state, and secret ownership |
| [0005](0005-openapi-first-http-contract.md) | Accepted | Use one specification-first OpenAPI contract for REST and interactive docs |

When a decision changes materially, create a superseding ADR rather than
rewriting history to imply the earlier decision never existed.

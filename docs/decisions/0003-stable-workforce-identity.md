# ADR 0003: Stable ERP workforce identity

Status: Accepted

Date: 2026-07-30

## Context

ERP modules such as projects, approvals, accounting, and later HRMS workflows
need a durable reference to a person. Native and external HRMS providers use
different identifiers, and provider selection may change only through a future
migration.

If every ERP module stores provider-specific IDs, provider behavior leaks across
the system and future migration becomes a broad data rewrite.

## Decision

ERP PostgreSQL owns a stable internal `workforce_id`.

Other ERP modules reference `workforce_id`, not native-HRMS row IDs, external
employee IDs, emails, session subjects, or provider tokens.

Maintain a minimal provider mapping that includes, as applicable:

- tenant;
- ERP user;
- `workforce_id`;
- provider;
- provider subject/person identifier;
- persona;
- connection state.

Mappings retain tenant scope. External personas may have separate connection
records while still mapping deliberately to the appropriate workforce identity.

## Consequences

Positive:

- non-HRMS modules remain independent of provider identifiers;
- provider-specific identifier formats do not leak into the ERP domain;
- provider migration has one explicit mapping boundary;
- external session rotation does not change workforce identity.

Tradeoffs:

- identity linking and conflict handling must be explicit;
- uniqueness constraints must prevent ambiguous mappings;
- merges, splits, and provider migration require audited operations;
- an ERP user and a workforce identity are related concepts, not necessarily
  interchangeable in every future workflow.

## Alternatives rejected

### Use email as the ERP person identifier

Rejected because email can change, may differ between systems, and does not
provide a durable tenant-scoped identity.

### Use the provider's employee identifier everywhere

Rejected because it couples every module to the active provider and makes
provider replacement unsafe.

### Copy the complete provider employee record

Rejected for the first version because the external HRMS remains authoritative
and general synchronization is not required.

## Revisit when

Revisit the detailed mapping model when Step 02/03 define ERP users and tenants,
and again before Step 05 implements external linking. The stable identity
principle remains unless superseded by a deliberate architecture decision.

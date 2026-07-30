# ADR 0002: Native and external HRMS providers

Status: Accepted

Date: 2026-07-30

## Context

The ERP must include a complete native HRMS for customers without one. The
house tenant already uses a standalone HRMS that owns its own identities,
sessions, roles, and HRMS data.

The ERP needs one frontend and one internal HRMS contract without copying the
entire external database or turning the product into a generic integration
platform.

## Decision

The ERP owns a canonical HRMS application and HTTP boundary. Behind it are two
real provider implementations:

- a native HRMS backed by ERP PostgreSQL and ERP authentication/RBAC;
- a narrow compiled Go adapter to the standalone house HRMS.

Each tenant selects exactly one authoritative HRMS provider.

Rules:

- never make both providers writable authorities for one tenant;
- never dual-write native and external HRMS state;
- treat provider changes as explicit migration and reconciliation operations;
- resolve the provider in one tenant-scoped application location;
- never scatter provider-name branches through handlers, frontends, or other
  ERP modules;
- keep provider payloads and transport details inside the adapter;
- report provider capabilities truthfully;
- do not implement placeholder provider success;
- add another adapter only for a concrete ERP-module requirement.

## Consequences

Positive:

- the frontend consumes one contract;
- other ERP modules are provider-independent;
- the house HRMS remains independently usable and authoritative;
- provider failures can be isolated to HRMS behavior;
- native HRMS development remains a first-class product path.

Tradeoffs:

- the canonical contract must represent capability differences explicitly;
- external failures and authentication require translation and recovery;
- provider migration needs deliberate tooling rather than a configuration flip;
- shared provider tests are necessary to prevent semantic drift.

## Alternatives rejected

### Replace or rewrite the house HRMS authentication

Rejected because connecting the ERP does not justify changing a working
standalone authentication boundary.

### Copy the external database into the ERP

Rejected because it creates competing sources of truth, synchronization, and
conflict resolution without a first-version requirement.

### Generic HTTP proxy or mapping framework

Rejected because it cannot safely express provider-specific authentication,
malformed responses, session recovery, authorization, idempotency, and future
write reconciliation without recreating application logic as configuration.

## Revisit when

Revisit the provider set only when a real tenant and workflow require another
supported ERP integration. Revisit provider authority only through a separately
approved migration design.

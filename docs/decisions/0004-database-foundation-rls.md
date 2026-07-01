# ADR 0004: Database Foundation and RLS Baseline

## Status

Accepted

## Context

The ERP platform requires strong tenant separation while starting with a shared PostgreSQL database.

The first database foundation must support:

- control plane tenant/module/subscription state
- ERP tenant users and RBAC
- document metadata
- audit logging
- outbox events
- tenant-owned module tables
- PostgreSQL Row Level Security

## Decision

The first migration creates platform schemas and baseline tables.

Schemas:

- control
- core
- documents
- audit
- workflow
- inventory
- sales
- purchase
- accounting
- reports

Tenant-owned application tables receive:

- tenant_id
- Row Level Security
- FORCE ROW LEVEL SECURITY
- policies based on core.current_tenant_id()

Control plane tables are not tenant-scoped application tables and do not use tenant RLS in the same way.

Outbox events include tenant_id, but are infrastructure data processed by workers. They are not exposed as tenant business tables.

## Consequences

Every request path that touches tenant-owned tables must eventually set app.tenant_id inside the database transaction.

Missing tenant context should deny tenant data access by default.

Future migrations must preserve tenant_id and RLS discipline.

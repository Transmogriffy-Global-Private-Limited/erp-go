# ADR 0006: Tenant-Scoped Database Transactions

## Status

Accepted

## Context

Tenant-owned database tables use PostgreSQL Row Level Security policies based on:

core.current_tenant_id()

That function reads the transaction-local setting:

app.tenant_id

Application code must set this value before touching tenant-owned tables.

## Decision

All tenant-owned database operations must run inside a tenant-scoped transaction.

The shared helper is:

internal/platform/db.WithTenantTx()

It opens a transaction and runs:

SELECT set_config('app.tenant_id', $1, true)

before executing tenant-owned database work.

## Consequences

Tenant-owned module stores must not query tenant-owned tables directly from the pool.

They must use WithTenantTx or an equivalent helper.

Missing tenant context should fail before data access or be denied by RLS.

This is now the required foundation for tenant-owned ERP module APIs.

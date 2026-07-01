# ADR 0003: Tenant Isolation and Encryption

## Status

Accepted

## Context

ERP data is sensitive.

Examples:

- ledgers
- invoices
- salary data
- employee records
- customer data
- supplier data
- contracts
- inventory records
- business reports

Strong tenant separation is mandatory.

The question of full E2EE was considered.

## Decision

Do not promise full ERP-wide E2EE.

Use strong tenant isolation and selective encryption.

Initial standard tenant model:

- shared PostgreSQL database
- tenant_id on every tenant-owned table
- PostgreSQL Row Level Security
- tenant-scoped cache keys
- tenant-scoped event subjects/payloads
- tenant-scoped object storage paths
- tenant-scoped jobs and reports
- tenant-scoped audit logs

Enterprise tenant option later:

- dedicated database
- dedicated object bucket
- dedicated encryption keys
- optional customer-managed keys
- optional dedicated deployment

Encryption model:

- TLS in transit
- encryption at rest
- envelope encryption for sensitive fields/files
- selective E2EE only where server-side computation is not required

## Why not full ERP-wide E2EE?

The server must compute:

- reports
- accounting ledgers
- inventory valuation
- workflow transitions
- search
- document generation
- tax calculations
- integrations
- alerts

If all ERP data is unreadable to the server, the system cannot function as a normal ERP.

## Consequences

Sensitive documents and secrets can receive stronger encryption.

Operational ERP data remains server-readable but strongly isolated, audited, and encrypted at rest.

Superadmin support access must be explicit, scoped, time-limited, reason-required, and audited.

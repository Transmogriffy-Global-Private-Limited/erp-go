# ADR 0044: Sales Customers

## Status

Accepted

## Context

The Sales module needs stable customer master data before Sales Orders, fulfillment, invoicing, receivables, and customer reporting can be modeled safely.

Free-text customer names would prevent reliable tenant-scoped references and lifecycle history.

## Decision

Add tenant-scoped Sales Customers.

New table:

- sales.customers

New permissions:

- sales.customer.read
- sales.customer.write

New endpoints:

- GET /api/v1/sales/customers
- POST /api/v1/sales/customers

Customer codes are normalized to uppercase and unique within a tenant. The same code may be used by different tenants.

Successful creation writes:

- audit action: sales.customer.create
- outbox event: sales.customer.created.v1

## Consequences

Sales Orders can reference durable customer identities without crossing module ownership boundaries.

Customer access requires ERP sessions, Sales entitlement, tenant RBAC, tenant-scoped transactions, and PostgreSQL RLS.

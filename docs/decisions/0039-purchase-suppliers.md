# ADR 0039: Purchase Suppliers

## Status

Accepted

## Context

Purchase Receipts can bring goods into the Inventory stock ledger, but the Purchase module does not yet own supplier master data.

Purchase Orders, supplier documents, payable integration, and supplier reporting need stable tenant-scoped supplier identities rather than repeated free-text names.

## Decision

Add tenant-scoped Purchase Suppliers.

New table:

- purchase.suppliers

New permissions:

- purchase.supplier.read
- purchase.supplier.write

New endpoints:

- GET /api/v1/purchase/suppliers
- POST /api/v1/purchase/suppliers

Supplier codes are normalized to uppercase and unique within a tenant. The same code may be used by different tenants.

Successful supplier creation writes, in the same tenant-scoped transaction:

- audit action: purchase.supplier.create
- outbox event: purchase.supplier.created.v1

Existing Purchase Receipts keep their current free-text supplier_name contract in this slice.

## Consequences

The Purchase module now owns the supplier master needed by Purchase Orders and later payable workflows.

Supplier reads and writes require ERP sessions, Purchase module entitlement, RBAC permissions, tenant-scoped transactions, and PostgreSQL RLS.

A later migration can link Purchase Orders and Purchase Receipts to supplier_id without silently changing the current receipt API.

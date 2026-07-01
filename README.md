# ERP Go

Cloud-first multi-tenant ERP SaaS platform written in Go.

## Current shape

This repository currently contains the first bootable backend spine:

- control-plane-api
- erp-api
- shared HTTP response helpers
- tenant context helper
- temporary user context helper
- shared PostgreSQL runtime connection helper
- tenant-scoped DB transaction helper
- native PowerShell migration workflow
- DB-backed module registry reads
- DB-backed control-plane tenant APIs
- control-plane tenant module entitlement APIs
- ERP-side module entitlement enforcement
- ERP-side RBAC permission enforcement
- first tenant-owned Inventory items API
- audit and outbox writes on Inventory item creation
- tenant-isolation verification script
- module-entitlement verification script
- RBAC verification script
- audit/outbox verification script

## Control Plane

The control plane manages platform concerns:

- tenants
- plans
- subscriptions
- licensing
- module entitlements
- support access
- superadmin users

## ERP Plane

The ERP plane manages tenant business runtime:

- tenant users
- roles and permissions
- modules
- documents
- workflow
- inventory
- sales
- purchase
- accounting

Mutation pattern:

- business row
- audit row
- outbox event
- one transaction

Current Inventory mutation:

- POST /api/v1/inventory/items writes inventory.items, audit.audit_log, and core.outbox_events.

## Local database

This project uses native PostgreSQL for local development.

It does not use Docker or containers.

Use .env.example as the template and create a local .env.

Check database connectivity:

.\scripts\db-check.ps1

Seed local dev tenant:

.\scripts\seed-dev-tenant.ps1

Seed local dev RBAC:

.\scripts\seed-dev-rbac.ps1

Verify tenant isolation:

.\scripts\verify-tenant-isolation.ps1

Verify module entitlement enforcement:

.\scripts\verify-module-entitlement.ps1

Verify RBAC permission enforcement:

.\scripts\verify-rbac.ps1

Verify audit/outbox mutation writes:

.\scripts\verify-audit-outbox.ps1

## Run locally

Control plane API:

.\scripts\run-control-plane-api.ps1

ERP API:

.\scripts\run-erp-api.ps1

Inventory items require:

- X-Tenant-ID
- X-User-ID

Example:

$headers = @{
  "X-Tenant-ID" = "00000000-0000-0000-0000-000000000001"
  "X-User-ID" = "11111111-1111-1111-1111-111111111111"
}

Invoke-RestMethod http://localhost:8080/api/v1/inventory/items -Headers $headers

## Project memory

Read these first:

- AGENTS.md
- docs/PROJECT_STATE.md
- docs/ARCHITECTURE_BASELINE.md
- docs/AI_CHANGELOG.md

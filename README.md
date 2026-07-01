# ERP Go

Cloud-first multi-tenant ERP SaaS platform written in Go.

## Current shape

This repository currently contains the first bootable backend spine:

- control-plane-api
- erp-api
- erp-worker
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
- DB-backed outbox worker v1
- master verification script

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

Worker pattern:

- claim outbox events
- process event
- mark published

## Local database

This project uses native PostgreSQL for local development.

It does not use Docker or containers.

Use .env.example as the template and create a local .env.

## Run locally

Control plane API:

.\scripts\run-control-plane-api.ps1

ERP API:

.\scripts\run-erp-api.ps1

ERP worker once:

.\scripts\run-erp-worker.ps1 -Once -Limit 100

ERP worker continuous:

.\scripts\run-erp-worker.ps1

## Verification

With control-plane-api and erp-api running, verify the full project spine:

.\scripts\verify-all.ps1

Individual checks:

.\scripts\db-check.ps1
.\scripts\verify-rbac.ps1
.\scripts\verify-tenant-isolation.ps1
.\scripts\verify-module-entitlement.ps1
.\scripts\verify-audit-outbox.ps1
.\scripts\verify-outbox-worker.ps1

## Temporary local identity

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

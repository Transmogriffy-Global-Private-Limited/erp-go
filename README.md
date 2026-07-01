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
- tenant-isolation verification script
- module-entitlement verification script
- RBAC verification script

## Control Plane

The control plane manages platform concerns:

- tenants
- plans
- subscriptions
- licensing
- module entitlements
- support access
- superadmin users

Current control-plane endpoints:

- GET /control/v1/modules
- GET /control/v1/tenants
- POST /control/v1/tenants
- GET /control/v1/tenants/{tenant_id}/modules
- POST /control/v1/tenants/{tenant_id}/modules/{module_id}/enable
- POST /control/v1/tenants/{tenant_id}/modules/{module_id}/disable

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

Module-specific ERP APIs must be entitlement-guarded and permission-guarded.

Current Inventory permission rules:

- GET /api/v1/inventory/items requires inventory.item.read
- POST /api/v1/inventory/items requires inventory.item.write

Temporary local identity:

- X-Tenant-ID
- X-User-ID

X-User-ID is a placeholder until real authentication is introduced.

## Local database

This project uses native PostgreSQL for local development.

It does not use Docker or containers.

Use .env.example as the template and create a local .env.

Check database connectivity:

.\scripts\db-check.ps1

Apply a migration:

.\scripts\apply-migration.ps1 -Name 000002_runtime_role_grants -Direction up

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

## Run locally

Control plane API:

.\scripts\run-control-plane-api.ps1

ERP API:

.\scripts\run-erp-api.ps1

Health checks:

Invoke-RestMethod http://localhost:8081/healthz
Invoke-RestMethod http://localhost:8081/healthz/db
Invoke-RestMethod http://localhost:8080/healthz
Invoke-RestMethod http://localhost:8080/healthz/db

Module endpoints:

Invoke-RestMethod http://localhost:8081/control/v1/modules
Invoke-RestMethod http://localhost:8080/api/v1/modules -Headers @{ "X-Tenant-ID" = "00000000-0000-0000-0000-000000000001" }

Inventory items:

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

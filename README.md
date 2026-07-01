# ERP Go

Cloud-first multi-tenant ERP SaaS platform written in Go.

## Current shape

This repository currently contains the first bootable backend spine:

- control-plane-api
- erp-api
- shared HTTP response helpers
- tenant context helper
- shared PostgreSQL runtime connection helper
- tenant-scoped DB transaction helper
- native PowerShell migration workflow
- DB-backed module registry reads
- DB-backed control-plane tenant APIs
- first tenant-owned Inventory items API
- tenant-isolation verification script

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

Seed a second local tenant:

.\scripts\seed-dev-tenant.ps1 -TenantID "00000000-0000-0000-0000-000000000002" -Slug "dev-tenant-b" -LegalName "Dev Tenant B Private Limited" -DisplayName "Dev Tenant B"

Verify tenant isolation:

.\scripts\verify-tenant-isolation.ps1

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

Tenant endpoints:

Invoke-RestMethod http://localhost:8081/control/v1/tenants

$tenant = @{
  slug = "sample-tenant"
  legal_name = "Sample Tenant Private Limited"
  display_name = "Sample Tenant"
  status = "trial"
} | ConvertTo-Json

Invoke-RestMethod http://localhost:8081/control/v1/tenants -Method Post -ContentType "application/json" -Body $tenant

Inventory items:

Invoke-RestMethod http://localhost:8080/api/v1/inventory/items -Headers @{ "X-Tenant-ID" = "00000000-0000-0000-0000-000000000001" }

$item = @{
  sku = "DEV-ITEM-001"
  name = "Dev Item 001"
  description = "First dev inventory item"
} | ConvertTo-Json

Invoke-RestMethod http://localhost:8080/api/v1/inventory/items -Method Post -ContentType "application/json" -Headers @{ "X-Tenant-ID" = "00000000-0000-0000-0000-000000000001" } -Body $item

## Project memory

Read these first:

- AGENTS.md
- docs/PROJECT_STATE.md
- docs/ARCHITECTURE_BASELINE.md
- docs/AI_CHANGELOG.md

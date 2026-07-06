# ERP Go

Cloud-first multi-tenant ERP SaaS platform written in Go.

## Current shape

This repository currently contains the first bootable backend spine:

- control-plane-api
- erp-api
- erp-worker
- temporary control-plane auth guard
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
- tenant-owned Purchase suppliers API
- tenant-owned Purchase Order lifecycle API
- approved-order-backed Purchase receiving with over-receipt prevention
- derived Purchase Order receipt progress and lifecycle states
- compensating Purchase Receipt reversal with immutable stock history
- audit and outbox writes on Inventory item creation
- DB-backed outbox worker v1
- master verification script

## Temporary local identity

Control-plane APIs require:

- X-Platform-Session

Protected ERP business APIs require:

- X-Tenant-ID
- X-ERP-Session

Session tokens are obtained through the platform or tenant login endpoints.

## Verification

With control-plane-api and erp-api running:

.\scripts\verify-all.ps1

Individual checks:

.\scripts\db-check.ps1
.\scripts\verify-control-plane-auth.ps1
.\scripts\verify-rbac.ps1
.\scripts\verify-tenant-isolation.ps1
.\scripts\verify-module-entitlement.ps1
.\scripts\verify-audit-outbox.ps1
.\scripts\verify-outbox-worker.ps1
.\scripts\verify-purchase-suppliers.ps1
.\scripts\verify-purchase-orders.ps1

## Run locally

Control plane API:

.\scripts\run-control-plane-api.ps1

ERP API:

.\scripts\run-erp-api.ps1

ERP worker once:

.\scripts\run-erp-worker.ps1 -Once -Limit 100

ERP worker continuous:

.\scripts\run-erp-worker.ps1

## Project memory

Read these first:

- AGENTS.md
- docs/PROJECT_STATE.md
- docs/ARCHITECTURE_BASELINE.md
- docs/AI_CHANGELOG.md


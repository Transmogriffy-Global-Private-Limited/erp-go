# ERP Go

Cloud-first multi-tenant ERP SaaS platform written in Go.

## Current shape

This repository currently contains the first bootable backend spine:

- control-plane-api
- erp-api
- shared HTTP response helpers
- tenant context helper
- shared PostgreSQL runtime connection helper
- native PowerShell migration workflow

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

## Local database

This project uses native PostgreSQL for local development.

It does not use Docker or containers.

Use .env.example as the template and create a local .env.

Check database connectivity:

.\scripts\db-check.ps1

Apply a migration:

.\scripts\apply-migration.ps1 -Name 000002_runtime_role_grants -Direction up

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

Tenant-scoped ERP endpoint:

Invoke-RestMethod http://localhost:8080/api/v1/modules -Headers @{ "X-Tenant-ID" = "00000000-0000-0000-0000-000000000001" }

## Project memory

Read these first:

- AGENTS.md
- docs/PROJECT_STATE.md
- docs/ARCHITECTURE_BASELINE.md
- docs/AI_CHANGELOG.md

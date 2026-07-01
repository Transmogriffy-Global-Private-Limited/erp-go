# ERP Go

Cloud-first multi-tenant ERP SaaS platform written in Go.

## Current shape

This repository currently contains the first bootable backend spine:

- control-plane-api
- erp-api
- shared HTTP response helpers
- tenant context helper

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

## Run locally

Control plane API:

go run ./cmd/control-plane-api

ERP API:

go run ./cmd/erp-api

Health checks:

Invoke-RestMethod http://localhost:8081/healthz
Invoke-RestMethod http://localhost:8080/healthz

Tenant-scoped ERP endpoint:

Invoke-RestMethod http://localhost:8080/api/v1/modules -Headers @{ "X-Tenant-ID" = "00000000-0000-0000-0000-000000000001" }

## Project memory

Read these first:

- AGENTS.md
- docs/PROJECT_STATE.md
- docs/ARCHITECTURE_BASELINE.md
- docs/AI_CHANGELOG.md

# Architecture Baseline

## One-line product definition

A fully online multi-tenant ERP SaaS platform with a control plane for superadmin/licensing/module subscriptions and an ERP plane for tenant business operations.

## Architecture split

ERP SaaS Platform:

- Control Plane
- ERP Plane

## Control Plane

The control plane is owned by the platform/operator.

It manages:

- tenants
- plans
- subscriptions
- module entitlements
- licensing
- tenant limits
- platform users
- support sessions
- billing hooks
- usage records
- module registry
- platform audit

The superadmin panel talks to the control plane.

## ERP Plane

The ERP plane is used by tenant/customer users.

It manages:

- tenant users
- roles
- permissions
- branches
- departments
- documents
- workflow
- notifications
- inventory
- sales
- purchase
- accounting
- reports

## Initial deployables

The first deployable set should be:

- control-plane-api
- erp-api
- erp-worker
- erp-realtime

Initial frontend apps later:

- superadmin-web
- erp-web

## Intended stack

Backend: Go
Frontend: React + TypeScript
Database: PostgreSQL
Cache: Redis
Event Bus: NATS JetStream
Filestore: S3-compatible object storage
Deployment: no Docker/container local workflow; deployment strategy to be decided later

## Database model

Initial model:

- shared PostgreSQL database
- schemas by platform/domain/module
- tenant_id on all tenant-owned tables
- PostgreSQL Row Level Security on tenant-owned tables

Expected schemas:

- control
- core
- documents
- audit
- workflow
- inventory
- sales
- purchase
- accounting
- reports

## Tenant isolation baseline

Tenant separation must apply to:

- database rows
- object storage keys/buckets
- cache keys
- event subjects/payloads
- jobs
- reports
- audit entries
- logs
- support sessions
- realtime messages

Standard tenant isolation:

- shared DB
- tenant_id
- RLS
- tenant-scoped objects/events/cache

Enterprise tenant isolation:

- dedicated DB
- dedicated bucket
- dedicated key
- optional dedicated deployment

## Encryption baseline

Use:

- TLS in transit
- encryption at rest
- envelope encryption for sensitive fields/files
- optional customer-managed keys for enterprise tenants

Do not promise full ERP-wide E2EE.

Selective E2EE/envelope encryption may be used for:

- employee documents
- contracts
- private attachments
- bank account details
- API secrets
- integration credentials
- confidential notes

Server-readable data is required for:

- accounting
- reporting
- inventory calculations
- workflow
- search
- PDF generation
- integrations
- tax calculations

## Module model

Modules should be manifest-driven.

A module manifest should declare:

- module id
- display name
- version
- dependencies
- permissions
- API routes
- UI routes later
- database migrations
- events published
- events subscribed
- jobs
- reports
- file attachment types
- limits

Initial business modules:

- inventory
- sales
- purchase
- accounting

## Inter-module rule

Modules do not directly write each other's tables.

Examples:

- Sales must not directly mutate stock balances.
- Inventory owns stock.
- Accounting owns ledgers and journal entries.
- Documents owns file metadata.
- Workflow owns approval state.
- Notifications owns delivery.

## Command/event rule

APIs perform commands.
Events announce facts.
UI broadcasts invalidate views.

## Superadmin rule

Superadmin controls platform configuration, not tenant business data.

Support access must be:

- explicit
- scoped
- time-limited
- reason-required
- audited

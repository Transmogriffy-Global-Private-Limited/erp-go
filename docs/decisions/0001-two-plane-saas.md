# ADR 0001: Two-plane SaaS Architecture

## Status

Accepted

## Context

The product is a fully online ERP SaaS platform.

It requires:

- superadmin panel
- tenant management
- licensing
- module subscriptions
- ERP business modules
- strong tenant separation

## Decision

The platform will be split into two planes:

1. Control Plane
2. ERP Plane

The control plane manages platform/operator concerns.

The ERP plane manages tenant business concerns.

## Control Plane responsibilities

- tenants
- plans
- subscriptions
- licensing
- module entitlements
- tenant limits
- platform users
- support access
- billing hooks
- usage records

## ERP Plane responsibilities

- tenant users
- roles and permissions
- organization structure
- documents
- workflow
- inventory
- sales
- purchase
- accounting
- reports

## Consequences

Superadmin/platform logic must not be mixed into business modules casually.

Tenant ERP business logic must not depend directly on billing/licensing implementation details.

Entitlements should be cached/propagated into ERP runtime rather than checked through a remote license call on every request.

# ADR 0026: Inventory APIs Require ERP Sessions

## Status

Accepted

## Context

Tenant-user ERP sessions now exist.

Inventory APIs previously used temporary user identity headers:

- X-Tenant-ID
- X-User-ID

This trusted caller-supplied user IDs.

## Decision

Inventory APIs now require:

- X-Tenant-ID
- X-ERP-Session

The ERP session resolves the active tenant user and injects tenant/user context for RBAC and audit.

X-User-ID is no longer accepted for Inventory APIs.

## Consequences

Inventory API authorization is now session-backed.

Verification scripts now login tenant users and call Inventory APIs with X-ERP-Session.

Other future ERP modules should use the same ERP session middleware.

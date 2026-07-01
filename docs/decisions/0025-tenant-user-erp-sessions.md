# ADR 0025: Tenant User ERP Sessions

## Status

Accepted

## Context

ERP APIs currently use temporary headers:

- X-Tenant-ID
- X-User-ID

This is useful for early development but should be replaced by tenant-user sessions.

## Decision

Add tenant-user login/session groundwork.

New endpoints:

- POST /api/v1/auth/login
- GET /api/v1/auth/me
- POST /api/v1/auth/logout

Tenant users authenticate with:

- tenant_id
- email
- password

The login endpoint returns an ERP session token.

Session-authenticated endpoints use:

- X-Tenant-ID
- X-ERP-Session

For this step, existing ERP business APIs still accept X-User-ID.

## Consequences

ERP tenant-user session support exists.

Future work should migrate protected ERP business APIs from X-User-ID to X-ERP-Session.

# ADR 0012: Temporary Control Plane Auth Guard

## Status

Accepted

## Context

Control-plane APIs manage platform-level state:

- tenants
- module entitlements
- plans
- subscriptions
- licensing
- support access

These APIs must not remain open.

Real authentication is not implemented yet.

## Decision

Add a temporary control-plane auth guard using request headers:

- X-Platform-User-ID
- X-Platform-Role: superadmin

Health endpoints remain public.

All /control/v1/* endpoints require the temporary superadmin guard.

## Consequences

Control-plane APIs are no longer openly callable in local development.

This is temporary and must be replaced by real platform authentication later.

Verification must include control-plane auth behavior.

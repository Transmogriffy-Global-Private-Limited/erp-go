# ADR 0017: DB-backed Platform Superadmin Check

## Status

Accepted

## Context

The previous temporary control-plane auth guard required:

- X-Platform-User-ID
- X-Platform-Role: superadmin

This trusted a caller-supplied role header.

## Decision

Keep X-Platform-User-ID as a temporary identity carrier, but stop trusting X-Platform-Role.

The control plane now checks control.platform_users for:

- id = X-Platform-User-ID
- status = active
- role = superadmin

X-Platform-Role may still be sent by old scripts, but it is ignored for authorization.

## Consequences

Control-plane authorization is now DB-backed.

This is still not real login/session auth, but it removes trusted role headers from the control-plane authorization decision.

Future work should replace X-Platform-User-ID with a real session/JWT identity.

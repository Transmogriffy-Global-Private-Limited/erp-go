# ADR 0021: Platform Logout Revokes Sessions

## Status

Accepted

## Context

Platform login and sessions exist.

Users need a way to explicitly end a platform control-plane session.

## Decision

Add:

POST /control/v1/auth/logout

It requires X-Platform-Session.

Logout marks the session as:

- status = revoked
- revoked_at = now()

The same session token must be rejected after logout.

## Consequences

Platform sessions now have an explicit revoke path.

Future work can add revoke-all-sessions, session listing, expiry cleanup, and audit logging for login/logout.

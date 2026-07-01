# ADR 0024: Platform Session Management

## Status

Accepted

## Context

Platform login/logout sessions exist.

Operators need visibility into active sessions and a way to revoke all sessions for the current platform user.

Expired sessions should also have a cleanup path.

## Decision

Add session management endpoints:

- GET /control/v1/auth/sessions
- POST /control/v1/auth/sessions/revoke-all
- POST /control/v1/auth/sessions/cleanup-expired

Revoke-all revokes all active sessions for the current platform user.

Cleanup-expired revokes expired active sessions globally and writes platform audit.

## Consequences

Platform sessions now have basic operational management.

Future work can add admin-level user session listing, revoke a specific session, device metadata, IP metadata, and automated cleanup through a worker.

# ADR 0022: Platform Auth Login Logout Audit

## Status

Accepted

## Context

Platform sessions exist, and platform logout revokes sessions.

Authentication lifecycle events are security-relevant control-plane events.

## Decision

Audit successful platform login and logout events in control.platform_audit_log.

Actions:

- control.platform_auth.login
- control.platform_auth.logout

Target type:

- control.platform_session

The platform actor is the authenticated platform user.

## Consequences

Successful platform auth lifecycle events are now accountable.

Future work can add failed-login audit, rate limiting, lockout, session listing, and revoke-all-sessions.

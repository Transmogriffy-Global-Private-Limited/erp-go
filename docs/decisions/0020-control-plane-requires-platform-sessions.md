# ADR 0020: Control Plane Requires Platform Sessions

## Status

Accepted

## Context

Platform login and sessions exist.

Control-plane verification scripts now use X-Platform-Session.

The temporary X-Platform-User-ID fallback is no longer needed and is unsafe to keep.

## Decision

Remove X-Platform-User-ID fallback from protected control-plane routes.

Protected control-plane APIs now require:

- X-Platform-Session

The session must belong to an active superadmin user in control.platform_users.

## Consequences

Control-plane authorization no longer accepts direct platform user identity headers.

Local scripts must login and pass X-Platform-Session.

Future platform auth work should improve session lifecycle, logout, expiry handling, and password policy.

# ADR 0023: Failed Platform Login Audit and Lockout

## Status

Accepted

## Context

Platform login/logout audit exists for successful auth lifecycle events.

Failed login attempts are also security-relevant.

The platform should not allow unlimited bad password attempts.

## Decision

Add failed platform login tracking to control.platform_users:

- failed_login_count
- last_failed_login_at
- locked_until

Failed platform login writes:

- control.platform_auth.login_failed

After five failed attempts, the platform user is temporarily locked for fifteen minutes.

Locked login attempts return HTTP 423.

The local seed script resets lockout state for the dev superadmin.

## Consequences

Platform auth has basic brute-force protection groundwork.

Future work can make lockout thresholds configurable, add IP/device metadata, add failed-login rate limits for unknown emails, and notify superadmins.

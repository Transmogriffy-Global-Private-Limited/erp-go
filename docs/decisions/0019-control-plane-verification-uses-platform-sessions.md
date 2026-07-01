# ADR 0019: Control Plane Verification Uses Platform Sessions

## Status

Accepted

## Context

Platform sessions were added, but control-plane verification scripts still used the temporary X-Platform-User-ID fallback.

Before removing the fallback, all scripts should prove that X-Platform-Session works everywhere.

## Decision

Add a reusable PowerShell helper:

scripts/Get-PlatformSessionHeaders.ps1

It seeds the dev platform superadmin, logs in, and returns:

- X-Platform-Session

Control-plane verification scripts now use platform sessions for control-plane API calls.

## Consequences

Verification no longer depends on X-Platform-User-ID for ordinary control-plane operations.

The server still keeps X-Platform-User-ID fallback temporarily.

The next step can remove that fallback safely.

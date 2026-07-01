# ADR 0027: Local Verification Restarts APIs

## Status

Accepted

## Context

Local verification was repeatedly hitting stale running API processes after code changes.

This caused confusing failures where Go tests passed but HTTP verification still exercised old server code.

## Decision

Add scripts to restart local APIs before verification:

- scripts/restart-local-apis.ps1
- scripts/ensure-local-apis.ps1

verify-all.ps1 now restarts local APIs before running checks.

High-touch individual verification scripts also call ensure-local-apis.ps1 when run directly.

## Consequences

Local verification now tests fresh code instead of stale running processes.

The scripts intentionally kill processes listening on local control-plane and ERP API ports.

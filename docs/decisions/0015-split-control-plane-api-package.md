# ADR 0015: Split Control Plane API Package

## Status

Accepted

## Context

cmd/control-plane-api/main.go had grown into a large file containing boot logic, routing, auth middleware, and all control-plane handlers.

This made future control-plane work risky.

## Decision

Move control-plane HTTP routing and handlers into:

internal/platform/controlapi

Keep cmd/control-plane-api/main.go limited to process boot:

- load environment
- connect database
- construct HTTP server
- start listener

## Consequences

The control-plane API is easier to extend.

Future tenant, module, plan, subscription, auth, and audit handlers should live in the controlapi package or smaller subpackages.

This refactor is behavior-preserving and must be verified with scripts/verify-all.ps1.

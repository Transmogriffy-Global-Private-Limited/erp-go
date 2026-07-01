# ADR 0030: ERP Middleware Extracted

## Status

Accepted

## Context

cmd/erp-api/main.go still contained middleware and guard logic after route registration was extracted.

This made the boot file harder to read and increased the risk of mixing process boot, route wiring, and authorization behavior.

## Decision

Move ERP middleware and guard functions into:

cmd/erp-api/middleware.go

Moved functions:

- requireModule
- requirePermission
- tenantMiddleware
- userMiddleware

## Consequences

cmd/erp-api/main.go is closer to process boot only.

No behavior change is intended.

Future work should continue splitting ERP handlers by surface area.

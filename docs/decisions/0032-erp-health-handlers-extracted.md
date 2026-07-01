# ADR 0032: ERP Health Handlers Extracted

## Status

Accepted

## Context

After ERP routes, middleware, and business handlers were extracted, cmd/erp-api/main.go still contained health handlers.

## Decision

Move ERP health handlers into:

cmd/erp-api/health_handlers.go

Moved functions:

- healthHandler
- dbHealthHandler

## Consequences

cmd/erp-api/main.go is closer to process boot only.

No behavior change is intended.

# ADR 0033: ERP App Construction Extracted

## Status

Accepted

## Context

cmd/erp-api/main.go was being reduced toward process boot only.

After extracting routes, middleware, business handlers, and health handlers, main.go still owned app struct construction.

## Decision

Move ERP app struct and construction into:

cmd/erp-api/app.go

New constructor:

newApp(pool *pgxpool.Pool) *app

## Consequences

cmd/erp-api/main.go is now mostly process boot:

- load environment
- connect database
- construct app
- start HTTP server

No behavior change is intended.

# ADR 0054: FE Local Integration Bootstrap

## Status

Accepted

## Context

Frontend integration required several manual and order-sensitive steps: database checks, migrations, platform and tenant seeding, API startup, credential validation, and copying identifiers and credentials into FE configuration. The existing scripts implemented those individual operations but did not provide one stable handoff contract.

Local setup also needs to remain aligned with the platform/ERP plane boundary. A tenant business user must not be confused with the platform superadmin, and an authenticated no-access user is useful for verifying RBAC failures.

## Decision

Provide `scripts/setup-fe-dev.ps1` as the idempotent local orchestration entry point. It composes the existing database, migration, seed, and API startup scripts, validates the seeded identities through the real HTTP login flows, and emits a schema-versioned manifest at `.local/fe-integration.json`.

The manifest distinguishes three identities:

- tenant allowed user for ERP feature development
- tenant no-access user for permission-denial testing
- platform superadmin for control-plane testing only

Passwords are included because the artifact is explicitly local development data and must support testing the real login forms. Live session tokens are excluded and validation sessions are revoked by default. Tokens are retained only when the caller explicitly supplies `-IncludeSessionTokens`.

The helper may create a missing default `.env` from `.env.example`, but it never overwrites an existing environment file. The manifest is written beneath the already-ignored `.local/` directory. No database URLs or environment secrets are copied into it.

## Consequences

Frontend developers receive one repeatable command and one machine-readable integration contract without learning the repository's seed ordering or copying values from console output. Re-running the helper converges local seed state and resets the documented test passwords.

This remains a development bootstrap, not a production tenant-provisioning API or an SSO protocol. It does not change runtime authentication, tenant isolation, or the separation between control-plane and ERP identities.

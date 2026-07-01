# Native Local Development

This project does not use Docker or containers for local development.

## Environment files

Use .env.example as the committed template.

Use .env as the local machine-specific file.

Never commit .env.

## Database URLs

Two database URLs are used:

- MIGRATION_DATABASE_URL
- ERP_DATABASE_URL

MIGRATION_DATABASE_URL is used by local migration scripts.

ERP_DATABASE_URL will be used by the application runtime later.

## Scripts

Check DB connectivity:

.\scripts\db-check.ps1

Apply migration:

.\scripts\apply-migration.ps1 -Direction up

Rollback migration:

.\scripts\apply-migration.ps1 -Direction down

# Migrations

Database migrations for the ERP platform.

Initial convention:

- `*.up.sql` applies a migration.
- `*.down.sql` rolls back a migration.

The first migrations are plain SQL.

A Go migration runner will be added later.

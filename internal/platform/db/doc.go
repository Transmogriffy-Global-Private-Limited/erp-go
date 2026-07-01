package db

// Package db owns PostgreSQL runtime connectivity.
//
// Migration scripts use MIGRATION_DATABASE_URL.
// Application services use runtime database URLs such as:
// - CONTROL_PLANE_DATABASE_URL
// - ERP_DATABASE_URL
//
// Application code must not use the migration owner URL.

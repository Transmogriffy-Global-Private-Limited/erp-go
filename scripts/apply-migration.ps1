param(
  [ValidateSet("up", "down")]
  [string] $Direction = "up",

  [string] $Name = "000001_platform_foundation",

  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$EnvPath = Join-Path $RepoRoot $EnvFile

. (Join-Path $PSScriptRoot "Import-DotEnv.ps1") -Path $EnvPath

if (-not $env:MIGRATION_DATABASE_URL) {
  throw "MIGRATION_DATABASE_URL is missing. Copy .env.example to .env and edit it."
}

$MigrationsDir = $env:MIGRATIONS_DIR
if (-not $MigrationsDir) {
  $MigrationsDir = "./migrations"
}

$MigrationPath = Join-Path $RepoRoot (Join-Path $MigrationsDir "$Name.$Direction.sql")

if (-not (Test-Path $MigrationPath)) {
  throw "Migration file not found: $MigrationPath"
}

$EnsureLedgerSql = @"
CREATE TABLE IF NOT EXISTS public.schema_migrations (
    name TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
"@

psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -q -c $EnsureLedgerSql
if ($LASTEXITCODE -ne 0) {
  throw "Failed to ensure public.schema_migrations."
}

$CheckSql = "SELECT 1 FROM public.schema_migrations WHERE name = '$Name';"
$AppliedOutput = @(psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -At -c $CheckSql)
if ($LASTEXITCODE -ne 0) {
  throw "Failed to check migration status."
}

$Applied = ($AppliedOutput -join "").Trim()

if ($Direction -eq "up" -and $Applied -eq "1") {
  Write-Host "Migration already applied: $Name"
  return
}

if ($Direction -eq "down" -and $Applied -ne "1") {
  Write-Host "Migration is not marked as applied, skipping rollback: $Name"
  return
}

Write-Host "Applying migration: $MigrationPath"

psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -f $MigrationPath
if ($LASTEXITCODE -ne 0) {
  throw "Migration failed: $Name.$Direction"
}

if ($Direction -eq "up") {
  $RecordSql = "INSERT INTO public.schema_migrations (name) VALUES ('$Name') ON CONFLICT (name) DO NOTHING;"
  psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -q -c $RecordSql
  if ($LASTEXITCODE -ne 0) {
    throw "Migration applied but failed to record ledger entry: $Name"
  }
}

if ($Direction -eq "down") {
  $DeleteSql = "DELETE FROM public.schema_migrations WHERE name = '$Name';"
  psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -q -c $DeleteSql
  if ($LASTEXITCODE -ne 0) {
    throw "Migration rolled back but failed to delete ledger entry: $Name"
  }
}

Write-Host ""
Write-Host "Migration applied: $Name.$Direction"

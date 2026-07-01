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

Write-Host "Applying migration: $MigrationPath"

psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -f $MigrationPath

Write-Host ""
Write-Host "Migration applied: $Name.$Direction"

param(
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$EnvPath = Join-Path $RepoRoot $EnvFile

. (Join-Path $PSScriptRoot "Import-DotEnv.ps1") -Path $EnvPath

if (-not $env:MIGRATION_DATABASE_URL) {
  throw "MIGRATION_DATABASE_URL is missing. Copy .env.example to .env and edit it."
}

Write-Host "Checking psql availability..."
psql --version

Write-Host ""
Write-Host "Checking database connection..."
psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -c "SELECT current_database() AS database, current_user AS user_name;"

Write-Host ""
Write-Host "Database check passed."

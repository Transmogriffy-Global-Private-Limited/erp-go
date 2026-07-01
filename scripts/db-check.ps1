param(
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$EnvPath = Join-Path $RepoRoot $EnvFile

. (Join-Path $PSScriptRoot "Import-DotEnv.ps1") -Path $EnvPath

function Test-DatabaseUrl {
  param(
    [string] $Name,
    [string] $Url
  )

  if (-not $Url) {
    throw "$Name is missing. Check .env."
  }

  Write-Host ""
  Write-Host "Checking $Name..."
  psql $Url -v ON_ERROR_STOP=1 -c "SELECT current_database() AS database, current_user AS user_name;"
  if ($LASTEXITCODE -ne 0) {
    throw "$Name connection check failed."
  }
}

Write-Host "Checking psql availability..."
psql --version
if ($LASTEXITCODE -ne 0) {
  throw "psql is not available or failed to run."
}

Test-DatabaseUrl -Name "MIGRATION_DATABASE_URL" -Url $env:MIGRATION_DATABASE_URL
Test-DatabaseUrl -Name "CONTROL_PLANE_DATABASE_URL" -Url $env:CONTROL_PLANE_DATABASE_URL
Test-DatabaseUrl -Name "ERP_DATABASE_URL" -Url $env:ERP_DATABASE_URL

Write-Host ""
Write-Host "Database checks passed."

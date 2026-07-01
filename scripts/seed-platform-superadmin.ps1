param(
  [string] $PlatformUserID = "00000000-0000-0000-0000-00000000aaaa",
  [string] $Email = "dev.superadmin@example.test",
  [string] $DisplayName = "Dev Superadmin",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$EnvPath = Join-Path $RepoRoot $EnvFile

. (Join-Path $PSScriptRoot "Import-DotEnv.ps1") -Path $EnvPath

if (-not $env:MIGRATION_DATABASE_URL) {
  throw "MIGRATION_DATABASE_URL is missing. Copy .env.example to .env and edit it."
}

$Sql = @"
INSERT INTO control.platform_users (
    id,
    email,
    display_name,
    status,
    role
)
VALUES (
    '$PlatformUserID',
    '$Email',
    '$DisplayName',
    'active',
    'superadmin'
)
ON CONFLICT (id) DO UPDATE
SET email = EXCLUDED.email,
    display_name = EXCLUDED.display_name,
    status = EXCLUDED.status,
    role = EXCLUDED.role,
    updated_at = now();

SELECT id, email, display_name, status, role
FROM control.platform_users
WHERE id = '$PlatformUserID';
"@

psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -c $Sql
if ($LASTEXITCODE -ne 0) {
  throw "Failed to seed platform superadmin."
}

Write-Host ""
Write-Host "Platform superadmin seeded: $PlatformUserID"

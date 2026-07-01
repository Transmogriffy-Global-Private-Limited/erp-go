param(
  [string] $PlatformUserID = "00000000-0000-0000-0000-00000000aaaa",
  [string] $Email = "dev.superadmin@example.test",
  [string] $DisplayName = "Dev Superadmin",
  [string] $Password = "dev-superadmin-password",
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
    role,
    password_hash,
    failed_login_count,
    last_failed_login_at,
    locked_until
)
VALUES (
    '$PlatformUserID',
    '$Email',
    '$DisplayName',
    'active',
    'superadmin',
    crypt('$Password', gen_salt('bf')),
    0,
    NULL,
    NULL
)
ON CONFLICT (id) DO UPDATE
SET email = EXCLUDED.email,
    display_name = EXCLUDED.display_name,
    status = EXCLUDED.status,
    role = EXCLUDED.role,
    password_hash = EXCLUDED.password_hash,
    failed_login_count = 0,
    last_failed_login_at = NULL,
    locked_until = NULL,
    updated_at = now();

SELECT
    id,
    email,
    display_name,
    status,
    role,
    password_hash IS NOT NULL AS has_password,
    failed_login_count,
    locked_until
FROM control.platform_users
WHERE id = '$PlatformUserID';
"@

psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -c $Sql
if ($LASTEXITCODE -ne 0) {
  throw "Failed to seed platform superadmin."
}

Write-Host ""
Write-Host "Platform superadmin seeded: $PlatformUserID"
Write-Host "Platform superadmin email: $Email"

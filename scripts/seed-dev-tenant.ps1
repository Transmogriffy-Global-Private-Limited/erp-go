param(
  [string] $TenantID = "00000000-0000-0000-0000-000000000001",
  [string] $Slug = "dev-tenant",
  [string] $LegalName = "Dev Tenant Private Limited",
  [string] $DisplayName = "Dev Tenant",
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
INSERT INTO control.tenants (
    id,
    slug,
    legal_name,
    display_name,
    status
)
VALUES (
    '$TenantID',
    '$Slug',
    '$LegalName',
    '$DisplayName',
    'active'
)
ON CONFLICT (id) DO UPDATE
SET slug = EXCLUDED.slug,
    legal_name = EXCLUDED.legal_name,
    display_name = EXCLUDED.display_name,
    status = EXCLUDED.status,
    updated_at = now();

INSERT INTO control.tenant_enabled_modules (
    tenant_id,
    module_id
)
SELECT
    '$TenantID',
    id
FROM control.modules
ON CONFLICT (tenant_id, module_id) DO UPDATE
SET disabled_at = NULL;

SELECT
    t.id,
    t.slug,
    t.status,
    array_agg(tem.module_id ORDER BY tem.module_id) AS enabled_modules
FROM control.tenants t
LEFT JOIN control.tenant_enabled_modules tem ON tem.tenant_id = t.id
WHERE t.id = '$TenantID'
GROUP BY t.id, t.slug, t.status;
"@

psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -c $Sql
if ($LASTEXITCODE -ne 0) {
  throw "Failed to seed dev tenant: $TenantID"
}

Write-Host ""
Write-Host "Dev tenant seeded: $TenantID"

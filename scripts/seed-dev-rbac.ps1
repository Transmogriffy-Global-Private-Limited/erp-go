param(
  [string] $TenantID = "00000000-0000-0000-0000-000000000001",
  [string] $AllowedUserID = "11111111-1111-1111-1111-111111111111",
  [string] $NoAccessUserID = "22222222-2222-2222-2222-222222222222",
  [string] $AllowedRoleID = "33333333-3333-3333-3333-333333333333",
  [string] $NoAccessRoleID = "44444444-4444-4444-4444-444444444444",
  [string] $AllowedPassword = "dev-tenant-password",
  [string] $NoAccessPassword = "dev-noaccess-password",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$EnvPath = Join-Path $RepoRoot $EnvFile

. (Join-Path $PSScriptRoot "Import-DotEnv.ps1") -Path $EnvPath

if (-not $env:MIGRATION_DATABASE_URL) {
  throw "MIGRATION_DATABASE_URL is missing. Copy .env.example to .env and edit it."
}

$AllowedEmail = "inventory.allowed+$($TenantID.Substring(0, 8))@example.test"
$NoAccessEmail = "inventory.noaccess+$($TenantID.Substring(0, 8))@example.test"

$Sql = @"
BEGIN;

SELECT set_config('app.tenant_id', '$TenantID', true);

INSERT INTO core.tenant_users (
    id,
    tenant_id,
    email,
    display_name,
    status,
    password_hash
)
VALUES
    (
        '$AllowedUserID',
        '$TenantID',
        '$AllowedEmail',
        'Dev Inventory Allowed User',
        'active',
        crypt('$AllowedPassword', gen_salt('bf'))
    ),
    (
        '$NoAccessUserID',
        '$TenantID',
        '$NoAccessEmail',
        'Dev No Access User',
        'active',
        crypt('$NoAccessPassword', gen_salt('bf'))
    )
ON CONFLICT (id) DO UPDATE
SET email = EXCLUDED.email,
    display_name = EXCLUDED.display_name,
    status = EXCLUDED.status,
    password_hash = EXCLUDED.password_hash,
    updated_at = now();

INSERT INTO core.roles (
    id,
    tenant_id,
    name,
    description
)
VALUES
    (
        '$AllowedRoleID',
        '$TenantID',
        'Dev Inventory Operator',
        'Local dev role with inventory item/unit read/write permissions'
    ),
    (
        '$NoAccessRoleID',
        '$TenantID',
        'Dev No Access',
        'Local dev role without inventory permissions'
    )
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    updated_at = now();

INSERT INTO core.role_permissions (
    tenant_id,
    role_id,
    permission_id
)
VALUES
    ('$TenantID', '$AllowedRoleID', 'inventory.item.read'),
    ('$TenantID', '$AllowedRoleID', 'inventory.item.write'),
    ('$TenantID', '$AllowedRoleID', 'inventory.unit.read'),
    ('$TenantID', '$AllowedRoleID', 'inventory.unit.write'),
    ('$TenantID', '$AllowedRoleID', 'inventory.location.read'),
    ('$TenantID', '$AllowedRoleID', 'inventory.location.write'),
    ('$TenantID', '$AllowedRoleID', 'inventory.stock_movement.read'),
    ('$TenantID', '$AllowedRoleID', 'inventory.stock_movement.write'),
    ('$TenantID', '$AllowedRoleID', 'inventory.stock_balance.read'),
    ('$TenantID', '$AllowedRoleID', 'purchase.receipt.read'),
    ('$TenantID', '$AllowedRoleID', 'purchase.receipt.write')
ON CONFLICT (tenant_id, role_id, permission_id) DO NOTHING;

INSERT INTO core.user_roles (
    tenant_id,
    user_id,
    role_id
)
VALUES
    ('$TenantID', '$AllowedUserID', '$AllowedRoleID'),
    ('$TenantID', '$NoAccessUserID', '$NoAccessRoleID')
ON CONFLICT (tenant_id, user_id, role_id) DO NOTHING;

SELECT
    tu.tenant_id,
    tu.id AS user_id,
    tu.email,
    tu.status,
    tu.password_hash IS NOT NULL AS has_password,
    array_agg(r.name ORDER BY r.name) AS roles
FROM core.tenant_users tu
LEFT JOIN core.user_roles ur
    ON ur.tenant_id = tu.tenant_id
   AND ur.user_id = tu.id
LEFT JOIN core.roles r
    ON r.tenant_id = ur.tenant_id
   AND r.id = ur.role_id
WHERE tu.tenant_id = '$TenantID'
  AND tu.id IN ('$AllowedUserID', '$NoAccessUserID')
GROUP BY tu.tenant_id, tu.id, tu.email, tu.status, tu.password_hash
ORDER BY tu.email;

COMMIT;
"@

psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -c $Sql
if ($LASTEXITCODE -ne 0) {
  throw "Failed to seed dev RBAC."
}

Write-Host ""
Write-Host "Dev RBAC seeded for tenant: $TenantID"
Write-Host "Allowed user: $AllowedUserID"
Write-Host "Allowed email: $AllowedEmail"
Write-Host "No-access user: $NoAccessUserID"
Write-Host "No-access email: $NoAccessEmail"

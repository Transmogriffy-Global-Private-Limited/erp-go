param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $TenantID = "00000000-0000-0000-0000-000000000001",
  [string] $UserID = "11111111-1111-1111-1111-111111111111",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

. (Join-Path $PSScriptRoot "Import-DotEnv.ps1") -Path (Join-Path $RepoRoot $EnvFile)

if (-not $env:MIGRATION_DATABASE_URL) {
  throw "MIGRATION_DATABASE_URL is missing. Copy .env.example to .env and edit it."
}

$PlatformHeaders = & (Join-Path $PSScriptRoot "Get-PlatformSessionHeaders.ps1") `
  -ControlPlaneUrl $ControlPlaneUrl `
  -EnvFile $EnvFile

function Invoke-ScalarSql {
  param(
    [string] $Sql
  )

  $Output = @(psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -At -c $Sql)

  if ($LASTEXITCODE -ne 0) {
    throw "SQL query failed."
  }

  return (($Output | Select-Object -Last 1) -join "").Trim()
}

Write-Host "Seeding tenant..."
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") -TenantID $TenantID -EnvFile $EnvFile

Write-Host ""
Write-Host "Seeding RBAC..."
& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") -TenantID $TenantID -AllowedUserID $UserID -EnvFile $EnvFile

Write-Host ""
Write-Host "Ensuring inventory module is enabled..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/inventory/enable" -Method Post -Headers $PlatformHeaders | Out-Null

$Headers = @{
  "X-Tenant-ID" = $TenantID
  "X-User-ID" = $UserID
}

$Sku = "AUDIT-OUTBOX-" + (Get-Date -Format "yyyyMMddHHmmss") + "-" + (Get-Random -Minimum 1000 -Maximum 9999)

$Body = @{
  sku = $Sku
  name = "Audit Outbox Verification Item"
  description = "Created to verify audit and outbox mutation writes"
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating inventory item with SKU: $Sku"
$Created = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $Body

$ItemID = $Created.item.id

if (-not $ItemID) {
  throw "Inventory item creation did not return item.id."
}

Write-Host "Created item: $ItemID"

$AuditSql = @"
SELECT set_config('app.tenant_id', '$TenantID', false);
SELECT count(*)
FROM audit.audit_log
WHERE tenant_id = '$TenantID'
  AND action = 'inventory.item.create'
  AND target_type = 'inventory.item'
  AND target_id = '$ItemID';
"@

$AuditCount = Invoke-ScalarSql -Sql $AuditSql

if ($AuditCount -ne "1") {
  throw "Expected exactly one audit row for item $ItemID, got: $AuditCount"
}

Write-Host "Audit row verified."

$OutboxSql = @"
SELECT count(*)
FROM core.outbox_events
WHERE tenant_id = '$TenantID'
  AND event_type = 'inventory.item.created.v1'
  AND aggregate_type = 'inventory.item'
  AND aggregate_id = '$ItemID';
"@

$OutboxCount = Invoke-ScalarSql -Sql $OutboxSql

if ($OutboxCount -ne "1") {
  throw "Expected exactly one outbox row for item $ItemID, got: $OutboxCount"
}

Write-Host "Outbox row verified."

Write-Host ""
Write-Host "Audit/outbox verification passed."

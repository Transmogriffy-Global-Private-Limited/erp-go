param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $TenantID = "00000000-0000-0000-0000-000000000001",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

. (Join-Path $PSScriptRoot "Import-DotEnv.ps1") -Path (Join-Path $RepoRoot $EnvFile)

if (-not $env:MIGRATION_DATABASE_URL) {
  throw "MIGRATION_DATABASE_URL is missing. Copy .env.example to .env and edit it."
}

$PlatformActorID = "00000000-0000-0000-0000-00000000aaaa"

$PlatformHeaders = @{
  "X-Platform-User-ID" = $PlatformActorID
  "X-Platform-Role" = "superadmin"
}

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

function Assert-AuditExists {
  param(
    [string] $Action,
    [string] $TargetID,
    [string] $Label
  )

  $Sql = @"
SELECT count(*)
FROM control.platform_audit_log
WHERE platform_actor_id = '$PlatformActorID'
  AND action = '$Action'
  AND target_id = '$TargetID';
"@

  $Count = Invoke-ScalarSql -Sql $Sql

  if ([int]$Count -lt 1) {
    throw "$Label audit row not found. action=$Action target_id=$TargetID count=$Count"
  }

  Write-Host "$Label audit row verified."
}

Write-Host "Seeding tenant..."
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") -TenantID $TenantID -EnvFile $EnvFile

$PlanID = "audit-plan-" + (Get-Date -Format "yyyyMMddHHmmss") + "-" + (Get-Random -Minimum 1000 -Maximum 9999)

$PlanBody = @{
  id = $PlanID
  name = "Audit Plan " + $PlanID
  status = "active"
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating audited plan: $PlanID"
Invoke-RestMethod "$ControlPlaneUrl/control/v1/plans" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $PlatformHeaders `
  -Body $PlanBody | Out-Null

Assert-AuditExists `
  -Action "control.plan.create" `
  -TargetID $PlanID `
  -Label "Plan create"

Write-Host ""
Write-Host "Enabling audited plan module..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/plans/$PlanID/modules/inventory/enable" `
  -Method Post `
  -Headers $PlatformHeaders | Out-Null

Assert-AuditExists `
  -Action "control.plan_module.enable" `
  -TargetID "${PlanID}:inventory" `
  -Label "Plan module enable"

$SubscriptionBody = @{
  plan_id = $PlanID
  status = "active"
} | ConvertTo-Json

Write-Host ""
Write-Host "Assigning audited subscription..."
$Subscription = Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/subscription" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $PlatformHeaders `
  -Body $SubscriptionBody

$SubscriptionID = $Subscription.subscription.id

if (-not $SubscriptionID) {
  throw "Subscription assignment did not return subscription.id."
}

Assert-AuditExists `
  -Action "control.tenant_subscription.assign" `
  -TargetID $SubscriptionID `
  -Label "Tenant subscription assign"

Write-Host ""
Write-Host "Disabling audited tenant module..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/inventory/disable" `
  -Method Post `
  -Headers $PlatformHeaders | Out-Null

Assert-AuditExists `
  -Action "control.tenant_module.disable" `
  -TargetID "${TenantID}:inventory" `
  -Label "Tenant module disable"

Write-Host ""
Write-Host "Re-enabling audited tenant module..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/inventory/enable" `
  -Method Post `
  -Headers $PlatformHeaders | Out-Null

Assert-AuditExists `
  -Action "control.tenant_module.enable" `
  -TargetID "${TenantID}:inventory" `
  -Label "Tenant module enable"

Write-Host ""
Write-Host "Control-plane audit verification passed."

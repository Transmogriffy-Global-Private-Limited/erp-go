param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $TenantID = "00000000-0000-0000-0000-000000000001",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

$PlatformHeaders = @{
  "X-Platform-User-ID" = "00000000-0000-0000-0000-00000000aaaa"
  "X-Platform-Role" = "superadmin"
}

Write-Host "Seeding tenant..."
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") -TenantID $TenantID -EnvFile $EnvFile

$PlanID = "dev-plan-" + (Get-Date -Format "yyyyMMddHHmmss") + "-" + (Get-Random -Minimum 1000 -Maximum 9999)

$PlanBody = @{
  id = $PlanID
  name = "Dev Plan " + $PlanID
  status = "active"
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating plan: $PlanID"
$CreatedPlan = Invoke-RestMethod "$ControlPlaneUrl/control/v1/plans" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $PlatformHeaders `
  -Body $PlanBody

if ($CreatedPlan.plan.id -ne $PlanID) {
  throw "Created plan did not return expected ID."
}

Write-Host "Plan created."

Write-Host ""
Write-Host "Enabling inventory module for plan..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/plans/$PlanID/modules/inventory/enable" `
  -Method Post `
  -Headers $PlatformHeaders | Out-Null

Write-Host "Enabling sales module for plan..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/plans/$PlanID/modules/sales/enable" `
  -Method Post `
  -Headers $PlatformHeaders | Out-Null

Write-Host ""
Write-Host "Checking plan modules..."
$PlanModules = Invoke-RestMethod "$ControlPlaneUrl/control/v1/plans/$PlanID/modules" `
  -Headers $PlatformHeaders

$PlanModuleIDs = @($PlanModules.modules | ForEach-Object { $_.id })

if ($PlanModuleIDs -notcontains "inventory") {
  throw "Plan modules does not contain inventory."
}

if ($PlanModuleIDs -notcontains "sales") {
  throw "Plan modules does not contain sales."
}

Write-Host "Plan modules verified."

$SubscriptionBody = @{
  plan_id = $PlanID
  status = "active"
} | ConvertTo-Json

Write-Host ""
Write-Host "Assigning subscription to tenant..."
$Subscription = Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/subscription" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $PlatformHeaders `
  -Body $SubscriptionBody

if ($Subscription.subscription.plan_id -ne $PlanID) {
  throw "Tenant subscription did not return expected plan_id."
}

Write-Host "Subscription assigned."

Write-Host ""
Write-Host "Checking control-plane tenant modules..."
$TenantModules = Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules" `
  -Headers $PlatformHeaders

$TenantModuleIDs = @($TenantModules.modules | ForEach-Object { $_.id })

if ($TenantModuleIDs -notcontains "inventory") {
  throw "Tenant modules does not contain inventory after subscription assignment."
}

if ($TenantModuleIDs -notcontains "sales") {
  throw "Tenant modules does not contain sales after subscription assignment."
}

Write-Host "Control-plane tenant modules verified."

Write-Host ""
Write-Host "Checking ERP module visibility..."
$ERPModules = Invoke-RestMethod "$BaseUrl/api/v1/modules" `
  -Headers @{ "X-Tenant-ID" = $TenantID }

$ERPModuleIDs = @($ERPModules.modules | ForEach-Object { $_.id })

if ($ERPModuleIDs -notcontains "inventory") {
  throw "ERP modules does not contain inventory after subscription assignment."
}

if ($ERPModuleIDs -notcontains "sales") {
  throw "ERP modules does not contain sales after subscription assignment."
}

Write-Host "ERP module visibility verified."

Write-Host ""
Write-Host "Plans/subscriptions verification passed."

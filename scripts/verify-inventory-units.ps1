param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $TenantID = "00000000-0000-0000-0000-000000000001",
  [string] $AllowedUserID = "11111111-1111-1111-1111-111111111111",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

& (Join-Path $PSScriptRoot "ensure-local-apis.ps1") `
  -BaseUrl $BaseUrl `
  -ControlPlaneUrl $ControlPlaneUrl

$PlatformHeaders = & (Join-Path $PSScriptRoot "Get-PlatformSessionHeaders.ps1") `
  -ControlPlaneUrl $ControlPlaneUrl `
  -EnvFile $EnvFile

Write-Host "Seeding tenant..."
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") -TenantID $TenantID -EnvFile $EnvFile

Write-Host ""
Write-Host "Seeding RBAC/password..."
& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") `
  -TenantID $TenantID `
  -AllowedUserID $AllowedUserID `
  -EnvFile $EnvFile

Write-Host ""
Write-Host "Ensuring inventory module is enabled..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/inventory/enable" `
  -Method Post `
  -Headers $PlatformHeaders | Out-Null

$Headers = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") `
  -BaseUrl $BaseUrl `
  -TenantID $TenantID `
  -UserKind "allowed"

$Code = "UOM" + (Get-Random -Minimum 1000 -Maximum 9999)

$Body = @{
  code = $Code.ToLower()
  name = "Verification Unit $Code"
  description = "Created by inventory unit verification"
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating inventory unit: $Code"
$Created = Invoke-RestMethod "$BaseUrl/api/v1/inventory/units" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $Body

if (-not $Created.unit.id) {
  throw "Create unit did not return unit.id."
}

if ($Created.unit.code -ne $Code) {
  throw "Expected unit code $Code, got $($Created.unit.code)"
}

Write-Host "Inventory unit created."

Write-Host ""
Write-Host "Listing inventory units..."
$List = Invoke-RestMethod "$BaseUrl/api/v1/inventory/units" -Headers $Headers

$CanSee = @($List.units | Where-Object { $_.code -eq $Code }).Count -gt 0
if (-not $CanSee) {
  throw "Created inventory unit not found in list."
}

Write-Host "Inventory unit list verified."

Write-Host ""
Write-Host "Inventory units verification passed."

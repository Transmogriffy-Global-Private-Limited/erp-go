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

& (Join-Path $PSScriptRoot "ensure-local-apis.ps1") `
  -BaseUrl $BaseUrl `
  -ControlPlaneUrl $ControlPlaneUrl

$PlatformHeaders = & (Join-Path $PSScriptRoot "Get-PlatformSessionHeaders.ps1") `
  -ControlPlaneUrl $ControlPlaneUrl `
  -EnvFile $EnvFile

Write-Host "Seeding tenant..."
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") -TenantID $TenantID -EnvFile $EnvFile

Write-Host ""
Write-Host "Seeding RBAC..."
& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") -TenantID $TenantID -AllowedUserID $UserID -EnvFile $EnvFile

$Headers = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") `
  -BaseUrl $BaseUrl `
  -TenantID $TenantID `
  -UserKind "allowed"

Write-Host ""
Write-Host "Ensuring inventory is enabled first..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/inventory/enable" `
  -Method Post `
  -Headers $PlatformHeaders | Out-Null

Write-Host "Checking inventory API while enabled..."
$EnabledResponse = Invoke-WebRequest "$BaseUrl/api/v1/inventory/items" `
  -Headers $Headers `
  -SkipHttpErrorCheck

if ($EnabledResponse.StatusCode -ne 200) {
  throw "Expected inventory API to return 200 while enabled, got HTTP $($EnabledResponse.StatusCode). Body: $($EnabledResponse.Content)"
}

Write-Host "Disabling inventory..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/inventory/disable" `
  -Method Post `
  -Headers $PlatformHeaders | Out-Null

try {
  Write-Host "Checking inventory API while disabled. Expecting 403..."
  $DisabledResponse = Invoke-WebRequest "$BaseUrl/api/v1/inventory/items" `
    -Headers $Headers `
    -SkipHttpErrorCheck

  if ($DisabledResponse.StatusCode -ne 403) {
    throw "Expected 403 while inventory disabled, got HTTP $($DisabledResponse.StatusCode). Body: $($DisabledResponse.Content)"
  }

  Write-Host "Inventory API returned 403 while disabled."
}
finally {
  Write-Host "Re-enabling inventory..."
  Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/inventory/enable" `
    -Method Post `
    -Headers $PlatformHeaders | Out-Null
}

Write-Host "Checking inventory API after re-enable..."
$ReenabledResponse = Invoke-WebRequest "$BaseUrl/api/v1/inventory/items" `
  -Headers $Headers `
  -SkipHttpErrorCheck

if ($ReenabledResponse.StatusCode -ne 200) {
  throw "Expected inventory API to return 200 after re-enable, got HTTP $($ReenabledResponse.StatusCode). Body: $($ReenabledResponse.Content)"
}

Write-Host ""
Write-Host "Module entitlement enforcement verification passed."

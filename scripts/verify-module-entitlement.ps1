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

$PlatformHeaders = @{
  "X-Platform-User-ID" = "00000000-0000-0000-0000-00000000aaaa"
  "X-Platform-Role" = "superadmin"
}

Write-Host "Seeding tenant..."
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") -TenantID $TenantID -EnvFile $EnvFile

Write-Host ""
Write-Host "Seeding RBAC..."
& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") -TenantID $TenantID -AllowedUserID $UserID -EnvFile $EnvFile

$Headers = @{
  "X-Tenant-ID" = $TenantID
  "X-User-ID" = $UserID
}

Write-Host ""
Write-Host "Ensuring inventory is enabled first..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/inventory/enable" -Method Post -Headers $PlatformHeaders | Out-Null

Write-Host "Checking inventory API while enabled..."
Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Headers $Headers | Out-Null

Write-Host "Disabling inventory..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/inventory/disable" -Method Post -Headers $PlatformHeaders | Out-Null

try {
  Write-Host "Checking inventory API while disabled. Expecting 403..."
  Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Headers $Headers | Out-Null
  throw "Inventory API succeeded while module was disabled. Entitlement enforcement failed."
}
catch {
  $statusCode = $null

  if ($_.Exception.Response -and $_.Exception.Response.StatusCode) {
    $statusCode = [int]$_.Exception.Response.StatusCode
  }

  if ($statusCode -ne 403) {
    throw "Expected 403 while inventory disabled, got: $statusCode"
  }

  Write-Host "Inventory API returned 403 while disabled."
}
finally {
  Write-Host "Re-enabling inventory..."
  Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/inventory/enable" -Method Post -Headers $PlatformHeaders | Out-Null
}

Write-Host "Checking inventory API after re-enable..."
Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Headers $Headers | Out-Null

Write-Host ""
Write-Host "Module entitlement enforcement verification passed."

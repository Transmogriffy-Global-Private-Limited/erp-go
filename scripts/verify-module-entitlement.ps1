param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $TenantID = "00000000-0000-0000-0000-000000000001"
)

$ErrorActionPreference = "Stop"

$Headers = @{
  "X-Tenant-ID" = $TenantID
}

Write-Host "Ensuring inventory is enabled first..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/inventory/enable" -Method Post | Out-Null

Write-Host "Checking inventory API while enabled..."
Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Headers $Headers | Out-Null

Write-Host "Disabling inventory..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/inventory/disable" -Method Post | Out-Null

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
  Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/inventory/enable" -Method Post | Out-Null
}

Write-Host "Checking inventory API after re-enable..."
Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Headers $Headers | Out-Null

Write-Host ""
Write-Host "Module entitlement enforcement verification passed."

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

$SessionHeaders = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") `
  -BaseUrl $BaseUrl `
  -TenantID $TenantID `
  -UserKind "allowed"

$Sku = "SESSION-INV-" + (Get-Date -Format "yyyyMMddHHmmss") + "-" + (Get-Random -Minimum 1000 -Maximum 9999)

$Body = @{
  sku = $Sku
  name = "Session Inventory Verification Item"
  description = "Created through X-ERP-Session"
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating inventory item with X-ERP-Session..."
$Created = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $SessionHeaders `
  -Body $Body

if ($Created.item.sku -ne $Sku) {
  throw "Session inventory create returned wrong SKU."
}

Write-Host "Inventory create with X-ERP-Session passed."

Write-Host ""
Write-Host "Listing inventory items with X-ERP-Session..."
$List = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Headers $SessionHeaders

$CanSee = @($List.items | Where-Object { $_.sku -eq $Sku }).Count -gt 0
if (-not $CanSee) {
  throw "Session inventory list cannot see created item."
}

Write-Host "Inventory list with X-ERP-Session passed."

$OldHeaders = @{
  "X-Tenant-ID" = $TenantID
  "X-User-ID" = $AllowedUserID
}

Write-Host ""
Write-Host "Checking old X-User-ID inventory access is rejected..."
try {
  Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Headers $OldHeaders | Out-Null
  throw "Old X-User-ID inventory access still worked."
}
catch {
  $statusCode = $null

  if ($_.Exception.Response -and $_.Exception.Response.StatusCode) {
    $statusCode = [int]$_.Exception.Response.StatusCode
  }

  if ($statusCode -ne 401) {
    throw "Expected old X-User-ID inventory access to return 401, got: $statusCode"
  }

  Write-Host "Old X-User-ID inventory access rejected as expected."
}

Write-Host ""
Write-Host "Inventory ERP session migration verification passed."


param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $TenantID = "00000000-0000-0000-0000-000000000001",
  [string] $AllowedUserID = "11111111-1111-1111-1111-111111111111",
  [string] $NoAccessUserID = "22222222-2222-2222-2222-222222222222",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

& (Join-Path $PSScriptRoot "ensure-local-apis.ps1") `
  -BaseUrl $BaseUrl `
  -ControlPlaneUrl $ControlPlaneUrl `
  -EnvFile $EnvFile

$PlatformHeaders = & (Join-Path $PSScriptRoot "Get-PlatformSessionHeaders.ps1") `
  -ControlPlaneUrl $ControlPlaneUrl `
  -EnvFile $EnvFile

Write-Host "Seeding tenant..."
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") -TenantID $TenantID -EnvFile $EnvFile

Write-Host ""
Write-Host "Seeding RBAC..."
& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") `
  -TenantID $TenantID `
  -AllowedUserID $AllowedUserID `
  -NoAccessUserID $NoAccessUserID `
  -EnvFile $EnvFile

Write-Host ""
Write-Host "Ensuring inventory module is enabled..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/inventory/enable" `
  -Method Post `
  -Headers $PlatformHeaders | Out-Null

$AllowedHeaders = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") `
  -BaseUrl $BaseUrl `
  -TenantID $TenantID `
  -UserKind "allowed"

$NoAccessHeaders = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") `
  -BaseUrl $BaseUrl `
  -TenantID $TenantID `
  -UserKind "noaccess"

$Sku = "RBAC-" + (Get-Date -Format "yyyyMMddHHmmss") + "-" + (Get-Random -Minimum 1000 -Maximum 9999)

$Body = @{
  sku = $Sku
  name = "RBAC Verification Item"
  description = "Created by allowed dev user"
} | ConvertTo-Json

Write-Host ""
$RBACUnitCode = "RBACUOM" + (Get-Random -Minimum 10000 -Maximum 99999)

$RBACUnitBody = @{
  code = $RBACUnitCode.ToLower()
  name = "RBAC Verification Unit $RBACUnitCode"
  description = "Created by RBAC verification"
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating active base unit for RBAC item: $RBACUnitCode"
$RBACCreatedUnit = Invoke-RestMethod "$BaseUrl/api/v1/inventory/units" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $AllowedHeaders `
  -Body $RBACUnitBody

if (-not $RBACCreatedUnit.unit.id) {
  throw "Create RBAC unit did not return unit.id."
}

Write-Host "Adding base_unit_id to RBAC item body..."
$RBACBodyObject = $Body | ConvertFrom-Json
$RBACBodyObject | Add-Member -NotePropertyName "base_unit_id" -NotePropertyValue $RBACCreatedUnit.unit.id -Force
$Body = $RBACBodyObject | ConvertTo-Json

Write-Host "Allowed user creating inventory item..."
$Created = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $AllowedHeaders `
  -Body $Body

if (-not $Created.item -or $Created.item.sku -ne $Sku) {
  throw "Allowed user did not create expected inventory item."
}

Write-Host "Allowed user created item."

Write-Host ""
Write-Host "Allowed user listing inventory items..."
$AllowedList = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Headers $AllowedHeaders
$AllowedCanSee = @($AllowedList.items | Where-Object { $_.sku -eq $Sku }).Count -gt 0

if (-not $AllowedCanSee) {
  throw "Allowed user cannot see created inventory item."
}

Write-Host "Allowed user can list inventory items."

function Expect-Forbidden {
  param(
    [scriptblock] $Action,
    [string] $Label
  )

  try {
    & $Action | Out-Null
    throw "$Label succeeded but should have returned 403."
  }
  catch {
    $statusCode = $null

    if ($_.Exception.Response -and $_.Exception.Response.StatusCode) {
      $statusCode = [int]$_.Exception.Response.StatusCode
    }

    if ($statusCode -ne 403) {
      throw "$Label expected 403, got: $statusCode"
    }

    Write-Host "$Label returned 403 as expected."
  }
}

Write-Host ""
Write-Host "No-access user attempting list..."
Expect-Forbidden -Label "No-access list" -Action {
  Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Headers $NoAccessHeaders
}

Write-Host ""
Write-Host "No-access user attempting create..."
Expect-Forbidden -Label "No-access create" -Action {
  Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" `
    -Method Post `
    -ContentType "application/json" `
    -Headers $NoAccessHeaders `
    -Body $Body
}

Write-Host ""
Write-Host "RBAC verification passed."


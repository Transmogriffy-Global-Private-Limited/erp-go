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

$NoAccessHeaders = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") `
  -BaseUrl $BaseUrl `
  -TenantID $TenantID `
  -UserKind "noaccess"

$Suffix = Get-Random -Minimum 10000 -Maximum 99999

$UnitCode = "STKUOM$Suffix"
$UnitBody = @{
  code = $UnitCode.ToLower()
  name = "Stock Unit $Suffix"
  description = "Created by stock verification"
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating stock verification unit: $UnitCode"
$Unit = Invoke-RestMethod "$BaseUrl/api/v1/inventory/units" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $UnitBody

if (-not $Unit.unit.id) {
  throw "Create unit did not return unit.id."
}

$ItemSku = "STKITEM-$Suffix"
$ItemBody = @{
  sku = $ItemSku
  name = "Stock Verification Item $Suffix"
  description = "Created by stock verification"
  base_unit_id = $Unit.unit.id
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating stock verification item: $ItemSku"
$Item = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $ItemBody

if (-not $Item.item.id) {
  throw "Create item did not return item.id."
}

$LocationCode = "STKLOC$Suffix"
$LocationBody = @{
  code = $LocationCode.ToLower()
  name = "Stock Location $Suffix"
  description = "Created by stock verification"
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating stock verification location: $LocationCode"
$Location = Invoke-RestMethod "$BaseUrl/api/v1/inventory/locations" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $LocationBody

if (-not $Location.location.id) {
  throw "Create location did not return location.id."
}

$MovementBody = @{
  movement_type = "adjustment"
  reference = "stock-verification-$Suffix"
  notes = "Created by inventory stock verification"
  lines = @(
    @{
      item_id = $Item.item.id
      location_id = $Location.location.id
      quantity_delta = "10.000"
    }
  )
} | ConvertTo-Json -Depth 10

Write-Host ""
Write-Host "Creating stock movement..."
$Movement = Invoke-RestMethod "$BaseUrl/api/v1/inventory/stock-movements" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $MovementBody

if (-not $Movement.movement.id) {
  throw "Create stock movement did not return movement.id."
}

if ($Movement.movement.movement_type -ne "adjustment") {
  throw "Expected movement_type adjustment, got $($Movement.movement.movement_type)."
}

if (@($Movement.movement.lines).Count -ne 1) {
  throw "Expected one movement line."
}

if ($Movement.movement.lines[0].quantity_delta -ne "10.000") {
  throw "Expected quantity_delta 10.000, got $($Movement.movement.lines[0].quantity_delta)."
}

Write-Host "Stock movement create verified."

Write-Host ""
Write-Host "Listing stock movements..."
$Movements = Invoke-RestMethod "$BaseUrl/api/v1/inventory/stock-movements" -Headers $Headers
$ListedMovement = @($Movements.movements | Where-Object { $_.id -eq $Movement.movement.id }) | Select-Object -First 1

if (-not $ListedMovement) {
  throw "Created stock movement not found in stock movement list."
}

Write-Host "Stock movement list verified."

Write-Host ""
Write-Host "Listing stock balances..."
$Balances = Invoke-RestMethod "$BaseUrl/api/v1/inventory/stock-balances" -Headers $Headers
$Balance = @($Balances.balances | Where-Object { $_.item_id -eq $Item.item.id -and $_.location_id -eq $Location.location.id }) | Select-Object -First 1

if (-not $Balance) {
  throw "Created stock balance not found."
}

if ($Balance.quantity -ne "10.000") {
  throw "Expected stock balance 10.000, got $($Balance.quantity)."
}

if ($Balance.base_unit_code -ne $UnitCode) {
  throw "Expected base_unit_code $UnitCode, got $($Balance.base_unit_code)."
}

Write-Host "Stock balance verified."

Write-Host ""
Write-Host "Verifying invalid zero quantity is rejected..."
$ZeroMovementBody = @{
  movement_type = "adjustment"
  lines = @(
    @{
      item_id = $Item.item.id
      location_id = $Location.location.id
      quantity_delta = "0"
    }
  )
} | ConvertTo-Json -Depth 10

$ZeroResponse = Invoke-WebRequest "$BaseUrl/api/v1/inventory/stock-movements" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $ZeroMovementBody `
  -SkipHttpErrorCheck

if ($ZeroResponse.StatusCode -ne 400) {
  throw "Expected zero quantity movement to return 400, got $($ZeroResponse.StatusCode)."
}

$ZeroError = $ZeroResponse.Content | ConvertFrom-Json
if ($ZeroError.error.code -ne "invalid_quantity_delta") {
  throw "Expected invalid_quantity_delta, got $($ZeroError.error.code)."
}

Write-Host "Zero quantity rejection verified."

Write-Host ""
Write-Host "Verifying no-access user cannot create stock movement..."
$DeniedResponse = Invoke-WebRequest "$BaseUrl/api/v1/inventory/stock-movements" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $NoAccessHeaders `
  -Body $MovementBody `
  -SkipHttpErrorCheck

if ($DeniedResponse.StatusCode -ne 403) {
  throw "Expected no-access stock movement create to return 403, got $($DeniedResponse.StatusCode)."
}

Write-Host "No-access stock movement create rejection verified."

Write-Host ""
Write-Host "Inventory stock verification passed."

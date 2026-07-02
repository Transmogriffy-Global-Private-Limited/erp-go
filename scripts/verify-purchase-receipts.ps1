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
Write-Host "Ensuring inventory and purchase modules are enabled..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/inventory/enable" `
  -Method Post `
  -Headers $PlatformHeaders | Out-Null

Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/purchase/enable" `
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

$UnitCode = "RCVUOM$Suffix"
$UnitBody = @{
  code = $UnitCode.ToLower()
  name = "Receipt Unit $Suffix"
  description = "Created by purchase receipt verification"
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating receipt verification unit: $UnitCode"
$Unit = Invoke-RestMethod "$BaseUrl/api/v1/inventory/units" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $UnitBody

if (-not $Unit.unit.id) {
  throw "Create unit did not return unit.id."
}

$ItemSku = "RCVITEM-$Suffix"
$ItemBody = @{
  sku = $ItemSku
  name = "Receipt Verification Item $Suffix"
  description = "Created by purchase receipt verification"
  base_unit_id = $Unit.unit.id
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating receipt verification item: $ItemSku"
$Item = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $ItemBody

if (-not $Item.item.id) {
  throw "Create item did not return item.id."
}

$LocationCode = "RCVLOC$Suffix"
$LocationBody = @{
  code = $LocationCode.ToLower()
  name = "Receipt Location $Suffix"
  description = "Created by purchase receipt verification"
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating receipt verification location: $LocationCode"
$Location = Invoke-RestMethod "$BaseUrl/api/v1/inventory/locations" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $LocationBody

if (-not $Location.location.id) {
  throw "Create location did not return location.id."
}

$ReceiptBody = @{
  supplier_name = "Verification Supplier $Suffix"
  reference = "supplier-ref-$Suffix"
  notes = "Created by purchase receipt verification"
  lines = @(
    @{
      item_id = $Item.item.id
      location_id = $Location.location.id
      quantity_received = "7.000"
    }
  )
} | ConvertTo-Json -Depth 10

Write-Host ""
Write-Host "Creating purchase receipt..."
$Receipt = Invoke-RestMethod "$BaseUrl/api/v1/purchase/receipts" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $ReceiptBody

if (-not $Receipt.receipt.id) {
  throw "Create purchase receipt did not return receipt.id."
}

if (-not $Receipt.receipt.stock_movement_id) {
  throw "Create purchase receipt did not return stock_movement_id."
}

if (@($Receipt.receipt.lines).Count -ne 1) {
  throw "Expected one purchase receipt line."
}

if ($Receipt.receipt.lines[0].quantity_received -ne "7.000") {
  throw "Expected quantity_received 7.000, got $($Receipt.receipt.lines[0].quantity_received)."
}

Write-Host "Purchase receipt create verified."

Write-Host ""
Write-Host "Listing purchase receipts..."
$Receipts = Invoke-RestMethod "$BaseUrl/api/v1/purchase/receipts" -Headers $Headers
$ListedReceipt = @($Receipts.receipts | Where-Object { $_.id -eq $Receipt.receipt.id }) | Select-Object -First 1

if (-not $ListedReceipt) {
  throw "Created purchase receipt not found in list."
}

Write-Host "Purchase receipt list verified."

Write-Host ""
Write-Host "Checking stock movement exists..."
$Movements = Invoke-RestMethod "$BaseUrl/api/v1/inventory/stock-movements" -Headers $Headers
$Movement = @($Movements.movements | Where-Object { $_.id -eq $Receipt.receipt.stock_movement_id }) | Select-Object -First 1

if (-not $Movement) {
  throw "Purchase receipt stock movement not found."
}

if ($Movement.movement_type -ne "receipt") {
  throw "Expected receipt stock movement type, got $($Movement.movement_type)."
}

if (@($Movement.lines).Count -ne 1) {
  throw "Expected receipt stock movement to have one line."
}

if ($Movement.lines[0].quantity_delta -ne "7.000") {
  throw "Expected stock movement quantity_delta 7.000, got $($Movement.lines[0].quantity_delta)."
}

Write-Host "Purchase receipt stock movement verified."

Write-Host ""
Write-Host "Checking stock balance increased..."
$Balances = Invoke-RestMethod "$BaseUrl/api/v1/inventory/stock-balances" -Headers $Headers
$Balance = @($Balances.balances | Where-Object { $_.item_id -eq $Item.item.id -and $_.location_id -eq $Location.location.id }) | Select-Object -First 1

if (-not $Balance) {
  throw "Purchase receipt stock balance not found."
}

if ($Balance.quantity -ne "7.000") {
  throw "Expected stock balance 7.000, got $($Balance.quantity)."
}

Write-Host "Purchase receipt stock balance verified."

Write-Host ""
Write-Host "Verifying invalid zero quantity is rejected..."
$ZeroReceiptBody = @{
  supplier_name = "Zero Supplier $Suffix"
  lines = @(
    @{
      item_id = $Item.item.id
      location_id = $Location.location.id
      quantity_received = "0"
    }
  )
} | ConvertTo-Json -Depth 10

$ZeroResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/receipts" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $ZeroReceiptBody `
  -SkipHttpErrorCheck

if ($ZeroResponse.StatusCode -ne 400) {
  throw "Expected zero receipt quantity to return 400, got $($ZeroResponse.StatusCode)."
}

$ZeroError = $ZeroResponse.Content | ConvertFrom-Json
if ($ZeroError.error.code -ne "invalid_quantity_received") {
  throw "Expected invalid_quantity_received, got $($ZeroError.error.code)."
}

Write-Host "Zero receipt quantity rejection verified."

Write-Host ""
Write-Host "Verifying no-access user cannot create purchase receipt..."
$DeniedResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/receipts" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $NoAccessHeaders `
  -Body $ReceiptBody `
  -SkipHttpErrorCheck

if ($DeniedResponse.StatusCode -ne 403) {
  throw "Expected no-access purchase receipt create to return 403, got $($DeniedResponse.StatusCode)."
}

Write-Host "No-access purchase receipt create rejection verified."

Write-Host ""
Write-Host "Purchase receipts verification passed."

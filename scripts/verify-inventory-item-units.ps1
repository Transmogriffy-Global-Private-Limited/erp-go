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
  -ControlPlaneUrl $ControlPlaneUrl `
  -EnvFile $EnvFile

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

$Suffix = Get-Random -Minimum 10000 -Maximum 99999
$UnitCode = "EA$Suffix"
$InactiveUnitCode = "OLD$Suffix"
$Sku = "ITEM-UOM-$Suffix"

$UnitBody = @{
  code = $UnitCode.ToLower()
  name = "Each $Suffix"
  description = "Active base unit for item/unit verification"
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating active base unit: $UnitCode"
$CreatedUnit = Invoke-RestMethod "$BaseUrl/api/v1/inventory/units" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $UnitBody

if (-not $CreatedUnit.unit.id) {
  throw "Create unit did not return unit.id."
}

$ItemBody = @{
  sku = $Sku
  name = "Verified Item $Suffix"
  description = "Created by inventory item/unit verification"
  base_unit_id = $CreatedUnit.unit.id
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating inventory item linked to base unit..."
$CreatedItem = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $ItemBody

if (-not $CreatedItem.item.id) {
  throw "Create item did not return item.id."
}

if ($CreatedItem.item.base_unit_id -ne $CreatedUnit.unit.id) {
  throw "Expected item base_unit_id $($CreatedUnit.unit.id), got $($CreatedItem.item.base_unit_id)."
}

if ($CreatedItem.item.base_unit.code -ne $UnitCode) {
  throw "Expected item base_unit.code $UnitCode, got $($CreatedItem.item.base_unit.code)."
}

Write-Host "Inventory item/base unit create verified."

Write-Host ""
Write-Host "Listing inventory items and checking base unit projection..."
$List = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Headers $Headers
$ListedItem = @($List.items | Where-Object { $_.sku -eq $Sku }) | Select-Object -First 1

if (-not $ListedItem) {
  throw "Created inventory item not found in list."
}

if ($ListedItem.base_unit_id -ne $CreatedUnit.unit.id) {
  throw "Listed item has wrong base_unit_id."
}

if ($ListedItem.base_unit.code -ne $UnitCode) {
  throw "Listed item has wrong base_unit.code."
}

Write-Host "Inventory item/base unit list projection verified."

Write-Host ""
Write-Host "Verifying item create rejects missing base_unit_id..."
$MissingBaseUnitBody = @{
  sku = "ITEM-NO-UOM-$Suffix"
  name = "No UOM Item $Suffix"
} | ConvertTo-Json

$MissingBaseUnitResponse = Invoke-WebRequest "$BaseUrl/api/v1/inventory/items" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $MissingBaseUnitBody `
  -SkipHttpErrorCheck

if ($MissingBaseUnitResponse.StatusCode -ne 400) {
  throw "Expected missing base_unit_id to return 400, got $($MissingBaseUnitResponse.StatusCode)."
}

$MissingBaseUnitError = $MissingBaseUnitResponse.Content | ConvertFrom-Json
if ($MissingBaseUnitError.error.code -ne "base_unit_required") {
  throw "Expected base_unit_required error, got $($MissingBaseUnitError.error.code)."
}

Write-Host "Missing base_unit_id rejection verified."

$InactiveUnitBody = @{
  code = $InactiveUnitCode.ToLower()
  name = "Inactive Unit $Suffix"
  description = "Inactive unit for negative verification"
  status = "inactive"
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating inactive unit and verifying it cannot be used as a base unit..."
$InactiveUnit = Invoke-RestMethod "$BaseUrl/api/v1/inventory/units" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $InactiveUnitBody

$InactiveItemBody = @{
  sku = "ITEM-INACTIVE-UOM-$Suffix"
  name = "Inactive UOM Item $Suffix"
  base_unit_id = $InactiveUnit.unit.id
} | ConvertTo-Json

$InactiveResponse = Invoke-WebRequest "$BaseUrl/api/v1/inventory/items" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $InactiveItemBody `
  -SkipHttpErrorCheck

if ($InactiveResponse.StatusCode -ne 400) {
  throw "Expected inactive base unit to return 400, got $($InactiveResponse.StatusCode)."
}

$InactiveError = $InactiveResponse.Content | ConvertFrom-Json
if ($InactiveError.error.code -ne "base_unit_invalid") {
  throw "Expected base_unit_invalid error, got $($InactiveError.error.code)."
}

Write-Host "Inactive base unit rejection verified."

Write-Host ""
Write-Host "Inventory item/base unit verification passed."

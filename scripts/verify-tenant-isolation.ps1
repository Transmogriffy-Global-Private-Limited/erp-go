param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $TenantAID = "00000000-0000-0000-0000-000000000001",
  [string] $TenantBID = "00000000-0000-0000-0000-000000000002",
  [string] $TenantAAllowedUserID = "11111111-1111-1111-1111-111111111111",
  [string] $TenantBAllowedUserID = "55555555-5555-5555-5555-555555555555",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

& (Join-Path $PSScriptRoot "ensure-local-apis.ps1") `
  -BaseUrl $BaseUrl `
  -ControlPlaneUrl $ControlPlaneUrl

Write-Host "Seeding tenant A..."
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") `
  -TenantID $TenantAID `
  -Slug "dev-tenant-a" `
  -LegalName "Dev Tenant A Private Limited" `
  -DisplayName "Dev Tenant A" `
  -EnvFile $EnvFile

Write-Host ""
Write-Host "Seeding tenant A RBAC..."
& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") `
  -TenantID $TenantAID `
  -AllowedUserID $TenantAAllowedUserID `
  -EnvFile $EnvFile

Write-Host ""
Write-Host "Seeding tenant B..."
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") `
  -TenantID $TenantBID `
  -Slug "dev-tenant-b" `
  -LegalName "Dev Tenant B Private Limited" `
  -DisplayName "Dev Tenant B" `
  -EnvFile $EnvFile

Write-Host ""
Write-Host "Seeding tenant B RBAC..."
& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") `
  -TenantID $TenantBID `
  -AllowedUserID $TenantBAllowedUserID `
  -AllowedRoleID "77777777-7777-7777-7777-777777777777" `
  -NoAccessUserID "66666666-6666-6666-6666-666666666666" `
  -NoAccessRoleID "88888888-8888-8888-8888-888888888888" `
  -EnvFile $EnvFile

$HeadersA = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") `
  -BaseUrl $BaseUrl `
  -TenantID $TenantAID `
  -UserKind "allowed"

$HeadersB = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") `
  -BaseUrl $BaseUrl `
  -TenantID $TenantBID `
  -UserKind "allowed"

$Sku = "RLS-" + (Get-Date -Format "yyyyMMddHHmmss") + "-" + (Get-Random -Minimum 1000 -Maximum 9999)

$Body = @{
  sku = $Sku
  name = "Tenant Isolation Verification Item"
  description = "Created under tenant A"
} | ConvertTo-Json

Write-Host ""
$TenantIsolationUnitCode = "TISO-UOM" + (Get-Random -Minimum 10000 -Maximum 99999)

$TenantIsolationUnitBody = @{
  code = $TenantIsolationUnitCode.ToLower()
  name = "Tenant Isolation Unit $TenantIsolationUnitCode"
  description = "Created by tenant isolation verification"
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating active base unit for tenant-isolation item: $TenantIsolationUnitCode"
$TenantIsolationCreatedUnit = Invoke-RestMethod "$BaseUrl/api/v1/inventory/units" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $HeadersA `
  -Body $TenantIsolationUnitBody

if (-not $TenantIsolationCreatedUnit.unit.id) {
  throw "Create tenant-isolation unit did not return unit.id."
}

Write-Host "Adding base_unit_id to tenant-isolation item body..."
$TenantIsolationBodyObject = $Body | ConvertFrom-Json
$TenantIsolationBodyObject | Add-Member -NotePropertyName "base_unit_id" -NotePropertyValue $TenantIsolationCreatedUnit.unit.id -Force
$Body = $TenantIsolationBodyObject | ConvertTo-Json

Write-Host "Creating item under tenant A with SKU: $Sku"
Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $HeadersA `
  -Body $Body | Out-Null

Write-Host ""
Write-Host "Listing tenant A items..."
$TenantAItems = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Headers $HeadersA
$TenantACanSee = @($TenantAItems.items | Where-Object { $_.sku -eq $Sku }).Count -gt 0

if (-not $TenantACanSee) {
  throw "Tenant A cannot see its own item."
}

Write-Host "Tenant A can see its own item."

Write-Host ""
Write-Host "Listing tenant B items..."
$TenantBItems = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Headers $HeadersB
$TenantBCanSee = @($TenantBItems.items | Where-Object { $_.sku -eq $Sku }).Count -gt 0

if ($TenantBCanSee) {
  throw "Tenant B can see Tenant A item. Tenant isolation failed."
}

Write-Host "Tenant B cannot see Tenant A item."

Write-Host ""
Write-Host "Tenant isolation verification passed."


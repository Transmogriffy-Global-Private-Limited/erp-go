param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $TenantA = "00000000-0000-0000-0000-000000000001",
  [string] $TenantB = "00000000-0000-0000-0000-000000000002",
  [string] $TenantAUserID = "11111111-1111-1111-1111-111111111111",
  [string] $TenantBUserID = "55555555-5555-5555-5555-555555555555",
  [string] $TenantBNoAccessUserID = "66666666-6666-6666-6666-666666666666",
  [string] $TenantBRoleID = "77777777-7777-7777-7777-777777777777",
  [string] $TenantBNoAccessRoleID = "88888888-8888-8888-8888-888888888888",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

Write-Host "Seeding tenant A..."
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") `
  -TenantID $TenantA `
  -Slug "dev-tenant-a" `
  -LegalName "Dev Tenant A Private Limited" `
  -DisplayName "Dev Tenant A" `
  -EnvFile $EnvFile

Write-Host ""
Write-Host "Seeding tenant A RBAC..."
& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") `
  -TenantID $TenantA `
  -AllowedUserID $TenantAUserID `
  -EnvFile $EnvFile

Write-Host ""
Write-Host "Seeding tenant B..."
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") `
  -TenantID $TenantB `
  -Slug "dev-tenant-b" `
  -LegalName "Dev Tenant B Private Limited" `
  -DisplayName "Dev Tenant B" `
  -EnvFile $EnvFile

Write-Host ""
Write-Host "Seeding tenant B RBAC..."
& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") `
  -TenantID $TenantB `
  -AllowedUserID $TenantBUserID `
  -NoAccessUserID $TenantBNoAccessUserID `
  -AllowedRoleID $TenantBRoleID `
  -NoAccessRoleID $TenantBNoAccessRoleID `
  -EnvFile $EnvFile

$Sku = "RLS-" + (Get-Date -Format "yyyyMMddHHmmss") + "-" + (Get-Random -Minimum 1000 -Maximum 9999)

$Body = @{
  sku = $Sku
  name = "RLS Isolation Test Item"
  description = "Created under tenant A only"
} | ConvertTo-Json

$TenantAHeaders = @{
  "X-Tenant-ID" = $TenantA
  "X-User-ID" = $TenantAUserID
}

$TenantBHeaders = @{
  "X-Tenant-ID" = $TenantB
  "X-User-ID" = $TenantBUserID
}

Write-Host ""
Write-Host "Creating item under tenant A with SKU: $Sku"
$Created = Invoke-RestMethod `
  -Uri "$BaseUrl/api/v1/inventory/items" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $TenantAHeaders `
  -Body $Body

if (-not $Created.item -or $Created.item.sku -ne $Sku) {
  throw "Tenant A item creation did not return the expected SKU."
}

Write-Host ""
Write-Host "Listing tenant A items..."
$TenantAList = Invoke-RestMethod `
  -Uri "$BaseUrl/api/v1/inventory/items" `
  -Headers $TenantAHeaders

$TenantAHasItem = @($TenantAList.items | Where-Object { $_.sku -eq $Sku }).Count -gt 0

if (-not $TenantAHasItem) {
  throw "Tenant A cannot see its own created item. Isolation check failed."
}

Write-Host "Tenant A can see its own item."

Write-Host ""
Write-Host "Listing tenant B items..."
$TenantBList = Invoke-RestMethod `
  -Uri "$BaseUrl/api/v1/inventory/items" `
  -Headers $TenantBHeaders

$TenantBHasItem = @($TenantBList.items | Where-Object { $_.sku -eq $Sku }).Count -gt 0

if ($TenantBHasItem) {
  throw "Tenant B can see Tenant A item. Tenant isolation is broken."
}

Write-Host "Tenant B cannot see Tenant A item."

Write-Host ""
Write-Host "Tenant isolation verification passed."

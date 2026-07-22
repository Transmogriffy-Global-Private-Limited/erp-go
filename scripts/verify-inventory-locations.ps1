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

$AllowedHeaders = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") `
  -BaseUrl $BaseUrl `
  -TenantID $TenantID `
  -UserKind "allowed"

$NoAccessHeaders = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") `
  -BaseUrl $BaseUrl `
  -TenantID $TenantID `
  -UserKind "noaccess"

$Suffix = Get-Random -Minimum 10000 -Maximum 99999
$Code = "LOC$Suffix"

$Body = @{
  code = $Code.ToLower()
  name = "Verification Location $Code"
  description = "Created by inventory location verification"
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating inventory location: $Code"
$Created = Invoke-RestMethod "$BaseUrl/api/v1/inventory/locations" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $AllowedHeaders `
  -Body $Body

if (-not $Created.location.id) {
  throw "Create location did not return location.id."
}

if ($Created.location.code -ne $Code) {
  throw "Expected location code $Code, got $($Created.location.code)."
}

if ($Created.location.status -ne "active") {
  throw "Expected location status active, got $($Created.location.status)."
}

Write-Host "Inventory location created."

Write-Host ""
Write-Host "Listing inventory locations..."
$List = Invoke-RestMethod "$BaseUrl/api/v1/inventory/locations" -Headers $AllowedHeaders

$ListedLocation = @($List.locations | Where-Object { $_.code -eq $Code }) | Select-Object -First 1
if (-not $ListedLocation) {
  throw "Created inventory location not found in list."
}

Write-Host "Inventory location list verified."

Write-Host ""
Write-Host "Verifying missing code is rejected..."
$MissingCodeBody = @{
  name = "Missing Code Location $Suffix"
} | ConvertTo-Json

$MissingCodeResponse = Invoke-WebRequest "$BaseUrl/api/v1/inventory/locations" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $AllowedHeaders `
  -Body $MissingCodeBody `
  -SkipHttpErrorCheck

if ($MissingCodeResponse.StatusCode -ne 400) {
  throw "Expected missing code to return 400, got $($MissingCodeResponse.StatusCode)."
}

$MissingCodeError = $MissingCodeResponse.Content | ConvertFrom-Json
if ($MissingCodeError.error.code -ne "code_required") {
  throw "Expected code_required error, got $($MissingCodeError.error.code)."
}

Write-Host "Missing code rejection verified."

Write-Host ""
Write-Host "Verifying no-access user cannot create inventory location..."
$DeniedCode = "DENYLOC$Suffix"
$DeniedBody = @{
  code = $DeniedCode.ToLower()
  name = "Denied Location $Suffix"
} | ConvertTo-Json

$DeniedResponse = Invoke-WebRequest "$BaseUrl/api/v1/inventory/locations" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $NoAccessHeaders `
  -Body $DeniedBody `
  -SkipHttpErrorCheck

if ($DeniedResponse.StatusCode -ne 403) {
  throw "Expected no-access create to return 403, got $($DeniedResponse.StatusCode)."
}

Write-Host "No-access create rejection verified."

Write-Host ""
Write-Host "Inventory locations verification passed."

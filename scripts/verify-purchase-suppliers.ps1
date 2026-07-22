param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $TenantAID = "00000000-0000-0000-0000-000000000001",
  [string] $TenantBID = "00000000-0000-0000-0000-000000000002",
  [string] $TenantAAllowedUserID = "11111111-1111-1111-1111-111111111111",
  [string] $TenantBAllowedUserID = "55555555-5555-5555-5555-555555555555",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

. (Join-Path $PSScriptRoot "Import-DotEnv.ps1") -Path (Join-Path $RepoRoot $EnvFile)

if (-not $env:MIGRATION_DATABASE_URL) {
  throw "MIGRATION_DATABASE_URL is missing. Copy .env.example to .env and edit it."
}

function Invoke-ScalarSql {
  param(
    [string] $Sql
  )

  $Output = @(psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -At -c $Sql)
  if ($LASTEXITCODE -ne 0) {
    throw "SQL query failed."
  }

  return (($Output | Select-Object -Last 1) -join "").Trim()
}

& (Join-Path $PSScriptRoot "ensure-local-apis.ps1") `
  -BaseUrl $BaseUrl `
  -ControlPlaneUrl $ControlPlaneUrl `
  -EnvFile $EnvFile

$PlatformHeaders = & (Join-Path $PSScriptRoot "Get-PlatformSessionHeaders.ps1") `
  -ControlPlaneUrl $ControlPlaneUrl `
  -EnvFile $EnvFile

Write-Host "Seeding tenant A..."
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") `
  -TenantID $TenantAID `
  -Slug "dev-tenant-a" `
  -LegalName "Dev Tenant A Private Limited" `
  -DisplayName "Dev Tenant A" `
  -EnvFile $EnvFile

Write-Host ""
Write-Host "Seeding tenant A RBAC/password..."
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
Write-Host "Seeding tenant B RBAC/password..."
& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") `
  -TenantID $TenantBID `
  -AllowedUserID $TenantBAllowedUserID `
  -AllowedRoleID "77777777-7777-7777-7777-777777777777" `
  -NoAccessUserID "66666666-6666-6666-6666-666666666666" `
  -NoAccessRoleID "88888888-8888-8888-8888-888888888888" `
  -EnvFile $EnvFile

Write-Host ""
Write-Host "Ensuring purchase module is enabled for both tenants..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantAID/modules/purchase/enable" `
  -Method Post `
  -Headers $PlatformHeaders | Out-Null

Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantBID/modules/purchase/enable" `
  -Method Post `
  -Headers $PlatformHeaders | Out-Null

$TenantAHeaders = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") `
  -BaseUrl $BaseUrl `
  -TenantID $TenantAID `
  -UserKind "allowed"

$TenantANoAccessHeaders = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") `
  -BaseUrl $BaseUrl `
  -TenantID $TenantAID `
  -UserKind "noaccess"

$TenantBHeaders = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") `
  -BaseUrl $BaseUrl `
  -TenantID $TenantBID `
  -UserKind "allowed"

$Suffix = Get-Random -Minimum 10000 -Maximum 99999
$Code = "SUP$Suffix"

$TenantABody = @{
  code = $Code.ToLower()
  name = "Tenant A Verification Supplier $Suffix"
  email = "supplier-$Suffix@example.test"
  phone = "+91-$Suffix"
  address = "Tenant A verification address"
} | ConvertTo-Json

Write-Host ""
Write-Host "Creating tenant A purchase supplier: $Code"
$Created = Invoke-RestMethod "$BaseUrl/api/v1/purchase/suppliers" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $TenantAHeaders `
  -Body $TenantABody

if (-not $Created.supplier.id) {
  throw "Create supplier did not return supplier.id."
}

if ($Created.supplier.code -ne $Code) {
  throw "Expected normalized supplier code $Code, got $($Created.supplier.code)."
}

if ($Created.supplier.status -ne "active") {
  throw "Expected supplier status active, got $($Created.supplier.status)."
}

Write-Host "Purchase supplier creation verified."

Write-Host ""
Write-Host "Listing tenant A purchase suppliers..."
$TenantAList = Invoke-RestMethod "$BaseUrl/api/v1/purchase/suppliers" -Headers $TenantAHeaders
$TenantASupplier = @($TenantAList.suppliers | Where-Object { $_.id -eq $Created.supplier.id }) | Select-Object -First 1

if (-not $TenantASupplier) {
  throw "Created supplier was not found in tenant A list."
}

Write-Host "Tenant A supplier list verified."

Write-Host ""
Write-Host "Verifying tenant B cannot see tenant A supplier..."
$TenantBList = Invoke-RestMethod "$BaseUrl/api/v1/purchase/suppliers" -Headers $TenantBHeaders
$TenantBCanSeeTenantA = @($TenantBList.suppliers | Where-Object { $_.id -eq $Created.supplier.id }).Count -gt 0

if ($TenantBCanSeeTenantA) {
  throw "Tenant B can see tenant A supplier."
}

Write-Host "Tenant supplier isolation verified."

Write-Host ""
Write-Host "Verifying supplier codes are unique per tenant, not globally..."
$TenantBBody = @{
  code = $Code.ToLower()
  name = "Tenant B Verification Supplier $Suffix"
} | ConvertTo-Json

$TenantBCreated = Invoke-RestMethod "$BaseUrl/api/v1/purchase/suppliers" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $TenantBHeaders `
  -Body $TenantBBody

if (-not $TenantBCreated.supplier.id) {
  throw "Tenant B could not create the same tenant-scoped supplier code."
}

Write-Host "Tenant-scoped supplier code uniqueness verified."

Write-Host ""
Write-Host "Verifying duplicate supplier code is rejected within tenant A..."
$DuplicateResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/suppliers" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $TenantAHeaders `
  -Body $TenantABody `
  -SkipHttpErrorCheck

if ($DuplicateResponse.StatusCode -ne 409) {
  throw "Expected duplicate supplier code to return 409, got $($DuplicateResponse.StatusCode)."
}

$DuplicateError = $DuplicateResponse.Content | ConvertFrom-Json
if ($DuplicateError.error.code -ne "supplier_code_exists") {
  throw "Expected supplier_code_exists, got $($DuplicateError.error.code)."
}

Write-Host "Duplicate supplier code rejection verified."

Write-Host ""
Write-Host "Verifying missing supplier code is rejected..."
$MissingCodeBody = @{
  name = "Missing Code Supplier $Suffix"
} | ConvertTo-Json

$MissingCodeResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/suppliers" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $TenantAHeaders `
  -Body $MissingCodeBody `
  -SkipHttpErrorCheck

if ($MissingCodeResponse.StatusCode -ne 400) {
  throw "Expected missing supplier code to return 400, got $($MissingCodeResponse.StatusCode)."
}

$MissingCodeError = $MissingCodeResponse.Content | ConvertFrom-Json
if ($MissingCodeError.error.code -ne "code_required") {
  throw "Expected code_required, got $($MissingCodeError.error.code)."
}

Write-Host "Missing supplier code rejection verified."

Write-Host ""
Write-Host "Verifying no-access user cannot list or create suppliers..."
$DeniedListResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/suppliers" `
  -Headers $TenantANoAccessHeaders `
  -SkipHttpErrorCheck

if ($DeniedListResponse.StatusCode -ne 403) {
  throw "Expected no-access supplier list to return 403, got $($DeniedListResponse.StatusCode)."
}

$DeniedCreateResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/suppliers" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $TenantANoAccessHeaders `
  -Body $TenantABody `
  -SkipHttpErrorCheck

if ($DeniedCreateResponse.StatusCode -ne 403) {
  throw "Expected no-access supplier create to return 403, got $($DeniedCreateResponse.StatusCode)."
}

Write-Host "Supplier RBAC rejection verified."

Write-Host ""
Write-Host "Verifying supplier creation audit and outbox records..."
$SupplierID = $Created.supplier.id
$AuditSql = @"
SELECT set_config('app.tenant_id', '$TenantAID', false);
SELECT count(*)
FROM audit.audit_log
WHERE tenant_id = '$TenantAID'
  AND action = 'purchase.supplier.create'
  AND target_type = 'purchase.supplier'
  AND target_id = '$SupplierID';
"@

if ((Invoke-ScalarSql -Sql $AuditSql) -ne "1") {
  throw "Expected exactly one audit row for supplier $SupplierID."
}

$OutboxSql = @"
SELECT count(*)
FROM core.outbox_events
WHERE tenant_id = '$TenantAID'
  AND event_type = 'purchase.supplier.created.v1'
  AND aggregate_type = 'purchase.supplier'
  AND aggregate_id = '$SupplierID';
"@

if ((Invoke-ScalarSql -Sql $OutboxSql) -ne "1") {
  throw "Expected exactly one outbox row for supplier $SupplierID."
}

Write-Host "Supplier audit/outbox records verified."

Write-Host ""
Write-Host "Verifying purchase module entitlement guard..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantAID/modules/purchase/disable" `
  -Method Post `
  -Headers $PlatformHeaders | Out-Null

try {
  $DisabledResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/suppliers" `
    -Headers $TenantAHeaders `
    -SkipHttpErrorCheck

  if ($DisabledResponse.StatusCode -ne 403) {
    throw "Expected disabled Purchase module to return 403, got $($DisabledResponse.StatusCode)."
  }

  $DisabledError = $DisabledResponse.Content | ConvertFrom-Json
  if ($DisabledError.error.code -ne "module_not_enabled") {
    throw "Expected module_not_enabled, got $($DisabledError.error.code)."
  }
}
finally {
  Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantAID/modules/purchase/enable" `
    -Method Post `
    -Headers $PlatformHeaders | Out-Null
}

Write-Host "Purchase module entitlement guard verified."

Write-Host ""
Write-Host "Purchase suppliers verification passed."

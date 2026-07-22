param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $TenantAID = "00000000-0000-0000-0000-000000000001",
  [string] $TenantBID = "00000000-0000-0000-0000-000000000002",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"
$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

. (Join-Path $PSScriptRoot "Import-DotEnv.ps1") -Path (Join-Path $RepoRoot $EnvFile)
if (-not $env:MIGRATION_DATABASE_URL) {
  throw "MIGRATION_DATABASE_URL is missing."
}

function Invoke-ScalarSql {
  param([string] $Sql)
  $Output = @(psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -At -c $Sql)
  if ($LASTEXITCODE -ne 0) { throw "SQL query failed." }
  return (($Output | Select-Object -Last 1) -join "").Trim()
}

& (Join-Path $PSScriptRoot "ensure-local-apis.ps1") -BaseUrl $BaseUrl -ControlPlaneUrl $ControlPlaneUrl -EnvFile $EnvFile
$PlatformHeaders = & (Join-Path $PSScriptRoot "Get-PlatformSessionHeaders.ps1") -ControlPlaneUrl $ControlPlaneUrl -EnvFile $EnvFile

Write-Host "Seeding tenants and RBAC..."
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") -TenantID $TenantAID -Slug "dev-tenant-a" -EnvFile $EnvFile
& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") -TenantID $TenantAID -EnvFile $EnvFile
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") -TenantID $TenantBID -Slug "dev-tenant-b" -EnvFile $EnvFile
& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") `
  -TenantID $TenantBID `
  -AllowedUserID "55555555-5555-5555-5555-555555555555" `
  -AllowedRoleID "77777777-7777-7777-7777-777777777777" `
  -NoAccessUserID "66666666-6666-6666-6666-666666666666" `
  -NoAccessRoleID "88888888-8888-8888-8888-888888888888" `
  -EnvFile $EnvFile

foreach ($TenantID in @($TenantAID, $TenantBID)) {
  Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/sales/enable" -Method Post -Headers $PlatformHeaders | Out-Null
}

$HeadersA = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") -BaseUrl $BaseUrl -TenantID $TenantAID -UserKind "allowed"
$NoAccessHeaders = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") -BaseUrl $BaseUrl -TenantID $TenantAID -UserKind "noaccess"
$HeadersB = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") -BaseUrl $BaseUrl -TenantID $TenantBID -UserKind "allowed"

$Suffix = Get-Random -Minimum 10000 -Maximum 99999
$Code = "CUS$Suffix"
$Body = @{
  code = $Code.ToLower()
  name = "Tenant A Verification Customer $Suffix"
  email = "customer-$Suffix@example.test"
  phone = "+91-$Suffix"
  billing_address = "Tenant A billing address"
  shipping_address = "Tenant A shipping address"
} | ConvertTo-Json

Write-Host "Creating tenant A sales customer: $Code"
$Created = Invoke-RestMethod "$BaseUrl/api/v1/sales/customers" -Method Post -ContentType "application/json" -Headers $HeadersA -Body $Body
if (-not $Created.customer.id) { throw "Create customer did not return customer.id." }
if ($Created.customer.code -ne $Code) { throw "Expected normalized customer code $Code." }
if ($Created.customer.status -ne "active") { throw "Expected active customer status." }

Write-Host "Listing tenant A sales customers..."
$ListA = Invoke-RestMethod "$BaseUrl/api/v1/sales/customers" -Headers $HeadersA
if (-not (@($ListA.customers | Where-Object { $_.id -eq $Created.customer.id }) | Select-Object -First 1)) {
  throw "Created customer was not found in tenant A list."
}

Write-Host "Verifying tenant isolation..."
$ListB = Invoke-RestMethod "$BaseUrl/api/v1/sales/customers" -Headers $HeadersB
if (@($ListB.customers | Where-Object { $_.id -eq $Created.customer.id }).Count -gt 0) {
  throw "Tenant B can see tenant A customer."
}

Write-Host "Verifying customer codes are tenant-scoped..."
$CreatedB = Invoke-RestMethod "$BaseUrl/api/v1/sales/customers" -Method Post -ContentType "application/json" -Headers $HeadersB -Body (@{
  code = $Code.ToLower()
  name = "Tenant B Verification Customer $Suffix"
} | ConvertTo-Json)
if (-not $CreatedB.customer.id) { throw "Tenant B could not reuse tenant-scoped customer code." }

Write-Host "Verifying duplicate customer code conflict..."
$DuplicateResponse = Invoke-WebRequest "$BaseUrl/api/v1/sales/customers" -Method Post -ContentType "application/json" -Headers $HeadersA -Body $Body -SkipHttpErrorCheck
if ($DuplicateResponse.StatusCode -ne 409) { throw "Expected duplicate customer code to return 409." }
$DuplicateError = $DuplicateResponse.Content | ConvertFrom-Json
if ($DuplicateError.error.code -ne "customer_code_exists") { throw "Expected customer_code_exists." }

Write-Host "Verifying required fields..."
$MissingCodeResponse = Invoke-WebRequest "$BaseUrl/api/v1/sales/customers" -Method Post -ContentType "application/json" -Headers $HeadersA -Body (@{
  name = "Missing Code Customer"
} | ConvertTo-Json) -SkipHttpErrorCheck
if ($MissingCodeResponse.StatusCode -ne 400) { throw "Expected missing code to return 400." }

Write-Host "Verifying no-access RBAC..."
$DeniedList = Invoke-WebRequest "$BaseUrl/api/v1/sales/customers" -Headers $NoAccessHeaders -SkipHttpErrorCheck
if ($DeniedList.StatusCode -ne 403) { throw "Expected no-access customer list to return 403." }
$DeniedCreate = Invoke-WebRequest "$BaseUrl/api/v1/sales/customers" -Method Post -ContentType "application/json" -Headers $NoAccessHeaders -Body $Body -SkipHttpErrorCheck
if ($DeniedCreate.StatusCode -ne 403) { throw "Expected no-access customer create to return 403." }

Write-Host "Verifying audit and outbox records..."
$CustomerID = $Created.customer.id
$AuditSql = @"
SELECT set_config('app.tenant_id', '$TenantAID', false);
SELECT count(*) FROM audit.audit_log
WHERE tenant_id = '$TenantAID'
  AND action = 'sales.customer.create'
  AND target_type = 'sales.customer'
  AND target_id = '$CustomerID';
"@
if ((Invoke-ScalarSql -Sql $AuditSql) -ne "1") { throw "Expected one customer audit record." }

$OutboxSql = @"
SELECT count(*) FROM core.outbox_events
WHERE tenant_id = '$TenantAID'
  AND event_type = 'sales.customer.created.v1'
  AND aggregate_type = 'sales.customer'
  AND aggregate_id = '$CustomerID';
"@
if ((Invoke-ScalarSql -Sql $OutboxSql) -ne "1") { throw "Expected one customer outbox record." }

Write-Host "Verifying Sales entitlement guard..."
Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantAID/modules/sales/disable" -Method Post -Headers $PlatformHeaders | Out-Null
try {
  $DisabledResponse = Invoke-WebRequest "$BaseUrl/api/v1/sales/customers" -Headers $HeadersA -SkipHttpErrorCheck
  if ($DisabledResponse.StatusCode -ne 403) { throw "Expected disabled Sales module to return 403." }
  $DisabledError = $DisabledResponse.Content | ConvertFrom-Json
  if ($DisabledError.error.code -ne "module_not_enabled") { throw "Expected module_not_enabled." }
}
finally {
  Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantAID/modules/sales/enable" -Method Post -Headers $PlatformHeaders | Out-Null
}

Write-Host "Sales customers verification passed."

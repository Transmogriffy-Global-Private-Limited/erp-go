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
  foreach ($ModuleID in @("inventory", "sales")) {
    Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/$ModuleID/enable" -Method Post -Headers $PlatformHeaders | Out-Null
  }
}

$HeadersA = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") -BaseUrl $BaseUrl -TenantID $TenantAID -UserKind "allowed"
$NoAccessHeaders = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") -BaseUrl $BaseUrl -TenantID $TenantAID -UserKind "noaccess"
$HeadersB = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") -BaseUrl $BaseUrl -TenantID $TenantBID -UserKind "allowed"
$Suffix = Get-Random -Minimum 10000 -Maximum 99999

Write-Host "Creating sales order prerequisites..."
$Unit = Invoke-RestMethod "$BaseUrl/api/v1/inventory/units" -Method Post -ContentType "application/json" -Headers $HeadersA -Body (@{
  code = "SOUOM$Suffix"
  name = "Sales Order Unit $Suffix"
} | ConvertTo-Json)

$Item = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Method Post -ContentType "application/json" -Headers $HeadersA -Body (@{
  sku = "SOITEM-$Suffix"
  name = "Sales Order Item $Suffix"
  base_unit_id = $Unit.unit.id
} | ConvertTo-Json)

$Customer = Invoke-RestMethod "$BaseUrl/api/v1/sales/customers" -Method Post -ContentType "application/json" -Headers $HeadersA -Body (@{
  code = "SOCUS$Suffix"
  name = "Sales Order Customer $Suffix"
} | ConvertTo-Json)

$OrderBody = @{
  customer_id = $Customer.customer.id
  customer_reference = "CUS-REF-$Suffix"
  notes = "Created by sales order verification"
  currency_code = "inr"
  lines = @(
    @{
      item_id = $Item.item.id
      quantity = "5.000"
      unit_price = "12.5000"
    }
  )
} | ConvertTo-Json -Depth 10

Write-Host "Creating draft sales order..."
$Created = Invoke-RestMethod "$BaseUrl/api/v1/sales/orders" -Method Post -ContentType "application/json" -Headers $HeadersA -Body $OrderBody
$Order = $Created.sales_order
if (-not $Order.id -or -not $Order.order_number) { throw "Sales order create response is incomplete." }
if ($Order.status -ne "draft") { throw "Expected draft status, got $($Order.status)." }
if ($Order.currency_code -ne "INR") { throw "Expected normalized INR currency code." }
if ($Order.customer.id -ne $Customer.customer.id) { throw "Customer reference was not preserved." }
if (@($Order.lines).Count -ne 1) { throw "Expected one sales order line." }
if ([decimal]$Order.lines[0].line_total -ne [decimal]62.5) { throw "Expected line total 62.5." }

Write-Host "Listing sales orders..."
$ListA = Invoke-RestMethod "$BaseUrl/api/v1/sales/orders" -Headers $HeadersA
if (-not (@($ListA.sales_orders | Where-Object { $_.id -eq $Order.id }) | Select-Object -First 1)) {
  throw "Created sales order not found in tenant A list."
}

Write-Host "Verifying tenant isolation..."
$ListB = Invoke-RestMethod "$BaseUrl/api/v1/sales/orders" -Headers $HeadersB
if (@($ListB.sales_orders | Where-Object { $_.id -eq $Order.id }).Count -gt 0) {
  throw "Tenant B can see tenant A sales order."
}

$CrossTenantConfirm = Invoke-WebRequest "$BaseUrl/api/v1/sales/orders/$($Order.id)/confirm" -Method Post -Headers $HeadersB -SkipHttpErrorCheck
if ($CrossTenantConfirm.StatusCode -ne 404) { throw "Expected tenant B confirmation of tenant A order to return 404." }

$StockSql = @"
SELECT set_config('app.tenant_id', '$TenantAID', false);
SELECT count(*) FROM inventory.stock_movements
WHERE tenant_id = '$TenantAID' AND reference = '$($Order.order_number)';
"@
if ((Invoke-ScalarSql -Sql $StockSql) -ne "0") { throw "Draft sales order created a stock movement." }

Write-Host "Verifying no-access RBAC..."
$DeniedList = Invoke-WebRequest "$BaseUrl/api/v1/sales/orders" -Headers $NoAccessHeaders -SkipHttpErrorCheck
if ($DeniedList.StatusCode -ne 403) { throw "Expected no-access order list to return 403." }
$DeniedCreate = Invoke-WebRequest "$BaseUrl/api/v1/sales/orders" -Method Post -ContentType "application/json" -Headers $NoAccessHeaders -Body $OrderBody -SkipHttpErrorCheck
if ($DeniedCreate.StatusCode -ne 403) { throw "Expected no-access order create to return 403." }
$DeniedConfirm = Invoke-WebRequest "$BaseUrl/api/v1/sales/orders/$($Order.id)/confirm" -Method Post -Headers $NoAccessHeaders -SkipHttpErrorCheck
if ($DeniedConfirm.StatusCode -ne 403) { throw "Expected no-access order confirmation to return 403." }

Write-Host "Confirming sales order..."
$ConfirmedResponse = Invoke-RestMethod "$BaseUrl/api/v1/sales/orders/$($Order.id)/confirm" -Method Post -Headers $HeadersA
$Confirmed = $ConfirmedResponse.sales_order
if ($Confirmed.status -ne "confirmed" -or -not $Confirmed.confirmed_at) { throw "Sales order confirmation response is incomplete." }
if ((Invoke-ScalarSql -Sql $StockSql) -ne "0") { throw "Confirmed sales order created a stock movement." }

Write-Host "Verifying repeat confirmation conflict..."
$RepeatResponse = Invoke-WebRequest "$BaseUrl/api/v1/sales/orders/$($Order.id)/confirm" -Method Post -Headers $HeadersA -SkipHttpErrorCheck
if ($RepeatResponse.StatusCode -ne 409) { throw "Expected repeat confirmation to return 409." }
$RepeatError = $RepeatResponse.Content | ConvertFrom-Json
if ($RepeatError.error.code -ne "sales_order_not_draft") { throw "Expected sales_order_not_draft." }

Write-Host "Verifying invalid quantity validation..."
$InvalidBodyObject = $OrderBody | ConvertFrom-Json
$InvalidBodyObject.lines[0].quantity = "0"
$InvalidResponse = Invoke-WebRequest "$BaseUrl/api/v1/sales/orders" -Method Post -ContentType "application/json" -Headers $HeadersA -Body ($InvalidBodyObject | ConvertTo-Json -Depth 10) -SkipHttpErrorCheck
if ($InvalidResponse.StatusCode -ne 400) { throw "Expected invalid quantity to return 400." }

Write-Host "Verifying duplicate item lines are rejected..."
$DuplicateBodyObject = $OrderBody | ConvertFrom-Json
$DuplicateBodyObject.lines = @($DuplicateBodyObject.lines[0], $DuplicateBodyObject.lines[0])
$DuplicateResponse = Invoke-WebRequest "$BaseUrl/api/v1/sales/orders" -Method Post -ContentType "application/json" -Headers $HeadersA -Body ($DuplicateBodyObject | ConvertTo-Json -Depth 10) -SkipHttpErrorCheck
if ($DuplicateResponse.StatusCode -ne 400) { throw "Expected duplicate item lines to return 400." }
$DuplicateError = $DuplicateResponse.Content | ConvertFrom-Json
if ($DuplicateError.error.code -ne "duplicate_item_id") { throw "Expected duplicate_item_id." }

Write-Host "Verifying audit and outbox records..."
$AuditSql = @"
SELECT set_config('app.tenant_id', '$TenantAID', false);
SELECT count(*) FROM audit.audit_log
WHERE tenant_id = '$TenantAID'
  AND target_type = 'sales.order'
  AND target_id = '$($Order.id)'
  AND action IN ('sales.order.create', 'sales.order.confirm');
"@
if ((Invoke-ScalarSql -Sql $AuditSql) -ne "2") { throw "Expected create and confirm audit records." }

$OutboxSql = @"
SELECT count(*) FROM core.outbox_events
WHERE tenant_id = '$TenantAID'
  AND aggregate_type = 'sales.order'
  AND aggregate_id = '$($Order.id)'
  AND event_type IN ('sales.order.created.v1', 'sales.order.confirmed.v1');
"@
if ((Invoke-ScalarSql -Sql $OutboxSql) -ne "2") { throw "Expected create and confirm outbox records." }

Write-Host "Sales orders verification passed."

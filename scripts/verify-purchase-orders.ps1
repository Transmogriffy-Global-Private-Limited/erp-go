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

& (Join-Path $PSScriptRoot "ensure-local-apis.ps1") -BaseUrl $BaseUrl -ControlPlaneUrl $ControlPlaneUrl
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
  foreach ($ModuleID in @("inventory", "purchase")) {
    Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/$ModuleID/enable" -Method Post -Headers $PlatformHeaders | Out-Null
  }
}

$HeadersA = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") -BaseUrl $BaseUrl -TenantID $TenantAID -UserKind "allowed"
$NoAccessHeaders = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") -BaseUrl $BaseUrl -TenantID $TenantAID -UserKind "noaccess"
$HeadersB = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") -BaseUrl $BaseUrl -TenantID $TenantBID -UserKind "allowed"
$Suffix = Get-Random -Minimum 10000 -Maximum 99999

Write-Host "Creating order prerequisites..."
$Unit = Invoke-RestMethod "$BaseUrl/api/v1/inventory/units" -Method Post -ContentType "application/json" -Headers $HeadersA -Body (@{
  code = "POUOM$Suffix"
  name = "Purchase Order Unit $Suffix"
} | ConvertTo-Json)

$Item = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Method Post -ContentType "application/json" -Headers $HeadersA -Body (@{
  sku = "POITEM-$Suffix"
  name = "Purchase Order Item $Suffix"
  base_unit_id = $Unit.unit.id
} | ConvertTo-Json)

$Supplier = Invoke-RestMethod "$BaseUrl/api/v1/purchase/suppliers" -Method Post -ContentType "application/json" -Headers $HeadersA -Body (@{
  code = "POSUP$Suffix"
  name = "Purchase Order Supplier $Suffix"
} | ConvertTo-Json)

$OrderBody = @{
  supplier_id = $Supplier.supplier.id
  supplier_reference = "SUP-REF-$Suffix"
  notes = "Created by purchase order verification"
  currency_code = "inr"
  lines = @(
    @{
      item_id = $Item.item.id
      quantity = "5.000"
      unit_price = "12.5000"
    }
  )
} | ConvertTo-Json -Depth 10

Write-Host "Creating draft purchase order..."
$Created = Invoke-RestMethod "$BaseUrl/api/v1/purchase/orders" -Method Post -ContentType "application/json" -Headers $HeadersA -Body $OrderBody
$Order = $Created.purchase_order
if (-not $Order.id -or -not $Order.order_number) { throw "Purchase order create response is incomplete." }
if ($Order.status -ne "draft") { throw "Expected draft status, got $($Order.status)." }
if ($Order.currency_code -ne "INR") { throw "Expected normalized INR currency code." }
if ($Order.supplier.id -ne $Supplier.supplier.id) { throw "Supplier reference was not preserved." }
if (@($Order.lines).Count -ne 1) { throw "Expected one purchase order line." }
if ([decimal]$Order.lines[0].line_total -ne [decimal]62.5) { throw "Expected line total 62.5." }
if ($Order.lines[0].received_quantity -ne "0.000") { throw "Expected initial received_quantity 0.000." }
if ($Order.lines[0].remaining_quantity -ne "5.000") { throw "Expected initial remaining_quantity 5.000." }

Write-Host "Listing purchase orders..."
$ListA = Invoke-RestMethod "$BaseUrl/api/v1/purchase/orders" -Headers $HeadersA
if (-not (@($ListA.purchase_orders | Where-Object { $_.id -eq $Order.id }) | Select-Object -First 1)) {
  throw "Created purchase order not found in tenant A list."
}

Write-Host "Verifying tenant isolation..."
$ListB = Invoke-RestMethod "$BaseUrl/api/v1/purchase/orders" -Headers $HeadersB
if (@($ListB.purchase_orders | Where-Object { $_.id -eq $Order.id }).Count -gt 0) {
  throw "Tenant B can see tenant A purchase order."
}

$StockSql = @"
SELECT set_config('app.tenant_id', '$TenantAID', false);
SELECT count(*) FROM inventory.stock_movements
WHERE tenant_id = '$TenantAID' AND reference = '$($Order.order_number)';
"@
if ((Invoke-ScalarSql -Sql $StockSql) -ne "0") { throw "Draft purchase order created stock movement." }

Write-Host "Approving purchase order..."
$ApprovedResponse = Invoke-RestMethod "$BaseUrl/api/v1/purchase/orders/$($Order.id)/approve" -Method Post -Headers $HeadersA
$Approved = $ApprovedResponse.purchase_order
if ($Approved.status -ne "approved" -or -not $Approved.approved_at) { throw "Purchase order approval response is incomplete." }
if ((Invoke-ScalarSql -Sql $StockSql) -ne "0") { throw "Approved purchase order created stock movement." }

Write-Host "Verifying repeat approval conflict..."
$RepeatResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/orders/$($Order.id)/approve" -Method Post -Headers $HeadersA -SkipHttpErrorCheck
if ($RepeatResponse.StatusCode -ne 409) { throw "Expected repeat approval to return 409." }
$RepeatError = $RepeatResponse.Content | ConvertFrom-Json
if ($RepeatError.error.code -ne "purchase_order_not_draft") { throw "Expected purchase_order_not_draft." }

Write-Host "Verifying invalid quantity validation..."
$InvalidBodyObject = $OrderBody | ConvertFrom-Json
$InvalidBodyObject.lines[0].quantity = "0"
$InvalidResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/orders" -Method Post -ContentType "application/json" -Headers $HeadersA -Body ($InvalidBodyObject | ConvertTo-Json -Depth 10) -SkipHttpErrorCheck
if ($InvalidResponse.StatusCode -ne 400) { throw "Expected invalid quantity to return 400." }

Write-Host "Verifying duplicate item lines are rejected..."
$DuplicateBodyObject = $OrderBody | ConvertFrom-Json
$DuplicateBodyObject.lines = @($DuplicateBodyObject.lines[0], $DuplicateBodyObject.lines[0])
$DuplicateResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/orders" -Method Post -ContentType "application/json" -Headers $HeadersA -Body ($DuplicateBodyObject | ConvertTo-Json -Depth 10) -SkipHttpErrorCheck
if ($DuplicateResponse.StatusCode -ne 400) { throw "Expected duplicate item lines to return 400." }
$DuplicateError = $DuplicateResponse.Content | ConvertFrom-Json
if ($DuplicateError.error.code -ne "duplicate_item_id") { throw "Expected duplicate_item_id." }

Write-Host "Verifying no-access RBAC..."
$DeniedResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/orders" -Method Post -ContentType "application/json" -Headers $NoAccessHeaders -Body $OrderBody -SkipHttpErrorCheck
if ($DeniedResponse.StatusCode -ne 403) { throw "Expected no-access order create to return 403." }

Write-Host "Verifying audit and outbox records..."
$AuditSql = @"
SELECT set_config('app.tenant_id', '$TenantAID', false);
SELECT count(*) FROM audit.audit_log
WHERE tenant_id = '$TenantAID'
  AND target_type = 'purchase.order'
  AND target_id = '$($Order.id)'
  AND action IN ('purchase.order.create', 'purchase.order.approve');
"@
if ((Invoke-ScalarSql -Sql $AuditSql) -ne "2") { throw "Expected create and approve audit records." }

$OutboxSql = @"
SELECT count(*) FROM core.outbox_events
WHERE tenant_id = '$TenantAID'
  AND aggregate_type = 'purchase.order'
  AND aggregate_id = '$($Order.id)'
  AND event_type IN ('purchase.order.created.v1', 'purchase.order.approved.v1');
"@
if ((Invoke-ScalarSql -Sql $OutboxSql) -ne "2") { throw "Expected create and approve outbox records." }

Write-Host "Purchase orders verification passed."

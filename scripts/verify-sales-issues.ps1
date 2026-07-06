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
if (-not $env:MIGRATION_DATABASE_URL) { throw "MIGRATION_DATABASE_URL is missing." }

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
  foreach ($ModuleID in @("inventory", "sales")) {
    Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$TenantID/modules/$ModuleID/enable" -Method Post -Headers $PlatformHeaders | Out-Null
  }
}

$HeadersA = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") -BaseUrl $BaseUrl -TenantID $TenantAID -UserKind "allowed"
$NoAccessHeaders = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") -BaseUrl $BaseUrl -TenantID $TenantAID -UserKind "noaccess"
$HeadersB = & (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") -BaseUrl $BaseUrl -TenantID $TenantBID -UserKind "allowed"
$Suffix = Get-Random -Minimum 10000 -Maximum 99999

Write-Host "Creating sales issue prerequisites..."
$Unit = Invoke-RestMethod "$BaseUrl/api/v1/inventory/units" -Method Post -ContentType "application/json" -Headers $HeadersA -Body (@{
  code = "SIUOM$Suffix"
  name = "Sales Issue Unit $Suffix"
} | ConvertTo-Json)
$Item = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Method Post -ContentType "application/json" -Headers $HeadersA -Body (@{
  sku = "SIITEM-$Suffix"
  name = "Sales Issue Item $Suffix"
  base_unit_id = $Unit.unit.id
} | ConvertTo-Json)
$OtherItem = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Method Post -ContentType "application/json" -Headers $HeadersA -Body (@{
  sku = "SIOTHER-$Suffix"
  name = "Unordered Sales Issue Item $Suffix"
  base_unit_id = $Unit.unit.id
} | ConvertTo-Json)
$Location = Invoke-RestMethod "$BaseUrl/api/v1/inventory/locations" -Method Post -ContentType "application/json" -Headers $HeadersA -Body (@{
  code = "SILOC$Suffix"
  name = "Sales Issue Location $Suffix"
} | ConvertTo-Json)
$EmptyLocation = Invoke-RestMethod "$BaseUrl/api/v1/inventory/locations" -Method Post -ContentType "application/json" -Headers $HeadersA -Body (@{
  code = "SIEMPTY$Suffix"
  name = "Empty Sales Issue Location $Suffix"
} | ConvertTo-Json)
$Customer = Invoke-RestMethod "$BaseUrl/api/v1/sales/customers" -Method Post -ContentType "application/json" -Headers $HeadersA -Body (@{
  code = "SICUS$Suffix"
  name = "Sales Issue Customer $Suffix"
} | ConvertTo-Json)

Write-Host "Seeding 10 units of available stock..."
Invoke-RestMethod "$BaseUrl/api/v1/inventory/stock-movements" -Method Post -ContentType "application/json" -Headers $HeadersA -Body (@{
  movement_type = "adjustment"
  reference = "SI-OPENING-$Suffix"
  notes = "Opening stock for sales issue verification"
  lines = @(@{
    item_id = $Item.item.id
    location_id = $Location.location.id
    quantity_delta = "10.000"
  })
} | ConvertTo-Json -Depth 10) | Out-Null

$OrderBody = @{
  customer_id = $Customer.customer.id
  currency_code = "INR"
  lines = @(@{
    item_id = $Item.item.id
    quantity = "10.000"
    unit_price = "15.0000"
  })
} | ConvertTo-Json -Depth 10
$OrderResponse = Invoke-RestMethod "$BaseUrl/api/v1/sales/orders" -Method Post -ContentType "application/json" -Headers $HeadersA -Body $OrderBody
$Order = $OrderResponse.sales_order

$PartialBody = @{
  sales_order_id = $Order.id
  reference = "customer-dispatch-$Suffix"
  notes = "Partial sales issue verification"
  lines = @(@{
    item_id = $Item.item.id
    location_id = $Location.location.id
    quantity_issued = "4.000"
  })
} | ConvertTo-Json -Depth 10

Write-Host "Verifying draft sales order cannot be issued..."
$DraftResponse = Invoke-WebRequest "$BaseUrl/api/v1/sales/issues" -Method Post -ContentType "application/json" -Headers $HeadersA -Body $PartialBody -SkipHttpErrorCheck
if ($DraftResponse.StatusCode -ne 400) { throw "Expected draft sales order issue to return 400." }
$DraftError = $DraftResponse.Content | ConvertFrom-Json
if ($DraftError.error.code -ne "sales_order_not_fulfillable") { throw "Expected sales_order_not_fulfillable." }

Invoke-RestMethod "$BaseUrl/api/v1/sales/orders/$($Order.id)/confirm" -Method Post -Headers $HeadersA | Out-Null

Write-Host "Verifying cross-tenant issue is rejected..."
$CrossTenantResponse = Invoke-WebRequest "$BaseUrl/api/v1/sales/issues" -Method Post -ContentType "application/json" -Headers $HeadersB -Body $PartialBody -SkipHttpErrorCheck
if ($CrossTenantResponse.StatusCode -ne 400) { throw "Expected cross-tenant issue to return 400." }
$CrossTenantError = $CrossTenantResponse.Content | ConvertFrom-Json
if ($CrossTenantError.error.code -ne "sales_order_not_fulfillable") { throw "Expected sales_order_not_fulfillable for cross-tenant issue." }

Write-Host "Verifying unordered item cannot be issued..."
$UnorderedBody = $PartialBody | ConvertFrom-Json
$UnorderedBody.lines[0].item_id = $OtherItem.item.id
$UnorderedResponse = Invoke-WebRequest "$BaseUrl/api/v1/sales/issues" -Method Post -ContentType "application/json" -Headers $HeadersA -Body ($UnorderedBody | ConvertTo-Json -Depth 10) -SkipHttpErrorCheck
if ($UnorderedResponse.StatusCode -ne 400) { throw "Expected unordered item issue to return 400." }
$UnorderedError = $UnorderedResponse.Content | ConvertFrom-Json
if ($UnorderedError.error.code -ne "issue_item_not_on_order") { throw "Expected issue_item_not_on_order." }

Write-Host "Verifying stock availability by location..."
$EmptyLocationBody = $PartialBody | ConvertFrom-Json
$EmptyLocationBody.lines[0].location_id = $EmptyLocation.location.id
$EmptyLocationBody.lines[0].quantity_issued = "1.000"
$EmptyResponse = Invoke-WebRequest "$BaseUrl/api/v1/sales/issues" -Method Post -ContentType "application/json" -Headers $HeadersA -Body ($EmptyLocationBody | ConvertTo-Json -Depth 10) -SkipHttpErrorCheck
if ($EmptyResponse.StatusCode -ne 409) { throw "Expected empty-location issue to return 409." }
$EmptyError = $EmptyResponse.Content | ConvertFrom-Json
if ($EmptyError.error.code -ne "insufficient_stock") { throw "Expected insufficient_stock." }

Write-Host "Verifying issue payload validation..."
$InvalidBody = $PartialBody | ConvertFrom-Json
$InvalidBody.lines[0].quantity_issued = "0"
$InvalidResponse = Invoke-WebRequest "$BaseUrl/api/v1/sales/issues" -Method Post -ContentType "application/json" -Headers $HeadersA -Body ($InvalidBody | ConvertTo-Json -Depth 10) -SkipHttpErrorCheck
if ($InvalidResponse.StatusCode -ne 400) { throw "Expected invalid issue quantity to return 400." }
$InvalidError = $InvalidResponse.Content | ConvertFrom-Json
if ($InvalidError.error.code -ne "invalid_quantity_issued") { throw "Expected invalid_quantity_issued." }

$DuplicateBody = $PartialBody | ConvertFrom-Json
$DuplicateBody.lines = @($DuplicateBody.lines[0], $DuplicateBody.lines[0])
$DuplicateResponse = Invoke-WebRequest "$BaseUrl/api/v1/sales/issues" -Method Post -ContentType "application/json" -Headers $HeadersA -Body ($DuplicateBody | ConvertTo-Json -Depth 10) -SkipHttpErrorCheck
if ($DuplicateResponse.StatusCode -ne 400) { throw "Expected duplicate issue item to return 400." }
$DuplicateError = $DuplicateResponse.Content | ConvertFrom-Json
if ($DuplicateError.error.code -ne "duplicate_item_id") { throw "Expected duplicate_item_id." }

Write-Host "Verifying no-access RBAC..."
$DeniedList = Invoke-WebRequest "$BaseUrl/api/v1/sales/issues" -Headers $NoAccessHeaders -SkipHttpErrorCheck
if ($DeniedList.StatusCode -ne 403) { throw "Expected no-access issue list to return 403." }
$DeniedCreate = Invoke-WebRequest "$BaseUrl/api/v1/sales/issues" -Method Post -ContentType "application/json" -Headers $NoAccessHeaders -Body $PartialBody -SkipHttpErrorCheck
if ($DeniedCreate.StatusCode -ne 403) { throw "Expected no-access issue create to return 403." }

Write-Host "Posting partial sales issue..."
$PartialResponse = Invoke-RestMethod "$BaseUrl/api/v1/sales/issues" -Method Post -ContentType "application/json" -Headers $HeadersA -Body $PartialBody
$Partial = $PartialResponse.issue
if ($Partial.status -ne "posted" -or -not $Partial.stock_movement_id) { throw "Partial sales issue response is incomplete." }
if ([decimal]$Partial.lines[0].quantity_issued -ne [decimal]4) { throw "Expected partial issue quantity 4." }

$MovementDeltaSql = @"
SELECT set_config('app.tenant_id', '$TenantAID', false);
SELECT quantity_delta::text FROM inventory.stock_movement_lines
WHERE tenant_id = '$TenantAID' AND movement_id = '$($Partial.stock_movement_id)'::uuid;
"@
if ([decimal](Invoke-ScalarSql -Sql $MovementDeltaSql) -ne [decimal]-4) { throw "Expected partial issue movement delta -4." }

$OrderList = Invoke-RestMethod "$BaseUrl/api/v1/sales/orders" -Headers $HeadersA
$OrderAfterPartial = @($OrderList.sales_orders | Where-Object { $_.id -eq $Order.id }) | Select-Object -First 1
if ($OrderAfterPartial.status -ne "partially_fulfilled") { throw "Expected partially_fulfilled order status." }
if ($OrderAfterPartial.lines[0].issued_quantity -ne "4.000") { throw "Expected issued_quantity 4.000." }
if ($OrderAfterPartial.lines[0].remaining_quantity -ne "6.000") { throw "Expected remaining_quantity 6.000." }

Write-Host "Verifying tenant isolation..."
$ListA = Invoke-RestMethod "$BaseUrl/api/v1/sales/issues" -Headers $HeadersA
if (-not (@($ListA.issues | Where-Object { $_.id -eq $Partial.id }) | Select-Object -First 1)) { throw "Partial issue missing from tenant A list." }
$ListB = Invoke-RestMethod "$BaseUrl/api/v1/sales/issues" -Headers $HeadersB
if (@($ListB.issues | Where-Object { $_.id -eq $Partial.id }).Count -gt 0) { throw "Tenant B can see tenant A sales issue." }

Write-Host "Verifying over-issue is rejected..."
$OverBody = $PartialBody | ConvertFrom-Json
$OverBody.lines[0].quantity_issued = "7.000"
$OverResponse = Invoke-WebRequest "$BaseUrl/api/v1/sales/issues" -Method Post -ContentType "application/json" -Headers $HeadersA -Body ($OverBody | ConvertTo-Json -Depth 10) -SkipHttpErrorCheck
if ($OverResponse.StatusCode -ne 409) { throw "Expected over-issue to return 409." }
$OverError = $OverResponse.Content | ConvertFrom-Json
if ($OverError.error.code -ne "issue_quantity_exceeds_remaining") { throw "Expected issue_quantity_exceeds_remaining." }

Write-Host "Posting exact remaining quantity..."
$FinalBody = $PartialBody | ConvertFrom-Json
$FinalBody.reference = "final-dispatch-$Suffix"
$FinalBody.notes = "Final sales issue verification"
$FinalBody.lines[0].quantity_issued = "6.000"
$FinalResponse = Invoke-RestMethod "$BaseUrl/api/v1/sales/issues" -Method Post -ContentType "application/json" -Headers $HeadersA -Body ($FinalBody | ConvertTo-Json -Depth 10)
$Final = $FinalResponse.issue
if ($Final.status -ne "posted") { throw "Final sales issue was not posted." }

$FinalOrderList = Invoke-RestMethod "$BaseUrl/api/v1/sales/orders" -Headers $HeadersA
$FulfilledOrder = @($FinalOrderList.sales_orders | Where-Object { $_.id -eq $Order.id }) | Select-Object -First 1
if ($FulfilledOrder.status -ne "fulfilled") { throw "Expected fulfilled order status." }
if ($FulfilledOrder.lines[0].issued_quantity -ne "10.000") { throw "Expected issued_quantity 10.000." }
if ($FulfilledOrder.lines[0].remaining_quantity -ne "0.000") { throw "Expected remaining_quantity 0.000." }

$BalanceSql = @"
SELECT set_config('app.tenant_id', '$TenantAID', false);
SELECT COALESCE(SUM(quantity_delta), 0)::numeric(18, 3)::text
FROM inventory.stock_movement_lines
WHERE tenant_id = '$TenantAID'
  AND item_id = '$($Item.item.id)'::uuid
  AND location_id = '$($Location.location.id)'::uuid;
"@
if ((Invoke-ScalarSql -Sql $BalanceSql) -ne "0.000") { throw "Expected final stock balance 0.000." }

Write-Host "Verifying fulfilled order cannot be issued again..."
$RepeatResponse = Invoke-WebRequest "$BaseUrl/api/v1/sales/issues" -Method Post -ContentType "application/json" -Headers $HeadersA -Body ($FinalBody | ConvertTo-Json -Depth 10) -SkipHttpErrorCheck
if ($RepeatResponse.StatusCode -ne 400) { throw "Expected fulfilled-order issue to return 400." }
$RepeatError = $RepeatResponse.Content | ConvertFrom-Json
if ($RepeatError.error.code -ne "sales_order_not_fulfillable") { throw "Expected sales_order_not_fulfillable after fulfillment." }

Write-Host "Verifying audit and outbox records..."
$IssueAuditSql = @"
SELECT set_config('app.tenant_id', '$TenantAID', false);
SELECT count(*) FROM audit.audit_log
WHERE tenant_id = '$TenantAID'
  AND action = 'sales.issue.post'
  AND target_id IN ('$($Partial.id)', '$($Final.id)');
"@
if ((Invoke-ScalarSql -Sql $IssueAuditSql) -ne "2") { throw "Expected two sales issue audit records." }

$IssueOutboxSql = @"
SELECT count(*) FROM core.outbox_events
WHERE tenant_id = '$TenantAID'
  AND event_type = 'sales.issue.posted.v1'
  AND aggregate_id IN ('$($Partial.id)', '$($Final.id)');
"@
if ((Invoke-ScalarSql -Sql $IssueOutboxSql) -ne "2") { throw "Expected two sales issue outbox records." }

$OrderProgressOutboxSql = @"
SELECT count(*) FROM core.outbox_events
WHERE tenant_id = '$TenantAID'
  AND aggregate_type = 'sales.order'
  AND aggregate_id = '$($Order.id)'
  AND event_type IN ('sales.order.partially_fulfilled.v1', 'sales.order.fulfilled.v1');
"@
if ((Invoke-ScalarSql -Sql $OrderProgressOutboxSql) -ne "2") { throw "Expected partial and fulfilled order events." }

Write-Host "Sales issues verification passed."

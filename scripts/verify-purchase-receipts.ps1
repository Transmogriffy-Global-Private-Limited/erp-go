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

Write-Host ""
Write-Host "Creating receipt verification supplier..."
$Supplier = Invoke-RestMethod "$BaseUrl/api/v1/purchase/suppliers" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body (@{
    code = "RCVSUP$Suffix"
    name = "Receipt Supplier $Suffix"
  } | ConvertTo-Json)

Write-Host "Creating and approving purchase order..."
$Order = Invoke-RestMethod "$BaseUrl/api/v1/purchase/orders" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body (@{
    supplier_id = $Supplier.supplier.id
    supplier_reference = "supplier-ref-$Suffix"
    currency_code = "INR"
    lines = @(
      @{
        item_id = $Item.item.id
        quantity = "10.000"
        unit_price = "25.0000"
      }
    )
  } | ConvertTo-Json -Depth 10)

Write-Host "Verifying draft purchase order cannot be received..."
$DraftReceiptResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/receipts" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body (@{
    purchase_order_id = $Order.purchase_order.id
    lines = @(
      @{
        item_id = $Item.item.id
        location_id = $Location.location.id
        quantity_received = "1.000"
      }
    )
  } | ConvertTo-Json -Depth 10) `
  -SkipHttpErrorCheck

if ($DraftReceiptResponse.StatusCode -ne 400) {
  throw "Expected draft purchase order receipt to return 400."
}

$DraftReceiptError = $DraftReceiptResponse.Content | ConvertFrom-Json
if ($DraftReceiptError.error.code -ne "purchase_order_not_receivable") {
  throw "Expected purchase_order_not_receivable, got $($DraftReceiptError.error.code)."
}

$ApprovedOrder = Invoke-RestMethod "$BaseUrl/api/v1/purchase/orders/$($Order.purchase_order.id)/approve" `
  -Method Post `
  -Headers $Headers

if ($ApprovedOrder.purchase_order.status -ne "approved") {
  throw "Purchase order was not approved."
}

Write-Host "Verifying unordered item cannot be received..."
$OtherItem = Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body (@{
    sku = "RCVOTHER-$Suffix"
    name = "Unordered Receipt Item $Suffix"
    base_unit_id = $Unit.unit.id
  } | ConvertTo-Json)

$UnorderedItemResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/receipts" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body (@{
    purchase_order_id = $Order.purchase_order.id
    lines = @(
      @{
        item_id = $OtherItem.item.id
        location_id = $Location.location.id
        quantity_received = "1.000"
      }
    )
  } | ConvertTo-Json -Depth 10) `
  -SkipHttpErrorCheck

if ($UnorderedItemResponse.StatusCode -ne 400) {
  throw "Expected unordered receipt item to return 400."
}

$UnorderedItemError = $UnorderedItemResponse.Content | ConvertFrom-Json
if ($UnorderedItemError.error.code -ne "receipt_item_not_on_order") {
  throw "Expected receipt_item_not_on_order, got $($UnorderedItemError.error.code)."
}

$ReceiptBody = @{
  purchase_order_id = $Order.purchase_order.id
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

if ($Receipt.receipt.purchase_order_id -ne $Order.purchase_order.id) {
  throw "Receipt did not return the linked purchase_order_id."
}

if ($Receipt.receipt.supplier.id -ne $Supplier.supplier.id) {
  throw "Receipt supplier was not derived from the purchase order."
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

Write-Host "Checking partially received order progress..."
$PartiallyReceivedOrders = Invoke-RestMethod "$BaseUrl/api/v1/purchase/orders" -Headers $Headers
$PartiallyReceivedOrder = @($PartiallyReceivedOrders.purchase_orders | Where-Object { $_.id -eq $Order.purchase_order.id }) | Select-Object -First 1
if ($PartiallyReceivedOrder.status -ne "partially_received") {
  throw "Expected partially_received status, got $($PartiallyReceivedOrder.status)."
}
if ($PartiallyReceivedOrder.lines[0].received_quantity -ne "7.000") {
  throw "Expected received_quantity 7.000."
}
if ($PartiallyReceivedOrder.lines[0].remaining_quantity -ne "3.000") {
  throw "Expected remaining_quantity 3.000."
}

Write-Host ""
Write-Host "Verifying over-receipt is rejected..."
$OverReceiptBody = @{
  purchase_order_id = $Order.purchase_order.id
  lines = @(
    @{
      item_id = $Item.item.id
      location_id = $Location.location.id
      quantity_received = "4.000"
    }
  )
} | ConvertTo-Json -Depth 10

$OverReceiptResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/receipts" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $OverReceiptBody `
  -SkipHttpErrorCheck

if ($OverReceiptResponse.StatusCode -ne 409) {
  throw "Expected over-receipt to return 409, got $($OverReceiptResponse.StatusCode)."
}

$OverReceiptError = $OverReceiptResponse.Content | ConvertFrom-Json
if ($OverReceiptError.error.code -ne "receipt_quantity_exceeds_remaining") {
  throw "Expected receipt_quantity_exceeds_remaining, got $($OverReceiptError.error.code)."
}

Write-Host "Over-receipt rejection verified."

Write-Host ""
Write-Host "Receiving exact remaining quantity..."
$RemainingReceiptBody = @{
  purchase_order_id = $Order.purchase_order.id
  lines = @(
    @{
      item_id = $Item.item.id
      location_id = $Location.location.id
      quantity_received = "3.000"
    }
  )
} | ConvertTo-Json -Depth 10

$RemainingReceipt = Invoke-RestMethod "$BaseUrl/api/v1/purchase/receipts" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body $RemainingReceiptBody

$FinalBalances = Invoke-RestMethod "$BaseUrl/api/v1/inventory/stock-balances" -Headers $Headers
$FinalBalance = @($FinalBalances.balances | Where-Object { $_.item_id -eq $Item.item.id -and $_.location_id -eq $Location.location.id }) | Select-Object -First 1
if ($FinalBalance.quantity -ne "10.000") {
  throw "Expected final stock balance 10.000, got $($FinalBalance.quantity)."
}

Write-Host "Partial receipt completion verified."

Write-Host "Checking received order progress..."
$ReceivedOrders = Invoke-RestMethod "$BaseUrl/api/v1/purchase/orders" -Headers $Headers
$ReceivedOrder = @($ReceivedOrders.purchase_orders | Where-Object { $_.id -eq $Order.purchase_order.id }) | Select-Object -First 1
if ($ReceivedOrder.status -ne "received") {
  throw "Expected received status, got $($ReceivedOrder.status)."
}
if ($ReceivedOrder.lines[0].received_quantity -ne "10.000") {
  throw "Expected final received_quantity 10.000."
}
if ($ReceivedOrder.lines[0].remaining_quantity -ne "0.000") {
  throw "Expected final remaining_quantity 0.000."
}

Write-Host ""
Write-Host "Verifying receipt reversal requires a reason..."
$MissingReasonResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/receipts/$($RemainingReceipt.receipt.id)/reverse" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body (@{ reason = "" } | ConvertTo-Json) `
  -SkipHttpErrorCheck
if ($MissingReasonResponse.StatusCode -ne 400) {
  throw "Expected missing reversal reason to return 400."
}
$MissingReasonError = $MissingReasonResponse.Content | ConvertFrom-Json
if ($MissingReasonError.error.code -ne "reversal_reason_required") {
  throw "Expected reversal_reason_required."
}

Write-Host "Verifying no-access user cannot reverse a receipt..."
$DeniedReverseResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/receipts/$($RemainingReceipt.receipt.id)/reverse" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $NoAccessHeaders `
  -Body (@{ reason = "Denied reversal" } | ConvertTo-Json) `
  -SkipHttpErrorCheck
if ($DeniedReverseResponse.StatusCode -ne 403) {
  throw "Expected no-access reversal to return 403."
}

Write-Host "Reversing final partial receipt..."
$ReversedRemaining = Invoke-RestMethod "$BaseUrl/api/v1/purchase/receipts/$($RemainingReceipt.receipt.id)/reverse" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body (@{ reason = "Verification reversal of remaining receipt" } | ConvertTo-Json)
if ($ReversedRemaining.receipt.status -ne "reversed") {
  throw "Expected reversed receipt status."
}
if (-not $ReversedRemaining.receipt.reversal_stock_movement_id) {
  throw "Reversal did not return reversal_stock_movement_id."
}

$ReversalMovements = Invoke-RestMethod "$BaseUrl/api/v1/inventory/stock-movements" -Headers $Headers
$ReversalMovement = @($ReversalMovements.movements | Where-Object { $_.id -eq $ReversedRemaining.receipt.reversal_stock_movement_id }) | Select-Object -First 1
if (-not $ReversalMovement) {
  throw "Receipt reversal stock movement was not found."
}
if ($ReversalMovement.movement_type -ne "receipt_reversal") {
  throw "Expected receipt_reversal movement type."
}
if ($ReversalMovement.lines[0].quantity_delta -ne "-3.000") {
  throw "Expected reversal quantity_delta -3.000."
}

$ReopenedBalances = Invoke-RestMethod "$BaseUrl/api/v1/inventory/stock-balances" -Headers $Headers
$ReopenedBalance = @($ReopenedBalances.balances | Where-Object { $_.item_id -eq $Item.item.id -and $_.location_id -eq $Location.location.id }) | Select-Object -First 1
if ($ReopenedBalance.quantity -ne "7.000") {
  throw "Expected stock balance 7.000 after first reversal."
}

$ReopenedOrders = Invoke-RestMethod "$BaseUrl/api/v1/purchase/orders" -Headers $Headers
$ReopenedOrder = @($ReopenedOrders.purchase_orders | Where-Object { $_.id -eq $Order.purchase_order.id }) | Select-Object -First 1
if ($ReopenedOrder.status -ne "partially_received") {
  throw "Expected partially_received after first reversal."
}
if ($ReopenedOrder.lines[0].received_quantity -ne "7.000" -or $ReopenedOrder.lines[0].remaining_quantity -ne "3.000") {
  throw "Unexpected Purchase Order progress after first reversal."
}

Write-Host "Verifying repeated reversal is rejected..."
$RepeatReverseResponse = Invoke-WebRequest "$BaseUrl/api/v1/purchase/receipts/$($RemainingReceipt.receipt.id)/reverse" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body (@{ reason = "Repeat reversal" } | ConvertTo-Json) `
  -SkipHttpErrorCheck
if ($RepeatReverseResponse.StatusCode -ne 409) {
  throw "Expected repeated reversal to return 409."
}
$RepeatReverseError = $RepeatReverseResponse.Content | ConvertFrom-Json
if ($RepeatReverseError.error.code -ne "purchase_receipt_already_reversed") {
  throw "Expected purchase_receipt_already_reversed."
}

Write-Host "Reversing original receipt..."
$ReversedOriginal = Invoke-RestMethod "$BaseUrl/api/v1/purchase/receipts/$($Receipt.receipt.id)/reverse" `
  -Method Post `
  -ContentType "application/json" `
  -Headers $Headers `
  -Body (@{ reason = "Verification reversal of original receipt" } | ConvertTo-Json)
if ($ReversedOriginal.receipt.status -ne "reversed") {
  throw "Expected original receipt to be reversed."
}

$ZeroBalances = Invoke-RestMethod "$BaseUrl/api/v1/inventory/stock-balances" -Headers $Headers
$ZeroBalance = @($ZeroBalances.balances | Where-Object { $_.item_id -eq $Item.item.id -and $_.location_id -eq $Location.location.id }) | Select-Object -First 1
if ($ZeroBalance.quantity -ne "0.000") {
  throw "Expected stock balance 0.000 after all reversals, got $($ZeroBalance.quantity)."
}

$ResetOrders = Invoke-RestMethod "$BaseUrl/api/v1/purchase/orders" -Headers $Headers
$ResetOrder = @($ResetOrders.purchase_orders | Where-Object { $_.id -eq $Order.purchase_order.id }) | Select-Object -First 1
if ($ResetOrder.status -ne "approved") {
  throw "Expected approved status after reversing all receipts."
}
if ($ResetOrder.lines[0].received_quantity -ne "0.000" -or $ResetOrder.lines[0].remaining_quantity -ne "10.000") {
  throw "Unexpected Purchase Order progress after all reversals."
}

Write-Host ""
Write-Host "Verifying invalid zero quantity is rejected..."
$ZeroReceiptBody = @{
  purchase_order_id = $Order.purchase_order.id
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
Write-Host "Verifying Purchase Order receipt lifecycle audit and outbox records..."
$OrderID = $Order.purchase_order.id
$LifecycleAuditSql = @"
SELECT set_config('app.tenant_id', '$TenantID', false);
SELECT count(*) FROM audit.audit_log
WHERE tenant_id = '$TenantID'
  AND target_type = 'purchase.order'
  AND target_id = '$OrderID'
  AND action IN ('purchase.order.partially_received', 'purchase.order.received');
"@
if ((Invoke-ScalarSql -Sql $LifecycleAuditSql) -ne "2") {
  throw "Expected partially_received and received audit records."
}

$LifecycleOutboxSql = @"
SELECT count(*) FROM core.outbox_events
WHERE tenant_id = '$TenantID'
  AND aggregate_type = 'purchase.order'
  AND aggregate_id = '$OrderID'
  AND event_type IN ('purchase.order.partially_received.v1', 'purchase.order.received.v1');
"@
if ((Invoke-ScalarSql -Sql $LifecycleOutboxSql) -ne "2") {
  throw "Expected partially_received and received outbox records."
}

$ReversalAuditSql = @"
SELECT set_config('app.tenant_id', '$TenantID', false);
SELECT count(*) FROM audit.audit_log
WHERE tenant_id = '$TenantID'
  AND action = 'purchase.receipt.reverse'
  AND target_id IN ('$($Receipt.receipt.id)', '$($RemainingReceipt.receipt.id)');
"@
if ((Invoke-ScalarSql -Sql $ReversalAuditSql) -ne "2") {
  throw "Expected two receipt reversal audit records."
}

$ReversalOutboxSql = @"
SELECT count(*) FROM core.outbox_events
WHERE tenant_id = '$TenantID'
  AND event_type = 'purchase.receipt.reversed.v1'
  AND aggregate_id IN ('$($Receipt.receipt.id)', '$($RemainingReceipt.receipt.id)');
"@
if ((Invoke-ScalarSql -Sql $ReversalOutboxSql) -ne "2") {
  throw "Expected two receipt reversal outbox records."
}

$ReopenedOrderEventSql = @"
SELECT count(*) FROM core.outbox_events
WHERE tenant_id = '$TenantID'
  AND event_type = 'purchase.order.receipt_progress_reopened.v1'
  AND aggregate_id = '$OrderID';
"@
if ((Invoke-ScalarSql -Sql $ReopenedOrderEventSql) -ne "2") {
  throw "Expected two Purchase Order receipt progress reopened events."
}

$ReopenedOrderAuditSql = @"
SELECT set_config('app.tenant_id', '$TenantID', false);
SELECT count(*) FROM audit.audit_log
WHERE tenant_id = '$TenantID'
  AND action = 'purchase.order.receipt_progress_reopened'
  AND target_id = '$OrderID';
"@
if ((Invoke-ScalarSql -Sql $ReopenedOrderAuditSql) -ne "2") {
  throw "Expected two Purchase Order receipt progress reopened audit records."
}

Write-Host ""
Write-Host "Purchase receipts verification passed."

param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $TenantAID = "00000000-0000-0000-0000-000000000001",
  [string] $TenantBID = "00000000-0000-0000-0000-000000000002",
  [string] $EnvFile = ".env"
)
$ErrorActionPreference="Stop";$Root=Resolve-Path(Join-Path $PSScriptRoot "..");Set-Location $Root
. (Join-Path $PSScriptRoot "Import-DotEnv.ps1") -Path (Join-Path $Root $EnvFile)
if(-not $env:MIGRATION_DATABASE_URL){throw "MIGRATION_DATABASE_URL is missing."}
function Scalar([string]$Sql){$o=@(psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -At -c $Sql);if($LASTEXITCODE-ne 0){throw "SQL failed."};return(($o|Select-Object -Last 1)-join"").Trim()}
& (Join-Path $PSScriptRoot "ensure-local-apis.ps1") -BaseUrl $BaseUrl -ControlPlaneUrl $ControlPlaneUrl
$Platform=& (Join-Path $PSScriptRoot "Get-PlatformSessionHeaders.ps1") -ControlPlaneUrl $ControlPlaneUrl -EnvFile $EnvFile
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") -TenantID $TenantAID -Slug "dev-tenant-a" -EnvFile $EnvFile
& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") -TenantID $TenantAID -EnvFile $EnvFile
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") -TenantID $TenantBID -Slug "dev-tenant-b" -EnvFile $EnvFile
& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") -TenantID $TenantBID -AllowedUserID "55555555-5555-5555-5555-555555555555" -AllowedRoleID "77777777-7777-7777-7777-777777777777" -NoAccessUserID "66666666-6666-6666-6666-666666666666" -NoAccessRoleID "88888888-8888-8888-8888-888888888888" -EnvFile $EnvFile
foreach($tid in @($TenantAID,$TenantBID)){Invoke-RestMethod "$ControlPlaneUrl/control/v1/tenants/$tid/modules/inventory/enable" -Method Post -Headers $Platform|Out-Null}
$A=& (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") -BaseUrl $BaseUrl -TenantID $TenantAID -UserKind allowed
$B=& (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") -BaseUrl $BaseUrl -TenantID $TenantBID -UserKind allowed
$Denied=& (Join-Path $PSScriptRoot "Get-TenantSessionHeaders.ps1") -BaseUrl $BaseUrl -TenantID $TenantAID -UserKind noaccess
$n=Get-Random -Minimum 10000 -Maximum 99999
$Unit=Invoke-RestMethod "$BaseUrl/api/v1/inventory/units" -Method Post -ContentType application/json -Headers $A -Body(@{code="V1U$n";name="Inventory V1 Unit $n"}|ConvertTo-Json)
$Item=Invoke-RestMethod "$BaseUrl/api/v1/inventory/items" -Method Post -ContentType application/json -Headers $A -Body(@{sku="V1ITEM-$n";name="Inventory V1 Item $n";base_unit_id=$Unit.unit.id}|ConvertTo-Json)
$Src=Invoke-RestMethod "$BaseUrl/api/v1/inventory/locations" -Method Post -ContentType application/json -Headers $A -Body(@{code="V1SRC$n";name="Source $n"}|ConvertTo-Json)
$Dst=Invoke-RestMethod "$BaseUrl/api/v1/inventory/locations" -Method Post -ContentType application/json -Headers $A -Body(@{code="V1DST$n";name="Destination $n"}|ConvertTo-Json)
Invoke-RestMethod "$BaseUrl/api/v1/inventory/stock-movements" -Method Post -ContentType application/json -Headers $A -Body(@{movement_type="adjustment";reference="V1-OPEN-$n";lines=@(@{item_id=$Item.item.id;location_id=$Src.location.id;quantity_delta="10.000"})}|ConvertTo-Json -Depth 8)|Out-Null

Write-Host "Step 48: verifying atomic inventory transfer..."
$TransferBody=@{reference="V1-TR-$n";lines=@(@{item_id=$Item.item.id;source_location_id=$Src.location.id;destination_location_id=$Dst.location.id;quantity="4.000"})}|ConvertTo-Json -Depth 8
$Transfer=(Invoke-RestMethod "$BaseUrl/api/v1/inventory/transfers" -Method Post -ContentType application/json -Headers $A -Body $TransferBody).transfer
if(-not $Transfer.id-or $Transfer.lines[0].quantity-ne"4.000"){throw "Transfer response incomplete."}
$Net=Scalar "SELECT set_config('app.tenant_id','$TenantAID',false);SELECT COALESCE(SUM(quantity_delta),0)::numeric(18,3)::text FROM inventory.stock_movement_lines WHERE movement_id='$($Transfer.stock_movement_id)'::uuid;"
if($Net-ne"0.000"){throw "Transfer movement must net to zero."}
$TooMuch=$TransferBody|ConvertFrom-Json;$TooMuch.lines[0].quantity="7.000";$Resp=Invoke-WebRequest "$BaseUrl/api/v1/inventory/transfers" -Method Post -ContentType application/json -Headers $A -Body($TooMuch|ConvertTo-Json -Depth 8)-SkipHttpErrorCheck
if($Resp.StatusCode-ne409-or($Resp.Content|ConvertFrom-Json).error.code-ne"insufficient_available_stock"){throw "Transfer availability guard failed."}
$Same=$TransferBody|ConvertFrom-Json;$Same.lines[0].destination_location_id=$Src.location.id;$Resp=Invoke-WebRequest "$BaseUrl/api/v1/inventory/transfers" -Method Post -ContentType application/json -Headers $A -Body($Same|ConvertTo-Json -Depth 8)-SkipHttpErrorCheck;if($Resp.StatusCode-ne400){throw "Same-location transfer was accepted."}
$BTransfers=Invoke-RestMethod "$BaseUrl/api/v1/inventory/transfers" -Headers $B;if(@($BTransfers.transfers|Where-Object id -eq $Transfer.id).Count){throw "Tenant B can see tenant A transfer."}

Write-Host "Step 49: verifying stock count variance..."
$Count=(Invoke-RestMethod "$BaseUrl/api/v1/inventory/stock-counts" -Method Post -ContentType application/json -Headers $A -Body(@{location_id=$Dst.location.id;reference="V1-SC-$n";lines=@(@{item_id=$Item.item.id;counted_quantity="5.000"})}|ConvertTo-Json -Depth 8)).stock_count
if($Count.lines[0].system_quantity-ne"4.000"-or $Count.lines[0].variance_quantity-ne"1.000"){throw "Stock count variance is wrong."}
$Counts=Invoke-RestMethod "$BaseUrl/api/v1/inventory/stock-counts" -Headers $A;if(-not(@($Counts.stock_counts|Where-Object id -eq $Count.id)|Select-Object -First 1)){throw "Stock count missing from list."}

Write-Host "Step 50: verifying reservations and ATP..."
$ReservationBody=@{item_id=$Item.item.id;location_id=$Src.location.id;quantity="3.000";reference="V1-RS-$n"}|ConvertTo-Json
$Reservation=(Invoke-RestMethod "$BaseUrl/api/v1/inventory/reservations" -Method Post -ContentType application/json -Headers $A -Body $ReservationBody).reservation
$Availability=(Invoke-RestMethod "$BaseUrl/api/v1/inventory/availability" -Headers $A).availability|Where-Object{$_.item_id-eq$Item.item.id-and$_.location_id-eq$Src.location.id}|Select-Object -First 1
if($Availability.on_hand-ne"6.000"-or$Availability.reserved-ne"3.000"-or$Availability.available-ne"3.000"){throw "ATP projection is wrong."}
$ReservedTransfer=$TransferBody|ConvertFrom-Json;$ReservedTransfer.lines[0].quantity="4.000";$Resp=Invoke-WebRequest "$BaseUrl/api/v1/inventory/transfers" -Method Post -ContentType application/json -Headers $A -Body($ReservedTransfer|ConvertTo-Json -Depth 8)-SkipHttpErrorCheck;if($Resp.StatusCode-ne409){throw "Reserved stock was transferable."}
$Released=(Invoke-RestMethod "$BaseUrl/api/v1/inventory/reservations/$($Reservation.id)/release" -Method Post -Headers $A).reservation;if($Released.status-ne"released"){throw "Reservation release failed."}
$Resp=Invoke-WebRequest "$BaseUrl/api/v1/inventory/reservations/$($Reservation.id)/release" -Method Post -Headers $A -SkipHttpErrorCheck;if($Resp.StatusCode-ne409){throw "Repeated reservation release was accepted."}

Write-Host "Step 51: verifying append-only cost and valuation..."
$Cost=(Invoke-RestMethod "$BaseUrl/api/v1/inventory/cost-layers" -Method Post -ContentType application/json -Headers $A -Body(@{item_id=$Item.item.id;unit_cost="2.5000";currency_code="inr";source="verification"}|ConvertTo-Json)).cost_layer
if($Cost.currency_code-ne"INR"-or$Cost.unit_cost-ne"2.5000"){throw "Cost layer normalization failed."}
$Valuation=(Invoke-RestMethod "$BaseUrl/api/v1/inventory/reports/valuation" -Headers $A).valuation|Where-Object item_id -eq $Item.item.id
if([decimal](($Valuation|Measure-Object -Property value -Sum).Sum)-ne[decimal]27.5){throw "Inventory valuation should be 27.5."}

Write-Host "Step 52: verifying consolidated reports and RBAC..."
$Summary=(Invoke-RestMethod "$BaseUrl/api/v1/inventory/reports/summary" -Headers $A).summary|Where-Object item_id -eq $Item.item.id|Select-Object -First 1
if($Summary.on_hand-ne"11.000"-or$Summary.available-ne"11.000"-or[decimal]$Summary.inventory_value-ne[decimal]27.5){throw "Inventory summary is wrong."}
$Ledger=Invoke-RestMethod "$BaseUrl/api/v1/inventory/reports/stock-ledger" -Headers $A;if(-not(@($Ledger.stock_ledger|Where-Object id -eq $Transfer.stock_movement_id)|Select-Object -First 1)){throw "Transfer missing from stock ledger report."}
$Resp=Invoke-WebRequest "$BaseUrl/api/v1/inventory/reports/summary" -Headers $Denied -SkipHttpErrorCheck;if($Resp.StatusCode-ne403){throw "No-access user can read inventory reports."}
$DirectTransfer=Invoke-WebRequest "$BaseUrl/api/v1/inventory/stock-movements" -Method Post -ContentType application/json -Headers $A -Body(@{movement_type="transfer";lines=@(@{item_id=$Item.item.id;location_id=$Src.location.id;quantity_delta="-1.000"})}|ConvertTo-Json -Depth 8)-SkipHttpErrorCheck;if($DirectTransfer.StatusCode-ne400-or($DirectTransfer.Content|ConvertFrom-Json).error.code-ne"invalid_movement_type"){throw "Generic movement API accepted a document-owned transfer."}
$Audit=Scalar "SELECT set_config('app.tenant_id','$TenantAID',false);SELECT count(*) FROM audit.audit_log WHERE action IN('inventory.transfer.post','inventory.stock_count.post','inventory.reservation.create','inventory.reservation.release','inventory.cost_layer.create') AND target_id IN('$($Transfer.id)','$($Count.id)','$($Reservation.id)','$($Cost.id)');"
if($Audit-ne"5"){throw "Expected five Inventory v1 audit records."}
$Outbox=Scalar "SELECT count(*) FROM core.outbox_events WHERE tenant_id='$TenantAID' AND ((aggregate_type='inventory.transfer' AND aggregate_id='$($Transfer.id)') OR (aggregate_type='inventory.stock_count' AND aggregate_id='$($Count.id)') OR (aggregate_type='inventory.reservation' AND aggregate_id='$($Reservation.id)') OR (event_type='inventory.cost.changed.v1' AND aggregate_id='$($Item.item.id)'));"
if($Outbox-ne"5"){throw "Expected five Inventory v1 outbox records."}
Write-Host "Inventory v1 closeout verification passed."

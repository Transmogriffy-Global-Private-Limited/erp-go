param()

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

$RoutesPath = Join-Path $RepoRoot "cmd/erp-api/routes.go"
$InventorySessionPath = Join-Path $RepoRoot "cmd/erp-api/inventory_session_handler.go"
$MainPath = Join-Path $RepoRoot "cmd/erp-api/main.go"

if (-not (Test-Path $RoutesPath)) {
  throw "cmd/erp-api/routes.go is missing."
}

if (-not (Test-Path $InventorySessionPath)) {
  throw "cmd/erp-api/inventory_session_handler.go is missing."
}

$Routes = Get-Content $RoutesPath -Raw
$InventorySession = Get-Content $InventorySessionPath -Raw
$Main = Get-Content $MainPath -Raw

function Assert-Contains {
  param(
    [string] $Text,
    [string] $Pattern,
    [string] $Label
  )

  if ($Text -notmatch $Pattern) {
    throw "$Label was not found."
  }

  Write-Host "$Label verified."
}

function Assert-NotContains {
  param(
    [string] $Text,
    [string] $Pattern,
    [string] $Label
  )

  if ($Text -match $Pattern) {
    throw "$Label was found but should not exist."
  }

  Write-Host "$Label not present, as expected."
}

Write-Host "Checking ERP API route wiring..."

Assert-Contains `
  -Text $Routes `
  -Pattern 'mux\.HandleFunc\("/api/v1/auth/login",\s*a\.tenantLoginHandler\)' `
  -Label "tenant login route"

Assert-Contains `
  -Text $Routes `
  -Pattern 'mux\.HandleFunc\("/api/v1/auth/me",\s*a\.tenantMeHandler\)' `
  -Label "tenant me route"

Assert-Contains `
  -Text $Routes `
  -Pattern 'mux\.HandleFunc\("/api/v1/auth/logout",\s*a\.tenantLogoutHandler\)' `
  -Label "tenant logout route"

Assert-Contains `
  -Text $Routes `
  -Pattern 'mux\.HandleFunc\("/api/v1/inventory/items",\s*a\.inventoryItemsSessionHandler\)' `
  -Label "inventory session route"

Assert-NotContains `
  -Text $Routes `
  -Pattern 'inventoryItemsHandler' `
  -Label "direct inventory handler route"

Assert-Contains `
  -Text $Routes `
  -Pattern 'mux\.Handle\("/api/v1/purchase/suppliers",\s*a\.erpSessionMiddleware\(a\.requireModule\("purchase",\s*http\.HandlerFunc\(a\.purchaseSuppliersHandler\)\)\)\)' `
  -Label "purchase suppliers session and module-guarded route"

Assert-Contains `
  -Text $Routes `
  -Pattern 'mux\.Handle\("/api/v1/purchase/orders",\s*a\.erpSessionMiddleware\(a\.requireModule\("purchase",\s*http\.HandlerFunc\(a\.purchaseOrdersHandler\)\)\)\)' `
  -Label "purchase orders session and module-guarded route"

Assert-Contains `
  -Text $Routes `
  -Pattern 'mux\.Handle\("/api/v1/purchase/orders/",\s*a\.erpSessionMiddleware\(a\.requireModule\("purchase",\s*http\.HandlerFunc\(a\.purchaseOrderActionHandler\)\)\)\)' `
  -Label "purchase order actions session and module-guarded route"

Assert-Contains `
  -Text $Routes `
  -Pattern 'mux\.Handle\("/api/v1/purchase/receipts/",\s*a\.erpSessionMiddleware\(a\.requireModule\("purchase",\s*http\.HandlerFunc\(a\.purchaseReceiptActionHandler\)\)\)\)' `
  -Label "purchase receipt actions session and module-guarded route"

Assert-Contains `
  -Text $Routes `
  -Pattern 'mux\.Handle\("/api/v1/sales/customers",\s*a\.erpSessionMiddleware\(a\.requireModule\("sales",\s*http\.HandlerFunc\(a\.salesCustomersHandler\)\)\)\)' `
  -Label "sales customers session and module-guarded route"

Assert-Contains `
  -Text $Routes `
  -Pattern 'mux\.Handle\("/api/v1/sales/orders",\s*a\.erpSessionMiddleware\(a\.requireModule\("sales",\s*http\.HandlerFunc\(a\.salesOrdersHandler\)\)\)\)' `
  -Label "sales orders session and module-guarded route"

Assert-Contains `
  -Text $Routes `
  -Pattern 'mux\.Handle\("/api/v1/sales/orders/",\s*a\.erpSessionMiddleware\(a\.requireModule\("sales",\s*http\.HandlerFunc\(a\.salesOrderActionHandler\)\)\)\)' `
  -Label "sales order actions session and module-guarded route"

Assert-Contains `
  -Text $Routes `
  -Pattern 'mux\.Handle\("/api/v1/sales/issues",\s*a\.erpSessionMiddleware\(a\.requireModule\("sales",\s*http\.HandlerFunc\(a\.salesIssuesHandler\)\)\)\)' `
  -Label "sales issues session and module-guarded route"

Assert-Contains `
  -Text $InventorySession `
  -Pattern 'X-ERP-Session' `
  -Label "inventory session header check"

Assert-Contains `
  -Text $InventorySession `
  -Pattern 'UserFromSession' `
  -Label "inventory user resolution from ERP session"

Assert-Contains `
  -Text $InventorySession `
  -Pattern 'requireModule\("inventory"' `
  -Label "inventory module entitlement guard"

Assert-Contains `
  -Text $InventorySession `
  -Pattern 'auth\.WithUserID' `
  -Label "inventory user context injection"

Assert-Contains `
  -Text $InventorySession `
  -Pattern 'tenancy\.WithTenantID' `
  -Label "inventory tenant context injection"

Assert-NotContains `
  -Text $InventorySession `
  -Pattern 'Header\.Set\("X-User-ID"' `
  -Label "internal X-User-ID injection in inventory session handler"

Assert-NotContains `
  -Text $Main `
  -Pattern 'mux\s*:=\s*http\.NewServeMux\(\)' `
  -Label "route registration in main.go"

Write-Host ""
Write-Host "ERP route wiring verification passed."

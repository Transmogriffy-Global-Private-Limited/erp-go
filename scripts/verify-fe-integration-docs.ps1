param()

$ErrorActionPreference = "Stop"

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$GuidePath = Join-Path $RepoRoot "docs/FE_INTEGRATION_GUIDE.md"
$ErpRoutesPath = Join-Path $RepoRoot "cmd/erp-api/routes.go"
$ControlRoutesPath = Join-Path $RepoRoot "internal/platform/controlapi/server.go"

if (-not (Test-Path -LiteralPath $GuidePath)) {
  throw "docs/FE_INTEGRATION_GUIDE.md is missing."
}

$Guide = Get-Content -LiteralPath $GuidePath -Raw
$ErpRoutes = Get-Content -LiteralPath $ErpRoutesPath -Raw
$ControlRoutes = Get-Content -LiteralPath $ControlRoutesPath -Raw

function Assert-ContainsLiteral {
  param(
    [string] $Text,
    [string] $Value,
    [string] $Label
  )

  if (-not $Text.Contains($Value)) {
    throw "$Label is missing from the FE integration guide: $Value"
  }

  Write-Host "$Label documented: $Value"
}

$RoutePattern = [regex] 'mux\.Handle(?:Func)?\("([^"]+)"'
$BackendRoutes = @(
  $RoutePattern.Matches($ErpRoutes) |
    ForEach-Object { $_.Groups[1].Value }
  $RoutePattern.Matches($ControlRoutes) |
    ForEach-Object { $_.Groups[1].Value }
) | Sort-Object -Unique

if ($BackendRoutes.Count -eq 0) {
  throw "No backend routes were discovered."
}

foreach ($Route in $BackendRoutes) {
  Assert-ContainsLiteral $Guide $Route "Backend route"
}

$ActionRoutes = @(
  "/api/v1/inventory/reservations/{id}/release",
  "/api/v1/purchase/orders/{id}/approve",
  "/api/v1/purchase/receipts/{id}/reverse",
  "/api/v1/sales/orders/{id}/confirm",
  "/api/v1/sales/issues/{id}/reverse",
  "/control/v1/tenants/{tenant_id}/modules/{module_id}/enable",
  "/control/v1/tenants/{tenant_id}/modules/{module_id}/disable",
  "/control/v1/plans/{plan_id}/modules/{module_id}/enable",
  "/control/v1/tenants/{tenant_id}/subscription"
)

foreach ($Route in $ActionRoutes) {
  Assert-ContainsLiteral $Guide $Route "Dynamic action route"
}

$ErpGoText = (
  Get-ChildItem -LiteralPath (Join-Path $RepoRoot "cmd/erp-api") -Filter "*.go" |
    ForEach-Object { Get-Content -LiteralPath $_.FullName -Raw }
) -join "`n"

$PermissionPattern = [regex] 'requirePermission\(w,\s*r,\s*"([^"]+)"\)'
$Permissions = @(
  $PermissionPattern.Matches($ErpGoText) |
    ForEach-Object { $_.Groups[1].Value }
) | Sort-Object -Unique

if ($Permissions.Count -eq 0) {
  throw "No ERP permissions were discovered."
}

foreach ($Permission in $Permissions) {
  Assert-ContainsLiteral $Guide $Permission "ERP permission"
}

foreach ($Header in @("X-Tenant-ID", "X-ERP-Session", "X-Platform-Session")) {
  Assert-ContainsLiteral $Guide $Header "Authentication header"
}

foreach ($RequiredConcept in @(
  "opaque",
  "do not install CORS middleware",
  "permission_denied",
  "module_not_enabled",
  "A platform superadmin is not a tenant administrator",
  "The setup helper does not discover a tenant ID",
  '$SetupResult.tenant.id',
  '$Handoff.tenant.id',
  "A tenant ID is not a password or secret",
  "tenant_id + email",
  'Do not map an existing HRMS `Admin` row to the ERP platform superadmin',
  "Do not forward the ERP token to HRMS",
  "There is no current endpoint that returns the authenticated tenant user's complete role/permission set"
)) {
  Assert-ContainsLiteral $Guide $RequiredConcept "Integration constraint"
}

Write-Host ""
Write-Host "FE integration documentation verification passed."
Write-Host "Routes covered: $($BackendRoutes.Count)"
Write-Host "Dynamic actions covered: $($ActionRoutes.Count)"
Write-Host "Permissions covered: $($Permissions.Count)"

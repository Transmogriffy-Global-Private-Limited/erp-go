param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

& (Join-Path $PSScriptRoot "ensure-local-apis.ps1") `
  -BaseUrl $BaseUrl `
  -ControlPlaneUrl $ControlPlaneUrl `
  -EnvFile $EnvFile

function Invoke-Step {
  param(
    [string] $Name,
    [scriptblock] $Action
  )

  Write-Host ""
  Write-Host "===== $Name ====="

  try {
    & $Action
    Write-Host "PASS: $Name"
  }
  catch {
    Write-Host "FAIL: $Name"
    throw
  }
}

Invoke-Step -Name "Go tests" -Action {
  go test ./...
  if ($LASTEXITCODE -ne 0) {
    throw "go test ./... failed."
  }
}
Invoke-Step -Name "ERP route wiring verification" -Action {
  & (Join-Path $PSScriptRoot "verify-erp-route-wiring.ps1")
}
Invoke-Step -Name "Server shutdown wiring verification" -Action {
  & (Join-Path $PSScriptRoot "verify-server-shutdown-wiring.ps1")
}
Invoke-Step -Name "FE local setup helper verification" -Action {
  & (Join-Path $PSScriptRoot "verify-fe-dev-setup.ps1")
}
Invoke-Step -Name "FE integration documentation verification" -Action {
  & (Join-Path $PSScriptRoot "verify-fe-integration-docs.ps1")
}

Invoke-Step -Name "Database connectivity" -Action {
  & (Join-Path $PSScriptRoot "db-check.ps1") -EnvFile $EnvFile
}

Invoke-Step -Name "Platform superadmin seed" -Action {
  & (Join-Path $PSScriptRoot "seed-platform-superadmin.ps1") -EnvFile $EnvFile
}

Invoke-Step -Name "Control plane health" -Action {
  $health = Invoke-RestMethod "$ControlPlaneUrl/healthz"
  if ($health.status -ne "ok") {
    throw "control-plane-api /healthz did not return ok."
  }

  $dbHealth = Invoke-RestMethod "$ControlPlaneUrl/healthz/db"
  if ($dbHealth.status -ne "db_ok") {
    throw "control-plane-api /healthz/db did not return db_ok."
  }
}

Invoke-Step -Name "ERP API health" -Action {
  $health = Invoke-RestMethod "$BaseUrl/healthz"
  if ($health.status -ne "ok") {
    throw "erp-api /healthz did not return ok."
  }

  $dbHealth = Invoke-RestMethod "$BaseUrl/healthz/db"
  if ($dbHealth.status -ne "db_ok") {
    throw "erp-api /healthz/db did not return db_ok."
  }
}
Invoke-Step -Name "Tenant session verification" -Action {
  & (Join-Path $PSScriptRoot "verify-tenant-session.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}
Invoke-Step -Name "Inventory ERP session auth verification" -Action {
  & (Join-Path $PSScriptRoot "verify-inventory-session-auth.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}
Invoke-Step -Name "Inventory units verification" -Action {
  & (Join-Path $PSScriptRoot "verify-inventory-units.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}
Invoke-Step -Name "Inventory item/base unit verification" -Action {
  & (Join-Path $PSScriptRoot "verify-inventory-item-units.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}


Invoke-Step -Name "Inventory locations verification" -Action {
  & (Join-Path $PSScriptRoot "verify-inventory-locations.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "Inventory stock verification" -Action {
  & (Join-Path $PSScriptRoot "verify-inventory-stock.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "Purchase suppliers verification" -Action {
  & (Join-Path $PSScriptRoot "verify-purchase-suppliers.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "Purchase orders verification" -Action {
  & (Join-Path $PSScriptRoot "verify-purchase-orders.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "Purchase receipts verification" -Action {
  & (Join-Path $PSScriptRoot "verify-purchase-receipts.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}
Invoke-Step -Name "Sales customers verification" -Action {
  & (Join-Path $PSScriptRoot "verify-sales-customers.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}
Invoke-Step -Name "Inventory v1 closeout verification" -Action {
  & (Join-Path $PSScriptRoot "verify-inventory-v1.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}
Invoke-Step -Name "Sales orders verification" -Action {
  & (Join-Path $PSScriptRoot "verify-sales-orders.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}
Invoke-Step -Name "Sales issues verification" -Action {
  & (Join-Path $PSScriptRoot "verify-sales-issues.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}
Invoke-Step -Name "Control-plane auth verification" -Action {
  & (Join-Path $PSScriptRoot "verify-control-plane-auth.ps1") `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "Platform session verification" -Action {
  & (Join-Path $PSScriptRoot "verify-platform-session.ps1") `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "Platform logout verification" -Action {
  & (Join-Path $PSScriptRoot "verify-platform-logout.ps1") `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "Platform auth audit verification" -Action {
  & (Join-Path $PSScriptRoot "verify-platform-auth-audit.ps1") `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "Platform login security verification" -Action {
  & (Join-Path $PSScriptRoot "verify-platform-login-security.ps1") `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "Platform session management verification" -Action {
  & (Join-Path $PSScriptRoot "verify-platform-session-management.ps1") `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "Plans/subscriptions verification" -Action {
  & (Join-Path $PSScriptRoot "verify-plans-subscriptions.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "Control-plane audit verification" -Action {
  & (Join-Path $PSScriptRoot "verify-control-plane-audit.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "RBAC verification" -Action {
  & (Join-Path $PSScriptRoot "verify-rbac.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "Tenant isolation verification" -Action {
  & (Join-Path $PSScriptRoot "verify-tenant-isolation.ps1") `
    -BaseUrl $BaseUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "Module entitlement verification" -Action {
  & (Join-Path $PSScriptRoot "verify-module-entitlement.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "Audit/outbox verification" -Action {
  & (Join-Path $PSScriptRoot "verify-audit-outbox.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Invoke-Step -Name "Outbox worker verification" -Action {
  & (Join-Path $PSScriptRoot "verify-outbox-worker.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile
}

Write-Host ""
Write-Host "All verification checks passed."

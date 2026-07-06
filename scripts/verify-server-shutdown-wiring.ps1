$ErrorActionPreference = "Stop"
$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")

function Assert-Contains {
  param([string] $Text, [string] $Pattern, [string] $Label)
  if ($Text -notmatch $Pattern) { throw "$Label is missing." }
  Write-Host "$Label verified."
}

Write-Host "Checking local server shutdown wiring..."
$ControlMain = Get-Content (Join-Path $RepoRoot "cmd/control-plane-api/main.go") -Raw
$ErpMain = Get-Content (Join-Path $RepoRoot "cmd/erp-api/main.go") -Raw
$WorkerMain = Get-Content (Join-Path $RepoRoot "cmd/erp-worker/main.go") -Raw

foreach ($Api in @(
  @{ Name = "control-plane-api"; Text = $ControlMain },
  @{ Name = "erp-api"; Text = $ErpMain }
)) {
  Assert-Contains $Api.Text 'signal\.NotifyContext' "$($Api.Name) interrupt subscription"
  Assert-Contains $Api.Text '<-ctx\.Done\(\)' "$($Api.Name) interrupt handling"
  Assert-Contains $Api.Text 'server\.Shutdown\(shutdownCtx\)' "$($Api.Name) graceful HTTP shutdown"
  Assert-Contains $Api.Text 'context\.WithTimeout\(context\.Background\(\),\s*10\s*\*\s*time\.Second\)' "$($Api.Name) shutdown timeout"
}

Assert-Contains $WorkerMain 'case\s+<-ctx\.Done\(\)' "erp-worker interrupt handling"

foreach ($ScriptName in @("run-control-plane-api.ps1", "run-erp-api.ps1", "run-erp-worker.ps1")) {
  $Script = Get-Content (Join-Path $PSScriptRoot $ScriptName) -Raw
  Assert-Contains $Script 'go build -o \$Binary' "$ScriptName direct binary build"
  if ($Script -match 'go run') { throw "$ScriptName still uses go run." }
  Write-Host "$ScriptName has no go-run wrapper."
}

Write-Host "Local server shutdown wiring verification passed."

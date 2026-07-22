param()

$ErrorActionPreference = "Stop"

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$SetupPath = Join-Path $PSScriptRoot "setup-fe-dev.ps1"
$MigrationPath = Join-Path $PSScriptRoot "apply-migration.ps1"
$RestartPath = Join-Path $PSScriptRoot "restart-local-apis.ps1"
$EnsurePath = Join-Path $PSScriptRoot "ensure-local-apis.ps1"

function Assert-Contains {
  param(
    [string] $Text,
    [string] $Pattern,
    [string] $Label
  )

  if ($Text -notmatch $Pattern) {
    throw "$Label is missing."
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

function Assert-PowerShellParses {
  param(
    [string] $Path,
    [string] $Label
  )

  $tokens = $null
  $errors = $null
  [System.Management.Automation.Language.Parser]::ParseFile(
    $Path,
    [ref] $tokens,
    [ref] $errors
  ) | Out-Null

  if ($errors.Count -gt 0) {
    $messages = $errors | ForEach-Object { $_.Message }
    throw "$Label has PowerShell parser errors: $($messages -join '; ')"
  }

  Write-Host "$Label parses successfully."
}

foreach ($Path in @($SetupPath, $MigrationPath, $RestartPath, $EnsurePath)) {
  if (-not (Test-Path -LiteralPath $Path)) {
    throw "Required script is missing: $Path"
  }
}

Write-Host "Checking FE local setup helper..."

Assert-PowerShellParses -Path $SetupPath -Label "setup-fe-dev.ps1"
Assert-PowerShellParses -Path $MigrationPath -Label "apply-migration.ps1"
Assert-PowerShellParses -Path $RestartPath -Label "restart-local-apis.ps1"
Assert-PowerShellParses -Path $EnsurePath -Label "ensure-local-apis.ps1"

$Setup = Get-Content -LiteralPath $SetupPath -Raw
$Migration = Get-Content -LiteralPath $MigrationPath -Raw
$Restart = Get-Content -LiteralPath $RestartPath -Raw
$Ensure = Get-Content -LiteralPath $EnsurePath -Raw

Assert-Contains $Setup 'db-check\.ps1' "database preflight"
Assert-Contains $Setup 'Get-ChildItem[^\r\n]+migrations[^\r\n]+\*\.up\.sql' "ordered migration discovery"
Assert-Contains $Setup 'apply-migration\.ps1' "migration application"
Assert-Contains $Setup 'seed-platform-superadmin\.ps1' "platform superadmin seed"
Assert-Contains $Setup 'seed-dev-tenant\.ps1' "tenant seed"
Assert-Contains $Setup 'seed-dev-rbac\.ps1' "tenant RBAC seed"
Assert-Contains $Setup 'restart-local-apis\.ps1' "API restart"
Assert-Contains $Setup '/api/v1/auth/login' "tenant login validation"
Assert-Contains $Setup '/api/v1/auth/me' "tenant identity validation"
Assert-Contains $Setup '/api/v1/modules' "tenant module validation"
Assert-Contains $Setup '/control/v1/auth/login' "platform login validation"
Assert-Contains $Setup '/control/v1/modules' "platform module validation"
Assert-Contains $Setup '\.local/fe-integration\.json' "gitignored manifest default"
Assert-Contains $Setup 'X-Tenant-ID' "tenant header contract"
Assert-Contains $Setup 'X-ERP-Session' "ERP session header contract"
Assert-Contains $Setup 'X-Platform-Session' "platform session header contract"
Assert-Contains $Setup 'IncludeSessionTokens' "optional session-token handoff"
Assert-Contains $Setup 'Invoke-RestMethod[^\r\n]+/auth/logout' "default validation-session cleanup"

Assert-NotContains $Migration '\bexit\s+0\b' "caller-terminating idempotent migration exit"
Assert-Contains $Migration 'Migration already applied:[^\r\n]+\r?\n\s+return' "composable already-applied migration return"
Assert-Contains $Restart '\[string\]\s+\$EnvFile\s*=\s*"\.env"' "restart environment-file parameter"
Assert-Contains `
  $Restart `
  ([regex]::Escape('-EnvFile ''$escapedEnvFile''')) `
  "restart environment forwarding"
Assert-Contains $Ensure '-EnvFile\s+\$EnvFile' "ensure environment forwarding"

$EnsureCallerCount = 0
foreach ($VerifierPath in Get-ChildItem -LiteralPath $PSScriptRoot -Filter "verify-*.ps1") {
  $Lines = @(Get-Content -LiteralPath $VerifierPath.FullName)

  for ($LineIndex = 0; $LineIndex -lt $Lines.Count; $LineIndex++) {
    if ($Lines[$LineIndex] -notmatch '^\s*&.*ensure-local-apis\.ps1') {
      continue
    }

    $EnsureCallerCount++
    $EndIndex = [Math]::Min($LineIndex + 4, $Lines.Count - 1)
    $CallText = ($Lines[$LineIndex..$EndIndex] -join "`n")

    if ($CallText -notmatch '-EnvFile\s+\$EnvFile') {
      throw "$($VerifierPath.Name) does not forward EnvFile to ensure-local-apis.ps1."
    }
  }
}

if ($EnsureCallerCount -eq 0) {
  throw "No focused verifier callers of ensure-local-apis.ps1 were found."
}

Write-Host "$EnsureCallerCount focused verifier API-start calls forward EnvFile."

$GitIgnore = Get-Content -LiteralPath (Join-Path $RepoRoot ".gitignore") -Raw
Assert-Contains $GitIgnore '(?m)^\.local/' "local credential artifact ignore rule"

Write-Host "FE local setup helper verification passed."

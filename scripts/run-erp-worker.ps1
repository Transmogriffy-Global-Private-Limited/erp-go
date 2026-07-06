param(
  [string] $EnvFile = ".env",
  [switch] $Once,
  [int] $Limit = 10
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

. (Join-Path $PSScriptRoot "Import-DotEnv.ps1") -Path (Join-Path $RepoRoot $EnvFile)

$Args = @()

if ($Once) {
  $Args += "-once"
}

$Args += "-limit"
$Args += "$Limit"

$BinDir = Join-Path $RepoRoot ".local/bin"
$Binary = Join-Path $BinDir "erp-worker.exe"
New-Item -ItemType Directory -Path $BinDir -Force | Out-Null

go build -o $Binary ./cmd/erp-worker
if ($LASTEXITCODE -ne 0) {
  throw "Failed to build erp-worker."
}

if (-not $Once) {
  Write-Host "erp-worker runs in this window. Press Ctrl+C to stop it."
}
& $Binary @Args
$WorkerExitCode = $LASTEXITCODE

if ($WorkerExitCode -ne 0) {
  throw "ERP worker failed with exit code $WorkerExitCode."
}

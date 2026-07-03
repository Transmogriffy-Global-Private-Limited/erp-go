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

go run ./cmd/erp-worker @Args
$WorkerExitCode = $LASTEXITCODE

if ($WorkerExitCode -ne 0) {
  throw "ERP worker failed with exit code $WorkerExitCode."
}

param(
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

. (Join-Path $PSScriptRoot "Import-DotEnv.ps1") -Path (Join-Path $RepoRoot $EnvFile)

$BinDir = Join-Path $RepoRoot ".local/bin"
$Binary = Join-Path $BinDir "erp-api.exe"
New-Item -ItemType Directory -Path $BinDir -Force | Out-Null

go build -o $Binary ./cmd/erp-api
if ($LASTEXITCODE -ne 0) {
  throw "Failed to build erp-api."
}

Write-Host "erp-api runs in this window. Press Ctrl+C to stop it."
& $Binary
exit $LASTEXITCODE

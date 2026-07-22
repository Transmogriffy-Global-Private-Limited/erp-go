param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

if ($env:ERP_GO_LOCAL_APIS_RESTARTED -eq "1") {
  Write-Host "Local APIs already restarted in this verification process."
  return
}

& (Join-Path $PSScriptRoot "restart-local-apis.ps1") `
  -BaseUrl $BaseUrl `
  -ControlPlaneUrl $ControlPlaneUrl `
  -EnvFile $EnvFile

$env:ERP_GO_LOCAL_APIS_RESTARTED = "1"

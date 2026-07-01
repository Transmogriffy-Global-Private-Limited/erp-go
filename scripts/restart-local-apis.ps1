param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [int] $ErpPort = 8080,
  [int] $ControlPlanePort = 8081,
  [int] $TimeoutSeconds = 30
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

Write-Host "Preflight: go test ./..."
go test ./...
if ($LASTEXITCODE -ne 0) {
  throw "Go tests failed. Not restarting local APIs."
}

function Stop-PortProcess {
  param(
    [int] $Port,
    [string] $Name
  )

  $connections = @(Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue)

  foreach ($connection in $connections) {
    $pidToStop = $connection.OwningProcess

    if ($pidToStop -and $pidToStop -ne $PID) {
      $process = Get-Process -Id $pidToStop -ErrorAction SilentlyContinue

      if ($process) {
        Write-Host "Stopping $Name on port $Port. PID=$pidToStop Process=$($process.ProcessName)"
        Stop-Process -Id $pidToStop -Force
      }
    }
  }
}

function Wait-Health {
  param(
    [string] $Url,
    [string] $Name,
    [int] $TimeoutSeconds
  )

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)

  while ((Get-Date) -lt $deadline) {
    try {
      $health = Invoke-RestMethod $Url -TimeoutSec 2
      if ($health.status -eq "ok" -or $health.status -eq "db_ok") {
        Write-Host "$Name healthy: $Url"
        return
      }
    }
    catch {
      Start-Sleep -Milliseconds 700
    }
  }

  throw "$Name did not become healthy before timeout: $Url"
}

function Start-ApiWindow {
  param(
    [string] $Title,
    [string] $ScriptPath
  )

  $pwshPath = (Get-Process -Id $PID).Path

  Start-Process `
    -FilePath $pwshPath `
    -WorkingDirectory $RepoRoot `
    -ArgumentList @(
      "-NoExit",
      "-ExecutionPolicy", "Bypass",
      "-Command",
      "`$Host.UI.RawUI.WindowTitle = '$Title'; cd '$RepoRoot'; & '$ScriptPath'"
    ) | Out-Null
}

Write-Host "Stopping stale local APIs..."
Stop-PortProcess -Port $ControlPlanePort -Name "control-plane-api"
Stop-PortProcess -Port $ErpPort -Name "erp-api"

Start-Sleep -Seconds 1

Write-Host "Starting fresh control-plane-api..."
Start-ApiWindow -Title "erp-go control-plane-api" -ScriptPath (Join-Path $RepoRoot "scripts/run-control-plane-api.ps1")

Write-Host "Starting fresh erp-api..."
Start-ApiWindow -Title "erp-go erp-api" -ScriptPath (Join-Path $RepoRoot "scripts/run-erp-api.ps1")

Write-Host "Waiting for APIs..."
Wait-Health -Url "$ControlPlaneUrl/healthz" -Name "control-plane-api" -TimeoutSeconds $TimeoutSeconds
Wait-Health -Url "$ControlPlaneUrl/healthz/db" -Name "control-plane-api DB" -TimeoutSeconds $TimeoutSeconds
Wait-Health -Url "$BaseUrl/healthz" -Name "erp-api" -TimeoutSeconds $TimeoutSeconds
Wait-Health -Url "$BaseUrl/healthz/db" -Name "erp-api DB" -TimeoutSeconds $TimeoutSeconds

$env:ERP_GO_LOCAL_APIS_RESTARTED = "1"

Write-Host ""
Write-Host "Local APIs restarted and healthy."


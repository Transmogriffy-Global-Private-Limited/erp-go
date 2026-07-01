param(
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

function Assert-Status {
  param(
    [scriptblock] $Action,
    [int] $ExpectedStatus,
    [string] $Label
  )

  $response = & $Action

  if ($response.StatusCode -ne $ExpectedStatus) {
    throw "$Label expected HTTP $ExpectedStatus, got HTTP $($response.StatusCode). Body: $($response.Content)"
  }

  Write-Host "$Label returned HTTP $ExpectedStatus as expected."
}

Write-Host "Creating platform session..."
$SessionHeaders = & (Join-Path $PSScriptRoot "Get-PlatformSessionHeaders.ps1") `
  -ControlPlaneUrl $ControlPlaneUrl `
  -EnvFile $EnvFile

$BadSessionHeaders = @{
  "X-Platform-Session" = "not-a-valid-session-token"
}

$OldFallbackHeaders = @{
  "X-Platform-User-ID" = "00000000-0000-0000-0000-00000000aaaa"
}

Write-Host ""
Write-Host "Checking health remains public..."
$Health = Invoke-RestMethod "$ControlPlaneUrl/healthz"
if ($Health.status -ne "ok") {
  throw "Expected public /healthz status ok."
}

Write-Host "Health is public."

Write-Host ""
Write-Host "Checking control endpoint without platform auth..."
Assert-Status -ExpectedStatus 401 -Label "No platform auth" -Action {
  Invoke-WebRequest "$ControlPlaneUrl/control/v1/modules" -SkipHttpErrorCheck
}

Write-Host ""
Write-Host "Checking old X-Platform-User-ID fallback is rejected..."
Assert-Status -ExpectedStatus 401 -Label "Old platform user fallback" -Action {
  Invoke-WebRequest "$ControlPlaneUrl/control/v1/modules" -Headers $OldFallbackHeaders -SkipHttpErrorCheck
}

Write-Host ""
Write-Host "Checking control endpoint with invalid session..."
Assert-Status -ExpectedStatus 403 -Label "Invalid platform session" -Action {
  Invoke-WebRequest "$ControlPlaneUrl/control/v1/modules" -Headers $BadSessionHeaders -SkipHttpErrorCheck
}

Write-Host ""
Write-Host "Checking control endpoint with DB-backed session..."
Assert-Status -ExpectedStatus 200 -Label "DB-backed platform session" -Action {
  Invoke-WebRequest "$ControlPlaneUrl/control/v1/modules" -Headers $SessionHeaders -SkipHttpErrorCheck
}

Write-Host ""
Write-Host "Control-plane auth verification passed."

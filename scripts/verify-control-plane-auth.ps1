param(
  [string] $ControlPlaneUrl = "http://localhost:8081"
)

$ErrorActionPreference = "Stop"

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

$SuperadminHeaders = @{
  "X-Platform-User-ID" = "00000000-0000-0000-0000-00000000aaaa"
  "X-Platform-Role" = "superadmin"
}

$WrongRoleHeaders = @{
  "X-Platform-User-ID" = "00000000-0000-0000-0000-00000000bbbb"
  "X-Platform-Role" = "viewer"
}

Write-Host "Checking health remains public..."
$Health = Invoke-RestMethod "$ControlPlaneUrl/healthz"
if ($Health.status -ne "ok") {
  throw "Expected public /healthz status ok."
}

Write-Host "Health is public."

Write-Host ""
Write-Host "Checking control endpoint without platform user..."
Assert-Status -ExpectedStatus 401 -Label "No platform user" -Action {
  Invoke-WebRequest "$ControlPlaneUrl/control/v1/modules" -SkipHttpErrorCheck
}

Write-Host ""
Write-Host "Checking control endpoint with wrong role..."
Assert-Status -ExpectedStatus 403 -Label "Wrong platform role" -Action {
  Invoke-WebRequest "$ControlPlaneUrl/control/v1/modules" -Headers $WrongRoleHeaders -SkipHttpErrorCheck
}

Write-Host ""
Write-Host "Checking control endpoint with superadmin..."
Assert-Status -ExpectedStatus 200 -Label "Superadmin" -Action {
  Invoke-WebRequest "$ControlPlaneUrl/control/v1/modules" -Headers $SuperadminHeaders -SkipHttpErrorCheck
}

Write-Host ""
Write-Host "Control-plane auth verification passed."

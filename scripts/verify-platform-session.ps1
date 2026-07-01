param(
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $Email = "dev.superadmin@example.test",
  [string] $Password = "dev-superadmin-password",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

Write-Host "Seeding platform superadmin..."
& (Join-Path $PSScriptRoot "seed-platform-superadmin.ps1") `
  -Email $Email `
  -Password $Password `
  -EnvFile $EnvFile

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

$LoginBody = @{
  email = $Email
  password = $Password
} | ConvertTo-Json

Write-Host ""
Write-Host "Logging in platform superadmin..."
$Login = Invoke-RestMethod "$ControlPlaneUrl/control/v1/auth/login" `
  -Method Post `
  -ContentType "application/json" `
  -Body $LoginBody

$SessionToken = $Login.session_token

if (-not $SessionToken) {
  throw "Login did not return session_token."
}

Write-Host "Login returned session token."

$SessionHeaders = @{
  "X-Platform-Session" = $SessionToken
}

Write-Host ""
Write-Host "Checking control endpoint with session..."
Assert-Status -ExpectedStatus 200 -Label "Platform session" -Action {
  Invoke-WebRequest "$ControlPlaneUrl/control/v1/modules" `
    -Headers $SessionHeaders `
    -SkipHttpErrorCheck
}

Write-Host ""
Write-Host "Checking bad login..."
$BadLoginBody = @{
  email = $Email
  password = "wrong-password"
} | ConvertTo-Json

Assert-Status -ExpectedStatus 401 -Label "Bad platform login" -Action {
  Invoke-WebRequest "$ControlPlaneUrl/control/v1/auth/login" `
    -Method Post `
    -ContentType "application/json" `
    -Body $BadLoginBody `
    -SkipHttpErrorCheck
}

Write-Host ""
Write-Host "Platform session verification passed."

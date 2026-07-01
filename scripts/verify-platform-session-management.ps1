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

function Login-Platform {
  $LoginBody = @{
    email = $Email
    password = $Password
  } | ConvertTo-Json

  $Login = Invoke-RestMethod "$ControlPlaneUrl/control/v1/auth/login" `
    -Method Post `
    -ContentType "application/json" `
    -Body $LoginBody

  if (-not $Login.session_token) {
    throw "Login did not return session_token."
  }

  return $Login.session_token
}

Write-Host ""
Write-Host "Creating two sessions..."
$TokenA = Login-Platform
$TokenB = Login-Platform

$HeadersA = @{
  "X-Platform-Session" = $TokenA
}

$HeadersB = @{
  "X-Platform-Session" = $TokenB
}

Write-Host "Two sessions created."

Write-Host ""
Write-Host "Listing active sessions..."
$Sessions = Invoke-RestMethod "$ControlPlaneUrl/control/v1/auth/sessions" `
  -Headers $HeadersA

if (@($Sessions.sessions).Count -lt 2) {
  throw "Expected at least two active sessions, got: $(@($Sessions.sessions).Count)"
}

Write-Host "Active sessions listed."

Write-Host ""
Write-Host "Cleaning expired sessions..."
$Cleanup = Invoke-RestMethod "$ControlPlaneUrl/control/v1/auth/sessions/cleanup-expired" `
  -Method Post `
  -Headers $HeadersA

if ($null -eq $Cleanup.revoked_count) {
  throw "cleanup-expired did not return revoked_count."
}

Write-Host "Expired session cleanup returned revoked_count=$($Cleanup.revoked_count)."

Write-Host ""
Write-Host "Revoking all sessions..."
$RevokeAll = Invoke-RestMethod "$ControlPlaneUrl/control/v1/auth/sessions/revoke-all" `
  -Method Post `
  -Headers $HeadersA

if ([int]$RevokeAll.revoked_count -lt 2) {
  throw "Expected revoke-all to revoke at least two sessions, got: $($RevokeAll.revoked_count)"
}

Write-Host "Revoke-all revoked sessions."

Write-Host ""
Write-Host "Checking session A rejected after revoke-all..."
Assert-Status -ExpectedStatus 403 -Label "Session A after revoke-all" -Action {
  Invoke-WebRequest "$ControlPlaneUrl/control/v1/modules" `
    -Headers $HeadersA `
    -SkipHttpErrorCheck
}

Write-Host ""
Write-Host "Checking session B rejected after revoke-all..."
Assert-Status -ExpectedStatus 403 -Label "Session B after revoke-all" -Action {
  Invoke-WebRequest "$ControlPlaneUrl/control/v1/modules" `
    -Headers $HeadersB `
    -SkipHttpErrorCheck
}

Write-Host ""
Write-Host "Platform session management verification passed."

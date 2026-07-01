param(
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $Email = "dev.superadmin@example.test",
  [string] $Password = "dev-superadmin-password",
  [string] $PlatformUserID = "00000000-0000-0000-0000-00000000aaaa",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

. (Join-Path $PSScriptRoot "Import-DotEnv.ps1") -Path (Join-Path $RepoRoot $EnvFile)

if (-not $env:MIGRATION_DATABASE_URL) {
  throw "MIGRATION_DATABASE_URL is missing. Copy .env.example to .env and edit it."
}

Write-Host "Seeding platform superadmin..."
& (Join-Path $PSScriptRoot "seed-platform-superadmin.ps1") `
  -Email $Email `
  -Password $Password `
  -PlatformUserID $PlatformUserID `
  -EnvFile $EnvFile

function Invoke-ScalarSql {
  param(
    [string] $Sql
  )

  $Output = @(psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -At -c $Sql)

  if ($LASTEXITCODE -ne 0) {
    throw "SQL query failed."
  }

  return (($Output | Select-Object -Last 1) -join "").Trim()
}

function Count-PlatformAudit {
  param(
    [string] $Action
  )

  $Sql = @"
SELECT count(*)
FROM control.platform_audit_log
WHERE platform_actor_id = '$PlatformUserID'
  AND action = '$Action';
"@

  return [int](Invoke-ScalarSql -Sql $Sql)
}

$LoginCountBefore = Count-PlatformAudit -Action "control.platform_auth.login"
$LogoutCountBefore = Count-PlatformAudit -Action "control.platform_auth.logout"

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

$LoginCountAfter = Count-PlatformAudit -Action "control.platform_auth.login"

if ($LoginCountAfter -le $LoginCountBefore) {
  throw "Expected platform login audit count to increase. before=$LoginCountBefore after=$LoginCountAfter"
}

Write-Host "Platform login audit row verified."

$SessionHeaders = @{
  "X-Platform-Session" = $SessionToken
}

Write-Host ""
Write-Host "Logging out platform session..."
$Logout = Invoke-RestMethod "$ControlPlaneUrl/control/v1/auth/logout" `
  -Method Post `
  -Headers $SessionHeaders

if ($Logout.logged_out -ne $true) {
  throw "Logout did not return logged_out=true."
}

$LogoutCountAfter = Count-PlatformAudit -Action "control.platform_auth.logout"

if ($LogoutCountAfter -le $LogoutCountBefore) {
  throw "Expected platform logout audit count to increase. before=$LogoutCountBefore after=$LogoutCountAfter"
}

Write-Host "Platform logout audit row verified."

Write-Host ""
Write-Host "Platform auth audit verification passed."

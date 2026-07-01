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

Write-Host "Seeding unlocked platform superadmin..."
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

$FailedAuditBefore = Count-PlatformAudit -Action "control.platform_auth.login_failed"

$BadLoginBody = @{
  email = $Email
  password = "wrong-password"
} | ConvertTo-Json

Write-Host ""
Write-Host "Running four bad login attempts. Expecting 401..."
for ($i = 1; $i -le 4; $i++) {
  Assert-Status -ExpectedStatus 401 -Label "Bad login attempt $i" -Action {
    Invoke-WebRequest "$ControlPlaneUrl/control/v1/auth/login" `
      -Method Post `
      -ContentType "application/json" `
      -Body $BadLoginBody `
      -SkipHttpErrorCheck
  }
}

Write-Host ""
Write-Host "Running fifth bad login attempt. Expecting lock..."
Assert-Status -ExpectedStatus 423 -Label "Bad login lock attempt" -Action {
  Invoke-WebRequest "$ControlPlaneUrl/control/v1/auth/login" `
    -Method Post `
    -ContentType "application/json" `
    -Body $BadLoginBody `
    -SkipHttpErrorCheck
}

$FailedAuditAfter = Count-PlatformAudit -Action "control.platform_auth.login_failed"

if (($FailedAuditAfter - $FailedAuditBefore) -lt 5) {
  throw "Expected at least 5 failed login audit rows. before=$FailedAuditBefore after=$FailedAuditAfter"
}

Write-Host "Failed login audit rows verified."

$LockSql = @"
SELECT
  failed_login_count,
  locked_until IS NOT NULL AND locked_until > now()
FROM control.platform_users
WHERE id = '$PlatformUserID';
"@

$LockState = Invoke-ScalarSql -Sql $LockSql
$Parts = $LockState -split "\|"

if ([int]$Parts[0] -lt 5) {
  throw "Expected failed_login_count >= 5, got: $($Parts[0])"
}

if ($Parts[1] -ne "t") {
  throw "Expected platform user to be locked, got locked=$($Parts[1])"
}

Write-Host "Platform lock state verified."

$GoodLoginBody = @{
  email = $Email
  password = $Password
} | ConvertTo-Json

Write-Host ""
Write-Host "Checking good login while locked. Expecting 423..."
Assert-Status -ExpectedStatus 423 -Label "Good login while locked" -Action {
  Invoke-WebRequest "$ControlPlaneUrl/control/v1/auth/login" `
    -Method Post `
    -ContentType "application/json" `
    -Body $GoodLoginBody `
    -SkipHttpErrorCheck
}

Write-Host ""
Write-Host "Reseeding platform superadmin to clear local lockout..."
& (Join-Path $PSScriptRoot "seed-platform-superadmin.ps1") `
  -Email $Email `
  -Password $Password `
  -PlatformUserID $PlatformUserID `
  -EnvFile $EnvFile | Out-Null

Write-Host "Checking good login after reset. Expecting 200..."
Assert-Status -ExpectedStatus 200 -Label "Good login after reset" -Action {
  Invoke-WebRequest "$ControlPlaneUrl/control/v1/auth/login" `
    -Method Post `
    -ContentType "application/json" `
    -Body $GoodLoginBody `
    -SkipHttpErrorCheck
}

Write-Host ""
Write-Host "Platform failed login lockout verification passed."

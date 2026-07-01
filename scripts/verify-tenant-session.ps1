param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $TenantID = "00000000-0000-0000-0000-000000000001",
  [string] $AllowedUserID = "11111111-1111-1111-1111-111111111111",
  [string] $Password = "dev-tenant-password",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

Write-Host "Seeding tenant..."
& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") -TenantID $TenantID -EnvFile $EnvFile

Write-Host ""
Write-Host "Seeding RBAC and password..."
& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") `
  -TenantID $TenantID `
  -AllowedUserID $AllowedUserID `
  -AllowedPassword $Password `
  -EnvFile $EnvFile

$Email = "inventory.allowed+$($TenantID.Substring(0, 8))@example.test"

$LoginBody = @{
  tenant_id = $TenantID
  email = $Email
  password = $Password
} | ConvertTo-Json

Write-Host ""
Write-Host "Logging in tenant user..."
$Login = Invoke-RestMethod "$BaseUrl/api/v1/auth/login" `
  -Method Post `
  -ContentType "application/json" `
  -Body $LoginBody

if (-not $Login.session_token) {
  throw "Tenant login did not return session_token."
}

if ($Login.user_id -ne $AllowedUserID) {
  throw "Tenant login returned wrong user_id. got=$($Login.user_id) expected=$AllowedUserID"
}

Write-Host "Tenant login returned session token."

$SessionHeaders = @{
  "X-Tenant-ID" = $TenantID
  "X-ERP-Session" = $Login.session_token
}

Write-Host ""
Write-Host "Checking /api/v1/auth/me..."
$Me = Invoke-RestMethod "$BaseUrl/api/v1/auth/me" -Headers $SessionHeaders

if ($Me.user.user_id -ne $AllowedUserID) {
  throw "Me endpoint returned wrong user_id. got=$($Me.user.user_id) expected=$AllowedUserID"
}

Write-Host "ERP session me endpoint verified."

Write-Host ""
Write-Host "Logging out tenant user..."
$Logout = Invoke-RestMethod "$BaseUrl/api/v1/auth/logout" `
  -Method Post `
  -Headers $SessionHeaders

if ($Logout.logged_out -ne $true) {
  throw "Tenant logout did not return logged_out=true."
}

Write-Host "Tenant logout verified."

Write-Host ""
Write-Host "Checking session rejected after logout..."
try {
  Invoke-RestMethod "$BaseUrl/api/v1/auth/me" -Headers $SessionHeaders | Out-Null
  throw "ERP session worked after logout."
}
catch {
  $statusCode = $null

  if ($_.Exception.Response -and $_.Exception.Response.StatusCode) {
    $statusCode = [int]$_.Exception.Response.StatusCode
  }

  if ($statusCode -ne 403) {
    throw "Expected 403 after logout, got: $statusCode"
  }

  Write-Host "ERP session rejected after logout."
}

Write-Host ""
Write-Host "ERP tenant session verification passed."

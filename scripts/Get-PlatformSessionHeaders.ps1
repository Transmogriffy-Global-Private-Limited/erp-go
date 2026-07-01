param(
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $Email = "dev.superadmin@example.test",
  [string] $Password = "dev-superadmin-password",
  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

& (Join-Path $PSScriptRoot "seed-platform-superadmin.ps1") `
  -Email $Email `
  -Password $Password `
  -EnvFile $EnvFile | Out-Null

$LoginBody = @{
  email = $Email
  password = $Password
} | ConvertTo-Json

$Login = Invoke-RestMethod "$ControlPlaneUrl/control/v1/auth/login" `
  -Method Post `
  -ContentType "application/json" `
  -Body $LoginBody

if (-not $Login.session_token) {
  throw "Platform login did not return session_token."
}

return @{
  "X-Platform-Session" = $Login.session_token
}

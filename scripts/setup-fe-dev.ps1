param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $ControlPlaneUrl = "http://localhost:8081",
  [string] $EnvFile = ".env",
  [string] $OutputPath = ".local/fe-integration.json",

  [string] $TenantID = "00000000-0000-0000-0000-000000000001",
  [string] $TenantSlug = "dev-tenant",
  [string] $TenantLegalName = "Dev Tenant Private Limited",
  [string] $TenantDisplayName = "Dev Tenant",

  [string] $AllowedUserID = "11111111-1111-1111-1111-111111111111",
  [string] $AllowedPassword = "dev-tenant-password",
  [string] $NoAccessUserID = "22222222-2222-2222-2222-222222222222",
  [string] $NoAccessPassword = "dev-noaccess-password",

  [string] $PlatformUserID = "00000000-0000-0000-0000-00000000aaaa",
  [string] $PlatformEmail = "dev.superadmin@example.test",
  [string] $PlatformPassword = "dev-superadmin-password",

  [switch] $SkipMigrations,
  [switch] $SkipRestart,
  [switch] $IncludeSessionTokens
)

$ErrorActionPreference = "Stop"

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $RepoRoot

$tenantGuid = [Guid]::Empty
if (-not [Guid]::TryParse($TenantID, [ref] $tenantGuid)) {
  throw "TenantID must be a valid UUID. Got: $TenantID"
}

$EnvPath = if ([IO.Path]::IsPathRooted($EnvFile)) {
  $EnvFile
}
else {
  Join-Path $RepoRoot $EnvFile
}

if (-not (Test-Path -LiteralPath $EnvPath)) {
  if ($EnvFile -ne ".env") {
    throw "Environment file not found: $EnvPath"
  }

  $EnvExamplePath = Join-Path $RepoRoot ".env.example"
  Copy-Item -LiteralPath $EnvExamplePath -Destination $EnvPath
  Write-Warning "Created .env from .env.example. Existing environment files are never overwritten."
}

$AllowedEmail = "inventory.allowed+$($TenantID.Substring(0, 8))@example.test"
$NoAccessEmail = "inventory.noaccess+$($TenantID.Substring(0, 8))@example.test"

Write-Host ""
Write-Host "===== FE local setup: database preflight ====="
& (Join-Path $PSScriptRoot "db-check.ps1") -EnvFile $EnvFile | Out-Host

if (-not $SkipMigrations) {
  Write-Host ""
  Write-Host "===== FE local setup: pending migrations ====="

  $MigrationFiles = @(
    Get-ChildItem -LiteralPath (Join-Path $RepoRoot "migrations") -Filter "*.up.sql" |
      Sort-Object Name
  )

  if ($MigrationFiles.Count -eq 0) {
    throw "No up migrations were found."
  }

  foreach ($MigrationFile in $MigrationFiles) {
    $MigrationName = $MigrationFile.Name -replace '\.up\.sql$', ''
    & (Join-Path $PSScriptRoot "apply-migration.ps1") `
      -Name $MigrationName `
      -Direction up `
      -EnvFile $EnvFile | Out-Host
  }
}
else {
  Write-Host "Skipping migrations because -SkipMigrations was supplied."
}

Write-Host ""
Write-Host "===== FE local setup: platform and tenant seed ====="
& (Join-Path $PSScriptRoot "seed-platform-superadmin.ps1") `
  -PlatformUserID $PlatformUserID `
  -Email $PlatformEmail `
  -Password $PlatformPassword `
  -EnvFile $EnvFile | Out-Host

& (Join-Path $PSScriptRoot "seed-dev-tenant.ps1") `
  -TenantID $TenantID `
  -Slug $TenantSlug `
  -LegalName $TenantLegalName `
  -DisplayName $TenantDisplayName `
  -EnvFile $EnvFile | Out-Host

& (Join-Path $PSScriptRoot "seed-dev-rbac.ps1") `
  -TenantID $TenantID `
  -AllowedUserID $AllowedUserID `
  -NoAccessUserID $NoAccessUserID `
  -AllowedPassword $AllowedPassword `
  -NoAccessPassword $NoAccessPassword `
  -EnvFile $EnvFile | Out-Host

if (-not $SkipRestart) {
  Write-Host ""
  Write-Host "===== FE local setup: start APIs ====="
  & (Join-Path $PSScriptRoot "restart-local-apis.ps1") `
    -BaseUrl $BaseUrl `
    -ControlPlaneUrl $ControlPlaneUrl `
    -EnvFile $EnvFile | Out-Host
}
else {
  Write-Host "Skipping API restart because -SkipRestart was supplied. Running APIs will still be validated."
}

Write-Host ""
Write-Host "===== FE local setup: validate handoff credentials ====="

$AllowedLogin = Invoke-RestMethod "$BaseUrl/api/v1/auth/login" `
  -Method Post `
  -ContentType "application/json" `
  -Body (@{
    tenant_id = $TenantID
    email = $AllowedEmail
    password = $AllowedPassword
  } | ConvertTo-Json)

if (-not $AllowedLogin.session_token -or $AllowedLogin.user_id -ne $AllowedUserID) {
  throw "Allowed tenant login validation failed."
}

$AllowedHeaders = @{
  "X-Tenant-ID" = $TenantID
  "X-ERP-Session" = $AllowedLogin.session_token
}

$AllowedMe = Invoke-RestMethod "$BaseUrl/api/v1/auth/me" -Headers $AllowedHeaders
if ($AllowedMe.user.user_id -ne $AllowedUserID) {
  throw "Allowed tenant /auth/me validation failed."
}

$TenantModules = Invoke-RestMethod "$BaseUrl/api/v1/modules" -Headers $AllowedHeaders
if ($TenantModules.tenant_id -ne $TenantID) {
  throw "Tenant module validation returned the wrong tenant."
}

$NoAccessLogin = Invoke-RestMethod "$BaseUrl/api/v1/auth/login" `
  -Method Post `
  -ContentType "application/json" `
  -Body (@{
    tenant_id = $TenantID
    email = $NoAccessEmail
    password = $NoAccessPassword
  } | ConvertTo-Json)

if (-not $NoAccessLogin.session_token -or $NoAccessLogin.user_id -ne $NoAccessUserID) {
  throw "No-access tenant login validation failed."
}

$NoAccessHeaders = @{
  "X-Tenant-ID" = $TenantID
  "X-ERP-Session" = $NoAccessLogin.session_token
}

$PlatformLogin = Invoke-RestMethod "$ControlPlaneUrl/control/v1/auth/login" `
  -Method Post `
  -ContentType "application/json" `
  -Body (@{
    email = $PlatformEmail
    password = $PlatformPassword
  } | ConvertTo-Json)

if (-not $PlatformLogin.session_token -or $PlatformLogin.platform_user_id -ne $PlatformUserID) {
  throw "Platform superadmin login validation failed."
}

$PlatformHeaders = @{
  "X-Platform-Session" = $PlatformLogin.session_token
}

$PlatformModules = Invoke-RestMethod "$ControlPlaneUrl/control/v1/modules" -Headers $PlatformHeaders

$Manifest = [ordered] @{
  schema_version = 1
  generated_at = (Get-Date).ToUniversalTime().ToString("o")
  environment = "local-development"
  warning = "Contains local development passwords. Do not commit or use outside local development."
  services = [ordered] @{
    erp_api = [ordered] @{
      base_url = $BaseUrl
      health_url = "$BaseUrl/healthz"
    }
    control_plane_api = [ordered] @{
      base_url = $ControlPlaneUrl
      health_url = "$ControlPlaneUrl/healthz"
    }
  }
  tenant = [ordered] @{
    id = $TenantID
    slug = $TenantSlug
    legal_name = $TenantLegalName
    display_name = $TenantDisplayName
    enabled_modules = @($TenantModules.modules | ForEach-Object { $_.id })
  }
  authentication = [ordered] @{
    tenant = [ordered] @{
      login_url = "$BaseUrl/api/v1/auth/login"
      me_url = "$BaseUrl/api/v1/auth/me"
      logout_url = "$BaseUrl/api/v1/auth/logout"
      request_headers = [ordered] @{
        tenant_id = "X-Tenant-ID"
        session = "X-ERP-Session"
      }
    }
    platform = [ordered] @{
      login_url = "$ControlPlaneUrl/control/v1/auth/login"
      logout_url = "$ControlPlaneUrl/control/v1/auth/logout"
      request_headers = [ordered] @{
        session = "X-Platform-Session"
      }
    }
  }
  credentials = [ordered] @{
    tenant_allowed = [ordered] @{
      tenant_id = $TenantID
      user_id = $AllowedUserID
      email = $AllowedEmail
      password = $AllowedPassword
      purpose = "All currently seeded Inventory, Purchase, and Sales permissions"
    }
    tenant_no_access = [ordered] @{
      tenant_id = $TenantID
      user_id = $NoAccessUserID
      email = $NoAccessEmail
      password = $NoAccessPassword
      purpose = "Authenticated user without business permissions; use for 403 tests"
    }
    platform_superadmin = [ordered] @{
      user_id = $PlatformUserID
      email = $PlatformEmail
      password = $PlatformPassword
      purpose = "Control-plane testing only"
    }
  }
  available_platform_modules = @($PlatformModules.modules | ForEach-Object { $_.id })
}

if ($IncludeSessionTokens) {
  $Manifest["sessions"] = [ordered] @{
    tenant_allowed = [ordered] @{
      token = $AllowedLogin.session_token
      expires_at = $AllowedLogin.expires_at
    }
    tenant_no_access = [ordered] @{
      token = $NoAccessLogin.session_token
      expires_at = $NoAccessLogin.expires_at
    }
    platform_superadmin = [ordered] @{
      token = $PlatformLogin.session_token
      expires_at = $PlatformLogin.expires_at
    }
  }
}
else {
  Invoke-RestMethod "$BaseUrl/api/v1/auth/logout" `
    -Method Post `
    -Headers $AllowedHeaders | Out-Null
  Invoke-RestMethod "$BaseUrl/api/v1/auth/logout" `
    -Method Post `
    -Headers $NoAccessHeaders | Out-Null
  Invoke-RestMethod "$ControlPlaneUrl/control/v1/auth/logout" `
    -Method Post `
    -Headers $PlatformHeaders | Out-Null
}

$ResolvedOutputPath = if ([IO.Path]::IsPathRooted($OutputPath)) {
  $OutputPath
}
else {
  Join-Path $RepoRoot $OutputPath
}

$OutputDirectory = Split-Path -Parent $ResolvedOutputPath
if ($OutputDirectory) {
  New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
}

$ManifestJson = $Manifest | ConvertTo-Json -Depth 10
[IO.File]::WriteAllText(
  $ResolvedOutputPath,
  $ManifestJson + [Environment]::NewLine,
  [Text.UTF8Encoding]::new($false)
)

Write-Host ""
Write-Host "FE local environment is ready."
Write-Host "ERP API:               $BaseUrl"
Write-Host "Control plane API:     $ControlPlaneUrl"
Write-Host "Tenant ID:              $TenantID"
Write-Host "Allowed user:           $AllowedEmail"
Write-Host "Allowed password:       $AllowedPassword"
Write-Host "No-access user:         $NoAccessEmail"
Write-Host "No-access password:     $NoAccessPassword"
Write-Host "Platform superadmin:    $PlatformEmail"
Write-Host "Platform password:      $PlatformPassword"
Write-Host "Integration manifest:   $ResolvedOutputPath"

[pscustomobject] $Manifest

param(
  [string] $Name = "000001_platform_foundation",

  [string] $EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$EnvPath = Join-Path $RepoRoot $EnvFile

. (Join-Path $PSScriptRoot "Import-DotEnv.ps1") -Path $EnvPath

if (-not $env:MIGRATION_DATABASE_URL) {
  throw "MIGRATION_DATABASE_URL is missing. Copy .env.example to .env and edit it."
}

$Sql = @"
CREATE TABLE IF NOT EXISTS public.schema_migrations (
    name TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO public.schema_migrations (name)
VALUES ('$Name')
ON CONFLICT (name) DO NOTHING;

SELECT name, applied_at
FROM public.schema_migrations
ORDER BY name;
"@

psql $env:MIGRATION_DATABASE_URL -v ON_ERROR_STOP=1 -c $Sql
if ($LASTEXITCODE -ne 0) {
  throw "Failed to mark migration as applied: $Name"
}

Write-Host ""
Write-Host "Migration marked as applied: $Name"

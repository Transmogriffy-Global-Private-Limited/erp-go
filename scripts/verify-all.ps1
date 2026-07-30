[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$localDirectory = Join-Path $repositoryRoot '.local'
$goCacheDirectory = Join-Path $localDirectory 'go-cache'
$binaryDirectory = Join-Path $localDirectory 'bin'
$binaryPath = Join-Path $binaryDirectory 'erp-api.exe'
$originalGoCache = [Environment]::GetEnvironmentVariable('GOCACHE', 'Process')
$startedProcesses = [System.Collections.Generic.List[System.Diagnostics.Process]]::new()
$httpHandler = [System.Net.Http.HttpClientHandler]::new()
$httpHandler.AllowAutoRedirect = $false
$httpClient = [System.Net.Http.HttpClient]::new($httpHandler)
$httpClient.Timeout = [TimeSpan]::FromSeconds(3)

function Assert-True {
    param(
        [Parameter(Mandatory)]
        [bool] $Condition,

        [Parameter(Mandatory)]
        [string] $Message
    )

    if (-not $Condition) {
        throw $Message
    }
}

function Invoke-GoCommand {
    param(
        [Parameter(Mandatory)]
        [string[]] $Arguments
    )

    & go @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "go $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
    }
}

function Get-FreeLoopbackPort {
    $listener = [System.Net.Sockets.TcpListener]::new(
        [System.Net.IPAddress]::Loopback,
        0
    )
    $listener.Start()
    try {
        return ([System.Net.IPEndPoint] $listener.LocalEndpoint).Port
    }
    finally {
        $listener.Stop()
    }
}

function Start-ERPApi {
    param(
        [Parameter(Mandatory)]
        [bool] $DocsEnabled
    )

    $port = Get-FreeLoopbackPort
    $startInfo = [System.Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $binaryPath
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    $startInfo.Environment['HTTP_HOST'] = '127.0.0.1'
    $startInfo.Environment['HTTP_PORT'] = [string] $port
    $startInfo.Environment['API_DOCS_ENABLED'] = $DocsEnabled.ToString().ToLowerInvariant()

    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    Assert-True -Condition $process.Start() -Message 'Failed to start erp-api.'
    $startedProcesses.Add($process)

    $deadline = [DateTime]::UtcNow.AddSeconds(8)
    while ([DateTime]::UtcNow -lt $deadline) {
        if ($process.HasExited) {
            $stdout = $process.StandardOutput.ReadToEnd()
            $stderr = $process.StandardError.ReadToEnd()
            throw "erp-api exited during startup.`nstdout: $stdout`nstderr: $stderr"
        }

        try {
            $response = $httpClient.GetAsync("http://127.0.0.1:$port/healthz").GetAwaiter().GetResult()
            try {
                if ($response.StatusCode -eq [System.Net.HttpStatusCode]::OK) {
                    return @{
                        Port = $port
                        Process = $process
                    }
                }
            }
            finally {
                $response.Dispose()
            }
        }
        catch [System.Net.Http.HttpRequestException] {
            # The listener may not be ready yet.
        }

        Start-Sleep -Milliseconds 100
    }

    throw "erp-api did not become healthy on loopback port $port within 8 seconds."
}

function Stop-ERPApi {
    param(
        [Parameter(Mandatory)]
        [System.Diagnostics.Process] $Process
    )

    if (-not $Process.HasExited) {
        $Process.Kill($true)
        $Process.WaitForExit(5000) | Out-Null
    }
}

function Invoke-HTTPGet {
    param(
        [Parameter(Mandatory)]
        [int] $Port,

        [Parameter(Mandatory)]
        [string] $Path
    )

    return $httpClient.GetAsync("http://127.0.0.1:$Port$Path").GetAwaiter().GetResult()
}

function Assert-JSONError {
    param(
        [Parameter(Mandatory)]
        [System.Net.Http.HttpResponseMessage] $Response,

        [Parameter(Mandatory)]
        [string] $ExpectedCode
    )

    $body = $Response.Content.ReadAsStringAsync().GetAwaiter().GetResult() | ConvertFrom-Json
    Assert-True -Condition ($body.error.code -eq $ExpectedCode) -Message (
        "Expected error code '$ExpectedCode', got '$($body.error.code)'."
    )
    Assert-True -Condition (-not [string]::IsNullOrWhiteSpace($body.error.message)) -Message (
        'Canonical error message must not be empty.'
    )
}

function Assert-InvalidConfigurationFails {
    $startInfo = [System.Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $binaryPath
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    $startInfo.Environment['HTTP_HOST'] = '0.0.0.0'
    $startInfo.Environment['HTTP_PORT'] = [string] (Get-FreeLoopbackPort)
    $startInfo.Environment['API_DOCS_ENABLED'] = 'false'

    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    Assert-True -Condition $process.Start() -Message 'Failed to start invalid-configuration check.'
    try {
        Assert-True -Condition $process.WaitForExit(5000) -Message (
            'erp-api did not reject invalid configuration within 5 seconds.'
        )
        $output = $process.StandardOutput.ReadToEnd() + $process.StandardError.ReadToEnd()
        Assert-True -Condition ($process.ExitCode -ne 0) -Message (
            'erp-api unexpectedly accepted a wildcard listen host.'
        )
        Assert-True -Condition ($output -match 'loopback host') -Message (
            'Invalid listen-host error was not actionable.'
        )
    }
    finally {
        if (-not $process.HasExited) {
            $process.Kill($true)
        }
        $process.Dispose()
    }
}

function Assert-DocsDisabled {
    $server = Start-ERPApi -DocsEnabled $false
    try {
        $health = Invoke-HTTPGet -Port $server.Port -Path '/healthz'
        try {
            Assert-True -Condition ($health.StatusCode -eq [System.Net.HttpStatusCode]::OK) -Message (
                'GET /healthz did not return 200 with documentation disabled.'
            )
        }
        finally {
            $health.Dispose()
        }

        foreach ($path in @('/docs', '/docs/', '/docs/swagger-ui.css', '/openapi.yaml')) {
            $response = Invoke-HTTPGet -Port $server.Port -Path $path
            try {
                Assert-True -Condition ($response.StatusCode -eq [System.Net.HttpStatusCode]::NotFound) -Message (
                    "GET $path must return 404 when API documentation is disabled."
                )
                Assert-JSONError -Response $response -ExpectedCode 'not_found'
            }
            finally {
                $response.Dispose()
            }
        }
    }
    finally {
        Stop-ERPApi -Process $server.Process
    }
}

function Assert-DocsEnabled {
    $server = Start-ERPApi -DocsEnabled $true
    try {
        $schema = Invoke-HTTPGet -Port $server.Port -Path '/openapi.yaml'
        try {
            Assert-True -Condition ($schema.StatusCode -eq [System.Net.HttpStatusCode]::OK) -Message (
                'GET /openapi.yaml did not return 200 with documentation enabled.'
            )
            $schemaBody = $schema.Content.ReadAsStringAsync().GetAwaiter().GetResult()
            Assert-True -Condition ($schemaBody -match 'openapi: 3.0.3') -Message (
                'The raw schema route did not serve the authoritative OpenAPI document.'
            )
        }
        finally {
            $schema.Dispose()
        }

        $docs = Invoke-HTTPGet -Port $server.Port -Path '/docs/'
        try {
            Assert-True -Condition ($docs.StatusCode -eq [System.Net.HttpStatusCode]::OK) -Message (
                'GET /docs/ did not return 200 with documentation enabled.'
            )
            $docsBody = $docs.Content.ReadAsStringAsync().GetAwaiter().GetResult()
            Assert-True -Condition ($docsBody -match '/openapi.yaml') -Message (
                'Swagger UI does not reference the authoritative schema route.'
            )
            Assert-True -Condition ($docsBody -notmatch '(?i)cdnjs') -Message (
                'Swagger UI unexpectedly uses CDN assets.'
            )
        }
        finally {
            $docs.Dispose()
        }

        $asset = Invoke-HTTPGet -Port $server.Port -Path '/docs/swagger-ui.css'
        try {
            Assert-True -Condition ($asset.StatusCode -eq [System.Net.HttpStatusCode]::OK) -Message (
                'The embedded Swagger UI stylesheet was not served.'
            )
        }
        finally {
            $asset.Dispose()
        }
    }
    finally {
        Stop-ERPApi -Process $server.Process
    }
}

Push-Location $repositoryRoot
try {
    New-Item -ItemType Directory -Force -Path $goCacheDirectory, $binaryDirectory | Out-Null
    [Environment]::SetEnvironmentVariable('GOCACHE', $goCacheDirectory, 'Process')

    Write-Host 'Checking Go formatting...'
    $goFiles = Get-ChildItem -Path $repositoryRoot -Recurse -File -Filter '*.go' |
        Where-Object { $_.FullName -notlike "$localDirectory*" }
    $unformatted = @(& gofmt -l @($goFiles.FullName))
    if ($LASTEXITCODE -ne 0) {
        throw "gofmt inspection failed with exit code $LASTEXITCODE"
    }
    Assert-True -Condition ($unformatted.Count -eq 0) -Message (
        "These Go files require gofmt:`n$($unformatted -join [Environment]::NewLine)"
    )

    Write-Host 'Running all Go tests...'
    Invoke-GoCommand -Arguments @('test', './...')

    Write-Host 'Running Go static analysis...'
    Invoke-GoCommand -Arguments @('vet', './...')

    Write-Host 'Building erp-api...'
    Invoke-GoCommand -Arguments @('build', '-o', $binaryPath, './cmd/erp-api')

    Write-Host 'Checking invalid configuration rejection...'
    Assert-InvalidConfigurationFails

    Write-Host 'Smoke-testing documentation disabled...'
    Assert-DocsDisabled

    Write-Host 'Smoke-testing documentation enabled...'
    Assert-DocsEnabled

    Write-Host 'All ERP verification checks passed.' -ForegroundColor Green
}
finally {
    foreach ($process in $startedProcesses) {
        if (-not $process.HasExited) {
            $process.Kill($true)
            $process.WaitForExit(5000) | Out-Null
        }
        $process.Dispose()
    }

    $httpClient.Dispose()
    $httpHandler.Dispose()
    [Environment]::SetEnvironmentVariable('GOCACHE', $originalGoCache, 'Process')
    Pop-Location
}

# Local development

This guide covers the implemented Step 01A API foundation. It uses native Go
and PowerShell without administrator access, containers, a database, or a
public listener.

## Requirements

- Windows 11
- PowerShell 7 or later
- Go 1.25 or later on the user PATH

## Configuration

| Variable | Required | Default | Accepted values | Purpose |
|---|---:|---|---|---|
| `HTTP_HOST` | No | `127.0.0.1` | `127.0.0.1`, `localhost`, `::1`, or another loopback IP | HTTP listen host. Wildcard and non-loopback hosts are rejected. |
| `HTTP_PORT` | No | `8080` | Integer from `1` through `65535` | Unprivileged HTTP listen port. |
| `API_DOCS_ENABLED` | No | `false` | `true` or `false`, case-insensitive | Registers `/docs`, `/docs/`, embedded UI assets, and `/openapi.yaml`. Restart is required after changing it. |

These variables are not secrets. Future secret configuration must use the
repository's approved secret mechanism and must not be committed.

## Run the API

From the repository root:

```powershell
$env:HTTP_HOST = '127.0.0.1'
$env:HTTP_PORT = '8080'
$env:API_DOCS_ENABLED = 'true'
go run ./cmd/erp-api
```

Use <http://127.0.0.1:8080/healthz> for liveness. With documentation enabled,
use <http://127.0.0.1:8080/docs/> for Swagger UI and
<http://127.0.0.1:8080/openapi.yaml> for the raw authoritative contract.

Press `Ctrl+C` to request graceful shutdown. The process stops accepting new
requests, allows active requests up to ten seconds to finish, and then exits.

## Verify everything

The repository's canonical local verification command is:

```powershell
.\scripts\verify-all.ps1
```

It uses an ignored repository-local Go cache, formats-checks all Go sources,
runs all Go tests and Go static analysis, builds the binary, proves wildcard
configuration is rejected, and smoke-tests the real binary with documentation
both disabled and enabled on temporary loopback ports. It does not modify a
database or contact a remote service.

## Troubleshooting

### Port already in use

Choose another unprivileged loopback port:

```powershell
$env:HTTP_PORT = '8081'
go run ./cmd/erp-api
```

### Go cache access denied

Use the ignored repository-local cache used by the verifier:

```powershell
$env:GOCACHE = Join-Path (Get-Location) '.local\go-cache'
go test ./...
```

### Documentation returns not found

Set `API_DOCS_ENABLED=true` before starting the process and restart it. When the
toggle is absent or false, the documentation routes are intentionally not
registered and return the canonical JSON `not_found` response.

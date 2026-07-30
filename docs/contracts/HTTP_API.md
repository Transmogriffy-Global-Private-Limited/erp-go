# HTTP API contract

The authoritative machine-readable contract is
[`../../api/openapi/v1/openapi.yaml`](../../api/openapi/v1/openapi.yaml). The
application embeds and serves those exact bytes; Swagger UI renders the same
document. This file explains operational semantics that are easier to read in
human form.

## Current boundary

Step 01A exposes only process liveness and optional API documentation. It has no
database, readiness claim, authentication, authorization, tenant scope, domain
command, event, background work, or durable state transition.

All implemented methods are unauthenticated because they disclose only the
minimal liveness result and the repository's public contract. Later business
routes must define authentication, authorization, and trusted scope explicitly.

## Endpoints

| Method | Path | Availability | Result |
|---|---|---|---|
| `GET` | `/healthz` | Always | `200` with `{"status":"ok"}`. This is process liveness, not dependency readiness. |
| `GET` | `/openapi.yaml` | Only when `API_DOCS_ENABLED=true` | `200` with the embedded OpenAPI YAML and `Cache-Control: no-store`. |
| `GET` | `/docs` | Only when `API_DOCS_ENABLED=true` | `308` redirect to `/docs/`. |
| `GET` | `/docs/` | Only when `API_DOCS_ENABLED=true` | Embedded Swagger UI. Assets are served below the same path without a CDN. |

When documentation is disabled, `/docs`, `/docs/`, their asset paths, and
`/openapi.yaml` are not registered and return the canonical `404` response.
Unsupported methods on registered exact routes return `405` with `Allow: GET`.

## Canonical JSON errors

Unknown routes use:

```json
{
  "error": {
    "code": "not_found",
    "message": "The requested resource was not found."
  }
}
```

Unsupported methods use the same envelope with code `method_not_allowed`. Error
codes are stable programmatic values. Messages are human-readable and must not
contain sensitive details.

## Contract maintenance and verification

The source of truth is specification-first and committed at
`api/openapi/v1/openapi.yaml`. Do not manually maintain another schema.

Contract drift is checked in Go: the document is parsed, structurally validated,
and compared with the complete current public route set. The full verifier also
builds the real binary and checks raw-schema and Swagger behavior with the
documentation toggle in both states.

Run:

```powershell
.\scripts\verify-all.ps1
```

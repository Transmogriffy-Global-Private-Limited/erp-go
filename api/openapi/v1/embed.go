// Package openapiv1 owns the authoritative HTTP API contract for version 1.
package openapiv1

import _ "embed"

// Document is the exact OpenAPI document served by the application.
//
//go:embed openapi.yaml
var Document []byte

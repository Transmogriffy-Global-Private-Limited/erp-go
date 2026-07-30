package httpapi

import (
	"context"
	"slices"
	"testing"

	openapiv1 "github.com/Transmogriffy-Global-Private-Limited/erp-go/api/openapi/v1"
	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPIDocument(t *testing.T) {
	t.Parallel()

	document, err := openapi3.NewLoader().LoadFromData(openapiv1.Document)
	if err != nil {
		t.Fatalf("load OpenAPI document: %v", err)
	}
	if err := document.Validate(context.Background()); err != nil {
		t.Fatalf("validate OpenAPI document: %v", err)
	}

	gotPaths := make([]string, 0, document.Paths.Len())
	for path := range document.Paths.Map() {
		gotPaths = append(gotPaths, path)
	}
	slices.Sort(gotPaths)

	wantPaths := []string{"/docs", "/docs/", "/healthz", "/openapi.yaml"}
	if !slices.Equal(gotPaths, wantPaths) {
		t.Fatalf("documented paths = %v, want %v", gotPaths, wantPaths)
	}
}

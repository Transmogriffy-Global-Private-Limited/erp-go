package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	openapiv1 "github.com/Transmogriffy-Global-Private-Limited/erp-go/api/openapi/v1"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/config"
)

func TestHealth(t *testing.T) {
	t.Parallel()

	response := serveRequest(t, false, http.MethodGet, "/healthz")
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}

	var body struct {
		Status string `json:"status"`
	}
	decodeJSON(t, response.Body, &body)
	if body.Status != "ok" {
		t.Fatalf("status body = %q, want ok", body.Status)
	}
}

func TestCanonicalErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantCode   string
		wantAllow  string
	}{
		{
			name:       "unknown route",
			method:     http.MethodGet,
			path:       "/missing",
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
		},
		{
			name:       "unsupported method",
			method:     http.MethodPost,
			path:       "/healthz",
			wantStatus: http.StatusMethodNotAllowed,
			wantCode:   "method_not_allowed",
			wantAllow:  http.MethodGet,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			response := serveRequest(t, false, test.method, test.path)
			defer response.Body.Close()

			if response.StatusCode != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.wantStatus)
			}
			if got := response.Header.Get("Allow"); got != test.wantAllow {
				t.Fatalf("Allow = %q, want %q", got, test.wantAllow)
			}

			var body errorEnvelope
			decodeJSON(t, response.Body, &body)
			if body.Error.Code != test.wantCode {
				t.Fatalf("error code = %q, want %q", body.Error.Code, test.wantCode)
			}
			if body.Error.Message == "" {
				t.Fatal("error message is empty")
			}
		})
	}
}

func TestDocsDisabled(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"/openapi.yaml",
		"/docs",
		"/docs/",
		"/docs/swagger-ui.css",
	} {
		response := serveRequest(t, false, http.MethodGet, path)
		response.Body.Close()

		if response.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want %d", path, response.StatusCode, http.StatusNotFound)
		}
	}
}

func TestDocsEnabled(t *testing.T) {
	t.Parallel()

	t.Run("raw contract", func(t *testing.T) {
		response := serveRequest(t, true, http.MethodGet, "/openapi.yaml")
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
		}
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != string(openapiv1.Document) {
			t.Fatal("served OpenAPI document differs from embedded source")
		}
	})

	t.Run("canonical redirect", func(t *testing.T) {
		response := serveRequest(t, true, http.MethodGet, "/docs")
		response.Body.Close()

		if response.StatusCode != http.StatusPermanentRedirect {
			t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusPermanentRedirect)
		}
		if got := response.Header.Get("Location"); got != "/docs/" {
			t.Fatalf("Location = %q, want /docs/", got)
		}
	})

	t.Run("interactive UI", func(t *testing.T) {
		response := serveRequest(t, true, http.MethodGet, "/docs/")
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
		}
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		page := string(body)
		if !strings.Contains(page, "/openapi.yaml") {
			t.Fatal("Swagger UI does not reference the authoritative OpenAPI route")
		}
		if strings.Contains(strings.ToLower(page), "cdnjs") {
			t.Fatal("Swagger UI unexpectedly references CDN assets")
		}
	})

	t.Run("embedded asset", func(t *testing.T) {
		response := serveRequest(t, true, http.MethodGet, "/docs/swagger-ui.css")
		response.Body.Close()

		if response.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
		}
	})
}

func serveRequest(t *testing.T, docsEnabled bool, method, path string) *http.Response {
	t.Helper()

	request := httptest.NewRequest(method, path, nil)
	recorder := httptest.NewRecorder()
	NewHandler(config.Config{APIDocsEnabled: docsEnabled}).ServeHTTP(recorder, request)
	return recorder.Result()
}

func decodeJSON(t *testing.T, reader io.Reader, target any) {
	t.Helper()

	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
}

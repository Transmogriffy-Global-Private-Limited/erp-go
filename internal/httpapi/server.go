package httpapi

import (
	"net/http"

	openapiv1 "github.com/Transmogriffy-Global-Private-Limited/erp-go/api/openapi/v1"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/config"
	"github.com/swaggest/swgui/v5emb"
)

// NewHandler constructs the complete Step 01A HTTP surface.
func NewHandler(cfg config.Config) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/healthz", getOnly(http.HandlerFunc(handleHealth)))

	if cfg.APIDocsEnabled {
		mux.Handle("/openapi.yaml", getOnly(http.HandlerFunc(handleOpenAPI)))
		mux.Handle("/docs", getOnly(http.HandlerFunc(redirectDocs)))
		mux.Handle("/docs/", getOnly(v5emb.New("ERP API", "/openapi.yaml", "/docs/")))
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(
			w,
			http.StatusNotFound,
			"not_found",
			"The requested resource was not found.",
		)
	})

	return mux
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, struct {
		Status string `json:"status"`
	}{Status: "ok"})
}

func handleOpenAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openapiv1.Document)
}

func redirectDocs(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/docs/", http.StatusPermanentRedirect)
}

func getOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeError(
				w,
				http.StatusMethodNotAllowed,
				"method_not_allowed",
				"The request method is not allowed for this resource.",
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}

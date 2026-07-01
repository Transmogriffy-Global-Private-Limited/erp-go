package controlapi

import (
	"net/http"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/httpx"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/licensing"
	platformmodules "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/modules"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/tenancy"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	db           *pgxpool.Pool
	platformAuth auth.Store
	modules      platformmodules.Store
	tenants      tenancy.Store
	licensing    licensing.Store
}

func NewHandler(pool *pgxpool.Pool) http.Handler {
	app := &App{
		db:           pool,
		platformAuth: auth.NewStore(pool),
		modules:      platformmodules.NewStore(pool),
		tenants:      tenancy.NewStore(pool),
		licensing:    licensing.NewStore(pool),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", app.healthHandler)
	mux.HandleFunc("/healthz/db", app.dbHealthHandler)

	mux.Handle("/control/v1/tenants", app.platformAuthMiddleware(http.HandlerFunc(app.tenantsHandler)))
	mux.Handle("/control/v1/tenants/", app.platformAuthMiddleware(http.HandlerFunc(app.tenantSubresourceHandler)))
	mux.Handle("/control/v1/modules", app.platformAuthMiddleware(http.HandlerFunc(app.modulesHandler)))
	mux.Handle("/control/v1/plans", app.platformAuthMiddleware(http.HandlerFunc(app.plansHandler)))
	mux.Handle("/control/v1/plans/", app.platformAuthMiddleware(http.HandlerFunc(app.planSubresourceHandler)))

	return mux
}

func (a *App) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"service": "control-plane-api",
		"status":  "ok",
	})
}

func (a *App) dbHealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}

	if err := db.Ping(r.Context(), a.db); err != nil {
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]any{
			"service": "control-plane-api",
			"status":  "db_unavailable",
			"error":   err.Error(),
		})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"service": "control-plane-api",
		"status":  "db_ok",
	})
}

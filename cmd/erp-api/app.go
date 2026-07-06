package main

import (
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/inventory"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/purchase"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/modules/sales"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/auth"
	platformmodules "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/modules"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/rbac"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	db             *pgxpool.Pool
	modules        platformmodules.Store
	rbac           rbac.Store
	inventory      inventory.Store
	purchase       purchase.Store
	sales          sales.Store
	tenantSessions auth.TenantSessionStore
}

func newApp(pool *pgxpool.Pool) *app {
	return &app{
		db:             pool,
		modules:        platformmodules.NewStore(pool),
		rbac:           rbac.NewStore(pool),
		inventory:      inventory.NewStore(pool),
		purchase:       purchase.NewStore(pool),
		sales:          sales.NewStore(pool),
		tenantSessions: auth.NewTenantSessionStore(pool),
	}
}

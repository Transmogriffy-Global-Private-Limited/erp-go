package licensing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/modules"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Plan struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type CreatePlanInput struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Subscription struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id"`
	PlanID    string     `json:"plan_id"`
	Status    string     `json:"status"`
	StartsAt  time.Time  `json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type AssignSubscriptionInput struct {
	PlanID string `json:"plan_id"`
	Status string `json:"status"`
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) Store {
	return Store{
		db: db,
	}
}

func (s Store) ListPlans(ctx context.Context) ([]Plan, error) {
	rows, err := s.db.Query(ctx, `
SELECT id, name, status, created_at
FROM control.plans
ORDER BY id
`)
	if err != nil {
		return nil, fmt.Errorf("query plans: %w", err)
	}
	defer rows.Close()

	plans := make([]Plan, 0)

	for rows.Next() {
		var plan Plan
		if err := rows.Scan(&plan.ID, &plan.Name, &plan.Status, &plan.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}

		plans = append(plans, plan)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plans: %w", err)
	}

	return plans, nil
}

func (s Store) CreatePlan(ctx context.Context, input CreatePlanInput) (Plan, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.Name = strings.TrimSpace(input.Name)
	input.Status = strings.TrimSpace(input.Status)

	if input.Status == "" {
		input.Status = "draft"
	}

	var plan Plan

	err := s.db.QueryRow(ctx, `
INSERT INTO control.plans (
id,
name,
status
)
VALUES (
$1,
$2,
$3
)
RETURNING id, name, status, created_at
`, input.ID, input.Name, input.Status).Scan(
		&plan.ID,
		&plan.Name,
		&plan.Status,
		&plan.CreatedAt,
	)
	if err != nil {
		return Plan{}, fmt.Errorf("insert plan: %w", err)
	}

	return plan, nil
}

func (s Store) ListPlanModules(ctx context.Context, planID string) ([]modules.Module, error) {
	rows, err := s.db.Query(ctx, `
SELECT m.id, m.name, m.status
FROM control.plan_modules pm
INNER JOIN control.modules m ON m.id = pm.module_id
WHERE pm.plan_id = $1
ORDER BY m.id
`, planID)
	if err != nil {
		return nil, fmt.Errorf("query plan modules: %w", err)
	}
	defer rows.Close()

	result := make([]modules.Module, 0)

	for rows.Next() {
		var module modules.Module
		if err := rows.Scan(&module.ID, &module.Name, &module.Status); err != nil {
			return nil, fmt.Errorf("scan plan module: %w", err)
		}

		result = append(result, module)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plan modules: %w", err)
	}

	return result, nil
}

func (s Store) EnablePlanModule(ctx context.Context, planID string, moduleID string) (modules.Module, error) {
	var module modules.Module

	err := s.db.QueryRow(ctx, `
WITH enabled AS (
INSERT INTO control.plan_modules (
plan_id,
module_id
)
VALUES (
$1,
$2
)
ON CONFLICT (plan_id, module_id) DO NOTHING
RETURNING module_id
),
selected_module AS (
SELECT module_id FROM enabled
UNION
SELECT module_id
FROM control.plan_modules
WHERE plan_id = $1
  AND module_id = $2
)
SELECT m.id, m.name, m.status
FROM selected_module sm
INNER JOIN control.modules m ON m.id = sm.module_id
LIMIT 1
`, planID, moduleID).Scan(&module.ID, &module.Name, &module.Status)
	if err != nil {
		return modules.Module{}, fmt.Errorf("enable plan module: %w", err)
	}

	return module, nil
}

func (s Store) AssignTenantSubscription(ctx context.Context, tenantID string, input AssignSubscriptionInput) (Subscription, []modules.Module, error) {
	input.PlanID = strings.TrimSpace(input.PlanID)
	input.Status = strings.TrimSpace(input.Status)

	if input.Status == "" {
		input.Status = "active"
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Subscription{}, nil, fmt.Errorf("begin subscription transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, `
UPDATE control.tenant_subscriptions
SET status = 'cancelled',
    ends_at = now()
WHERE tenant_id = $1
  AND status IN ('trialing', 'active', 'past_due', 'suspended')
`, tenantID)
	if err != nil {
		return Subscription{}, nil, fmt.Errorf("cancel previous subscriptions: %w", err)
	}

	var subscription Subscription

	err = tx.QueryRow(ctx, `
INSERT INTO control.tenant_subscriptions (
tenant_id,
plan_id,
status,
starts_at
)
VALUES (
$1,
$2,
$3,
now()
)
RETURNING id::text, tenant_id::text, plan_id, status, starts_at, ends_at, created_at
`, tenantID, input.PlanID, input.Status).Scan(
		&subscription.ID,
		&subscription.TenantID,
		&subscription.PlanID,
		&subscription.Status,
		&subscription.StartsAt,
		&subscription.EndsAt,
		&subscription.CreatedAt,
	)
	if err != nil {
		return Subscription{}, nil, fmt.Errorf("insert tenant subscription: %w", err)
	}

	_, err = tx.Exec(ctx, `
INSERT INTO control.tenant_enabled_modules (
tenant_id,
module_id,
enabled_at,
disabled_at
)
SELECT
$1,
pm.module_id,
now(),
NULL
FROM control.plan_modules pm
WHERE pm.plan_id = $2
ON CONFLICT (tenant_id, module_id) DO UPDATE
SET enabled_at = now(),
    disabled_at = NULL
`, tenantID, input.PlanID)
	if err != nil {
		return Subscription{}, nil, fmt.Errorf("enable tenant modules from plan: %w", err)
	}

	enabledModules, err := listPlanModulesTx(ctx, tx, input.PlanID)
	if err != nil {
		return Subscription{}, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Subscription{}, nil, fmt.Errorf("commit subscription transaction: %w", err)
	}

	return subscription, enabledModules, nil
}

func listPlanModulesTx(ctx context.Context, tx pgx.Tx, planID string) ([]modules.Module, error) {
	rows, err := tx.Query(ctx, `
SELECT m.id, m.name, m.status
FROM control.plan_modules pm
INNER JOIN control.modules m ON m.id = pm.module_id
WHERE pm.plan_id = $1
ORDER BY m.id
`, planID)
	if err != nil {
		return nil, fmt.Errorf("query plan modules: %w", err)
	}
	defer rows.Close()

	result := make([]modules.Module, 0)

	for rows.Next() {
		var module modules.Module
		if err := rows.Scan(&module.ID, &module.Name, &module.Status); err != nil {
			return nil, fmt.Errorf("scan plan module: %w", err)
		}

		result = append(result, module)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plan modules: %w", err)
	}

	return result, nil
}

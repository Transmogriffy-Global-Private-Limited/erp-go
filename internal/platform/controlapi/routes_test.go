package controlapi

import "testing"

func TestParseTenantModulePath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantTenant string
		wantModule string
		wantAction string
		wantOK     bool
	}{
		{
			name:       "list tenant modules",
			path:       "/control/v1/tenants/tenant-1/modules",
			wantTenant: "tenant-1",
			wantModule: "",
			wantAction: "list",
			wantOK:     true,
		},
		{
			name:       "enable tenant module",
			path:       "/control/v1/tenants/tenant-1/modules/inventory/enable",
			wantTenant: "tenant-1",
			wantModule: "inventory",
			wantAction: "enable",
			wantOK:     true,
		},
		{
			name:       "disable tenant module",
			path:       "/control/v1/tenants/tenant-1/modules/inventory/disable",
			wantTenant: "tenant-1",
			wantModule: "inventory",
			wantAction: "disable",
			wantOK:     true,
		},
		{
			name:   "reject unknown tenant action",
			path:   "/control/v1/tenants/tenant-1/modules/inventory/delete",
			wantOK: false,
		},
		{
			name:   "reject empty segment",
			path:   "/control/v1/tenants/tenant-1/modules//enable",
			wantOK: false,
		},
		{
			name:   "reject wrong prefix",
			path:   "/control/v1/plans/basic/modules",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantID, moduleID, action, ok := parseTenantModulePath(tt.path)

			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}

			if tenantID != tt.wantTenant {
				t.Fatalf("tenantID = %q, want %q", tenantID, tt.wantTenant)
			}

			if moduleID != tt.wantModule {
				t.Fatalf("moduleID = %q, want %q", moduleID, tt.wantModule)
			}

			if action != tt.wantAction {
				t.Fatalf("action = %q, want %q", action, tt.wantAction)
			}
		})
	}
}

func TestParseTenantSubscriptionPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantTenant string
		wantOK     bool
	}{
		{
			name:       "tenant subscription",
			path:       "/control/v1/tenants/tenant-1/subscription",
			wantTenant: "tenant-1",
			wantOK:     true,
		},
		{
			name:   "reject extra segment",
			path:   "/control/v1/tenants/tenant-1/subscription/active",
			wantOK: false,
		},
		{
			name:   "reject wrong suffix",
			path:   "/control/v1/tenants/tenant-1/modules",
			wantOK: false,
		},
		{
			name:   "reject wrong prefix",
			path:   "/control/v1/plans/basic/modules",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantID, ok := parseTenantSubscriptionPath(tt.path)

			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}

			if tenantID != tt.wantTenant {
				t.Fatalf("tenantID = %q, want %q", tenantID, tt.wantTenant)
			}
		})
	}
}

func TestParsePlanModulePath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantPlan   string
		wantModule string
		wantAction string
		wantOK     bool
	}{
		{
			name:       "list plan modules",
			path:       "/control/v1/plans/basic/modules",
			wantPlan:   "basic",
			wantModule: "",
			wantAction: "list",
			wantOK:     true,
		},
		{
			name:       "enable plan module",
			path:       "/control/v1/plans/basic/modules/inventory/enable",
			wantPlan:   "basic",
			wantModule: "inventory",
			wantAction: "enable",
			wantOK:     true,
		},
		{
			name:   "reject disable plan module for now",
			path:   "/control/v1/plans/basic/modules/inventory/disable",
			wantOK: false,
		},
		{
			name:   "reject empty segment",
			path:   "/control/v1/plans/basic/modules//enable",
			wantOK: false,
		},
		{
			name:   "reject wrong prefix",
			path:   "/control/v1/tenants/tenant-1/modules",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planID, moduleID, action, ok := parsePlanModulePath(tt.path)

			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}

			if planID != tt.wantPlan {
				t.Fatalf("planID = %q, want %q", planID, tt.wantPlan)
			}

			if moduleID != tt.wantModule {
				t.Fatalf("moduleID = %q, want %q", moduleID, tt.wantModule)
			}

			if action != tt.wantAction {
				t.Fatalf("action = %q, want %q", action, tt.wantAction)
			}
		})
	}
}

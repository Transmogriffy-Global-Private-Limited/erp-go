package controlapi

import "strings"

func splitControlPath(path string, prefix string) ([]string, bool) {
	if !strings.HasPrefix(path, prefix) {
		return nil, false
	}

	rest := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if rest == "" {
		return []string{}, true
	}

	parts := strings.Split(rest, "/")
	for _, part := range parts {
		if part == "" {
			return nil, false
		}
	}

	return parts, true
}

func parseTenantModulePath(path string) (tenantID string, moduleID string, action string, ok bool) {
	parts, ok := splitControlPath(path, "/control/v1/tenants/")
	if !ok {
		return "", "", "", false
	}

	if len(parts) == 2 && parts[1] == "modules" {
		return parts[0], "", "list", true
	}

	if len(parts) == 4 && parts[1] == "modules" {
		if parts[3] == "enable" || parts[3] == "disable" {
			return parts[0], parts[2], parts[3], true
		}
	}

	return "", "", "", false
}

func parseTenantSubscriptionPath(path string) (tenantID string, ok bool) {
	parts, ok := splitControlPath(path, "/control/v1/tenants/")
	if !ok {
		return "", false
	}

	if len(parts) == 2 && parts[1] == "subscription" {
		return parts[0], true
	}

	return "", false
}

func parsePlanModulePath(path string) (planID string, moduleID string, action string, ok bool) {
	parts, ok := splitControlPath(path, "/control/v1/plans/")
	if !ok {
		return "", "", "", false
	}

	if len(parts) == 2 && parts[1] == "modules" {
		return parts[0], "", "list", true
	}

	if len(parts) == 4 && parts[1] == "modules" && parts[3] == "enable" {
		return parts[0], parts[2], "enable", true
	}

	return "", "", "", false
}

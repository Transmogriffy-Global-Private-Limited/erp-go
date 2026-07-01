package auth

// Package auth will own authentication primitives for platform and tenant users.
//
// It must support separate auth concerns for:
// - platform users, used by the control plane and superadmin panel
// - tenant users, used by ERP tenants
//
// Do not put authorization policy decisions here.
// Authorization belongs in the rbac package and module permission checks.

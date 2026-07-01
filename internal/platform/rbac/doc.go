package rbac

// Package rbac will own roles, permissions, permission grants, and scope checks.
//
// It must support:
// - platform-side permissions for superadmin/control-plane actions
// - tenant-side permissions for ERP actions
// - module-defined permissions
// - branch/warehouse/department scope checks later
//
// Roles are bundles of permissions.
// Permissions are the actual authorization units.

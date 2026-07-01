package licensing

// Package licensing will own tenant plans, subscriptions, entitlements, limits,
// and enabled-module decisions.
//
// ERP business modules should not call a remote license server on every request.
// Entitlements should be propagated/cached into the ERP runtime.

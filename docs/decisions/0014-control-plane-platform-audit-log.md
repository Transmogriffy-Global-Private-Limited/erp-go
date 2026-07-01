# ADR 0014: Control Plane Platform Audit Log

## Status

Accepted

## Context

Control-plane operations are platform-powerful:

- creating tenants
- creating plans
- enabling modules
- assigning subscriptions
- changing tenant module entitlements

These actions need auditability separate from tenant ERP business audit.

## Decision

Create control.platform_audit_log for platform/superadmin actions.

This is separate from audit.audit_log.

audit.audit_log remains tenant ERP business audit and is RLS-protected.

control.platform_audit_log records:

- platform_actor_id
- action
- target_type
- target_id
- optional tenant_id
- metadata
- created_at

## Consequences

Superadmin/control-plane actions are accountable.

Future real platform authentication should continue writing platform_actor_id into this table.

Tenant ERP business audit and platform audit remain separate.

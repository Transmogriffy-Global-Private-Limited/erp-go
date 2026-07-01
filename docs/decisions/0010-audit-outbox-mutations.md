# ADR 0010: Audit and Outbox on Business Mutations

## Status

Accepted

## Context

ERP business mutations must be accountable and event-capable.

Creating a business record without audit makes support/security weak.

Creating a business record without an outbox event makes module intercommunication unreliable.

## Decision

Business mutations must write business data, audit log, and outbox event in the same database transaction.

The first implemented mutation is:

POST /api/v1/inventory/items

It now writes:

- inventory.items
- audit.audit_log with action inventory.item.create
- core.outbox_events with event inventory.item.created.v1

## Consequences

Future module mutations should follow the same pattern.

Events are durable in core.outbox_events before any external event bus is introduced.

A future worker will publish outbox events and mark them published.

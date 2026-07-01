# ADR 0011: DB-backed Outbox Worker First

## Status

Accepted

## Context

Business mutations write durable outbox events to core.outbox_events.

A future event bus may be added, but the project does not use Docker or containers for local development, and no external bus is required yet.

## Decision

Implement the first worker as a DB-backed outbox processor.

The worker:

- claims pending/failed events
- marks them publishing
- logs the event payload
- marks them published

No external broker is introduced in this step.

## Consequences

The event processing spine is now executable without NATS or container infrastructure.

Future work can replace the log-publish step with NATS, webhooks, notifications, or other delivery mechanisms.

The durable database outbox remains the source of truth for events.

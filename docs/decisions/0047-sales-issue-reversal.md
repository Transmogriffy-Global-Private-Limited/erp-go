# ADR 0047: Sales Issue Reversal

## Status

Accepted

## Context

Posted Sales Issues create immutable negative Inventory movements and advance Sales Order fulfillment. Dispatch mistakes must be correctable without deleting issue documents or editing posted stock facts.

## Decision

Add explicit Sales Issue reversal.

New endpoint:

- POST /api/v1/sales/issues/{sales_issue_id}/reverse

New permission:

- sales.issue.reverse

Reversal requires a non-empty reason and a posted, tenant-owned Sales Issue. It:

- preserves the original issue and negative movement
- appends a positive `issue_reversal` Inventory movement
- marks the issue reversed with actor, reason, and timestamp
- excludes reversed issues from fulfillment calculations
- recalculates the order as confirmed, partially_fulfilled, or fulfilled
- writes Inventory, Sales Issue, and Sales Order audit/outbox records atomically

Inventory owns creation of the compensating stock movement. Sales owns issue status and order progress.

## Consequences

Dispatch errors have a traceable compensating path without rewriting ledger history.

A Sales Issue can be reversed only once. Migration rollback is refused while reversed issues exist because removing reversal metadata would invalidate retained stock facts.

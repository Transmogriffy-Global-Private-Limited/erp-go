# ADR 0046: Posted Sales Issues and Fulfillment Progress

## Status

Accepted

## Context

Confirmed Sales Orders express commercial intent but do not change stock. Fulfillment needs an auditable document that reduces stock, prevents over-fulfillment, and respects Inventory ownership.

Availability checks and negative stock ledger writes must be atomic with Sales Issue creation and Sales Order progress changes.

## Decision

Add posted, tenant-scoped Sales Issues and Sales Issue Lines.

New tables:

- sales.issues
- sales.issue_lines

New endpoints:

- GET /api/v1/sales/issues
- POST /api/v1/sales/issues

New permissions:

- sales.issue.read
- sales.issue.write

Sales validates that the order is confirmed or partially fulfilled, that each item is ordered, and that the requested quantity does not exceed the order remainder.

Inventory owns the stock command. It serializes availability checks per tenant, item, and location, then appends a posted negative `issue` movement. Sales Issue rows, Inventory movement rows, audit entries, outbox events, and order progress commit in one tenant-scoped transaction.

Sales Orders derive issued and remaining quantities from posted issue lines and transition through:

- confirmed
- partially_fulfilled
- fulfilled

## Consequences

Sales fulfillment cannot create negative availability through the Sales Issue path and cannot issue unordered or excess quantity.

Posted issue history is immutable in this slice. Corrections require a future compensating Sales Issue reversal rather than editing or deleting stock facts.

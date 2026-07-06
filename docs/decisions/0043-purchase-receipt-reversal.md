# ADR 0043: Purchase Receipt Reversal

## Status

Accepted

## Context

Posted Purchase Receipts create immutable positive Inventory stock movements and advance Purchase Order receipt progress. Operational mistakes must be correctable without deleting receipt history or editing posted ledger entries.

## Decision

Add explicit Purchase Receipt reversal.

New endpoint:

- POST /api/v1/purchase/receipts/{receipt_id}/reverse

Reversal requires:

- purchase.receipt.reverse permission
- a non-empty reason
- a posted, tenant-owned, Purchase-Order-backed receipt

The reversal transaction:

- locks the receipt and Purchase Order
- creates a posted receipt_reversal Inventory movement
- copies receipt lines as negative stock deltas
- marks the receipt reversed without deleting it
- excludes reversed receipts from ordered/received/remaining calculations
- recalculates the Purchase Order to approved, partially_received, or received
- writes audit and outbox records

The original receipt and original stock movement remain immutable.

## Consequences

Receipt mistakes can be corrected through compensating ledger facts with complete traceability.

Reversal is idempotent at the business level: a reversed receipt cannot be reversed again.

Migration rollback is refused while reversed receipts exist because removing the feature would invalidate retained ledger facts.

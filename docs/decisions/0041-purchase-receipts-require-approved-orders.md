# ADR 0041: Purchase Receipts Require Approved Purchase Orders

## Status

Accepted

## Context

Purchase Orders now record approved purchasing commitments, while Purchase Receipts currently accept a free-text supplier name and can receive arbitrary tenant items.

That gap allows receipts with no approved commitment, supplier mismatch, unordered items, or quantities above the order.

## Decision

New Purchase Receipts must reference an approved tenant Purchase Order using purchase_order_id.

The receipt transaction:

- locks the approved Purchase Order to serialize concurrent receipts
- derives and snapshots the supplier from the order
- accepts only items present on the order
- sums ordered quantities for repeated item lines
- subtracts quantities from earlier posted receipts
- rejects quantities above the remaining ordered quantity
- creates the receipt and Inventory stock movement atomically

The database columns remain nullable so existing local receipt rows survive migration. The API requires purchase_order_id for every new receipt.

## Consequences

Purchase receiving now has an enforced approved-order boundary and cannot over-receive through concurrent requests.

Inventory continues to own stock movements; Purchase only creates them as part of the receipt command transaction.

Future work can expose ordered, received, and remaining quantities and move Purchase Orders to partially_received or received lifecycle states.

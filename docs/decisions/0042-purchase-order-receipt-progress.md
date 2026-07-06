# ADR 0042: Purchase Order Receipt Progress

## Status

Accepted

## Context

Purchase Receipts are linked to approved Purchase Orders and cannot exceed ordered quantities, but Purchase Order responses do not show received or remaining quantities and their status remains approved after receiving begins.

Callers need authoritative progress without reconstructing it from receipts.

## Decision

Derive receipt progress from posted Purchase Receipt lines.

Each Purchase Order line exposes:

- quantity: ordered quantity
- received_quantity
- remaining_quantity

Purchase Order status transitions automatically inside the receipt transaction:

- approved to partially_received after an incomplete receipt
- approved or partially_received to received after all ordered quantities are received

Each item may appear only once per Purchase Order. This keeps item-level receipt allocation unambiguous because receipt lines currently reference an order and item rather than a specific order line.

Status transitions write audit and outbox records atomically with the receipt and Inventory stock movement.

## Consequences

Purchase Order APIs expose current receipt progress without mutable quantity counters.

Progress remains derived from receipt history, while status provides a durable workflow projection.

Future receipt reversal must append compensating Inventory movements and recalculate Purchase Order progress in the same transaction.

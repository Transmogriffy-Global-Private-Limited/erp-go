package outbox

// Package outbox will own durable event publishing.
//
// Business state changes that need events should write:
// - business table changes
// - outbox event row
//
// in the same database transaction.
//
// A worker will later publish pending outbox events to the event bus.

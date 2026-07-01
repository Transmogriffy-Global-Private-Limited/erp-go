package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	TenantID      string
	EventType     string
	EventVersion  int
	AggregateType string
	AggregateID   string
	Payload       any
}

type EventRecord struct {
	ID            string
	TenantID      string
	EventType     string
	EventVersion  int
	AggregateType string
	AggregateID   string
	Payload       json.RawMessage
	Status        string
	AttemptCount  int
	CreatedAt     time.Time
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) Store {
	return Store{
		db: db,
	}
}

func Insert(ctx context.Context, tx pgx.Tx, event Event) error {
	payloadJSON := "{}"

	if event.Payload != nil {
		encoded, err := json.Marshal(event.Payload)
		if err != nil {
			return fmt.Errorf("marshal outbox payload: %w", err)
		}

		payloadJSON = string(encoded)
	}

	_, err := tx.Exec(ctx, `
INSERT INTO core.outbox_events (
tenant_id,
event_type,
event_version,
aggregate_type,
aggregate_id,
payload
)
VALUES (
$1,
$2,
$3,
$4,
$5,
$6::jsonb
)
`, event.TenantID, event.EventType, event.EventVersion, event.AggregateType, event.AggregateID, payloadJSON)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}

	return nil
}

func (s Store) ClaimPending(ctx context.Context, limit int) ([]EventRecord, error) {
	if limit <= 0 {
		limit = 10
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin outbox claim transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	rows, err := tx.Query(ctx, `
WITH claimed AS (
SELECT id
FROM core.outbox_events
WHERE status IN ('pending', 'failed')
ORDER BY created_at ASC, id ASC
LIMIT $1
FOR UPDATE SKIP LOCKED
)
UPDATE core.outbox_events e
SET status = 'publishing',
    attempt_count = attempt_count + 1
FROM claimed
WHERE e.id = claimed.id
RETURNING
e.id::text,
COALESCE(e.tenant_id::text, '') AS tenant_id,
e.event_type,
e.event_version,
e.aggregate_type,
e.aggregate_id,
e.payload::text,
e.status,
e.attempt_count,
e.created_at
`, limit)
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}
	defer rows.Close()

	events := make([]EventRecord, 0)

	for rows.Next() {
		var event EventRecord
		var payloadText string

		if err := rows.Scan(
			&event.ID,
			&event.TenantID,
			&event.EventType,
			&event.EventVersion,
			&event.AggregateType,
			&event.AggregateID,
			&payloadText,
			&event.Status,
			&event.AttemptCount,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan claimed outbox event: %w", err)
		}

		event.Payload = json.RawMessage(payloadText)
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed outbox events: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit outbox claim transaction: %w", err)
	}

	return events, nil
}

func (s Store) MarkPublished(ctx context.Context, eventID string) error {
	_, err := s.db.Exec(ctx, `
UPDATE core.outbox_events
SET status = 'published',
    published_at = now()
WHERE id = $1
`, eventID)
	if err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}

	return nil
}

func (s Store) MarkFailed(ctx context.Context, eventID string) error {
	_, err := s.db.Exec(ctx, `
UPDATE core.outbox_events
SET status = 'failed'
WHERE id = $1
`, eventID)
	if err != nil {
		return fmt.Errorf("mark outbox event failed: %w", err)
	}

	return nil
}

package outbox

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Event struct {
	TenantID      string
	EventType     string
	EventVersion  int
	AggregateType string
	AggregateID   string
	Payload       any
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

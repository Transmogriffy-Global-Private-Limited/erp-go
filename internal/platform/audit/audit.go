package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Entry struct {
	TenantID   string
	ActorType  string
	ActorID    string
	Action     string
	TargetType string
	TargetID   string
	Reason     string
	Metadata   any
}

func Insert(ctx context.Context, tx pgx.Tx, entry Entry) error {
	metadataJSON := "{}"

	if entry.Metadata != nil {
		encoded, err := json.Marshal(entry.Metadata)
		if err != nil {
			return fmt.Errorf("marshal audit metadata: %w", err)
		}

		metadataJSON = string(encoded)
	}

	_, err := tx.Exec(ctx, `
INSERT INTO audit.audit_log (
tenant_id,
actor_type,
actor_id,
action,
target_type,
target_id,
reason,
metadata
)
VALUES (
$1,
$2,
$3,
$4,
$5,
$6,
$7,
$8::jsonb
)
`, entry.TenantID, entry.ActorType, entry.ActorID, entry.Action, entry.TargetType, entry.TargetID, entry.Reason, metadataJSON)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}

	return nil
}

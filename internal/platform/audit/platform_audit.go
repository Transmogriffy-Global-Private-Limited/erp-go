package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type PlatformEntry struct {
	PlatformActorID string
	Action          string
	TargetType      string
	TargetID        string
	TenantID        string
	Reason          string
	Metadata        any
}

func InsertPlatform(ctx context.Context, tx pgx.Tx, entry PlatformEntry) error {
	metadataJSON := "{}"

	if entry.Metadata != nil {
		encoded, err := json.Marshal(entry.Metadata)
		if err != nil {
			return fmt.Errorf("marshal platform audit metadata: %w", err)
		}

		metadataJSON = string(encoded)
	}

	_, err := tx.Exec(ctx, `
INSERT INTO control.platform_audit_log (
platform_actor_id,
action,
target_type,
target_id,
tenant_id,
reason,
metadata
)
VALUES (
$1,
$2,
$3,
$4,
NULLIF($5, '')::uuid,
$6,
$7::jsonb
)
`, entry.PlatformActorID, entry.Action, entry.TargetType, entry.TargetID, entry.TenantID, entry.Reason, metadataJSON)
	if err != nil {
		return fmt.Errorf("insert platform audit log: %w", err)
	}

	return nil
}

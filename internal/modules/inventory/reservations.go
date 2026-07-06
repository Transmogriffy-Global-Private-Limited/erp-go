package inventory

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/audit"
	platformdb "github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/outbox"
	"github.com/jackc/pgx/v5"
)

var (
	ErrReservationReferenceNotFound     = errors.New("reservation item or location not found or inactive")
	ErrReservationInsufficientAvailable = errors.New("insufficient available stock for reservation")
	ErrReservationNotFound              = errors.New("reservation not found")
	ErrReservationNotActive             = errors.New("reservation is not active")
)

type Reservation struct {
	ID                string     `json:"id"`
	ReservationNumber string     `json:"reservation_number"`
	ItemID            string     `json:"item_id"`
	ItemSKU           string     `json:"item_sku"`
	ItemName          string     `json:"item_name"`
	LocationID        string     `json:"location_id"`
	LocationCode      string     `json:"location_code"`
	Quantity          string     `json:"quantity"`
	Reference         string     `json:"reference"`
	Notes             string     `json:"notes"`
	Status            string     `json:"status"`
	ExpiresAt         *time.Time `json:"expires_at"`
	ReleasedAt        *time.Time `json:"released_at"`
	CreatedAt         time.Time  `json:"created_at"`
}
type CreateReservationInput struct {
	ItemID     string     `json:"item_id"`
	LocationID string     `json:"location_id"`
	Quantity   string     `json:"quantity"`
	Reference  string     `json:"reference"`
	Notes      string     `json:"notes"`
	ExpiresAt  *time.Time `json:"expires_at"`
}
type Availability struct {
	ItemID       string `json:"item_id"`
	ItemSKU      string `json:"item_sku"`
	ItemName     string `json:"item_name"`
	LocationID   string `json:"location_id"`
	LocationCode string `json:"location_code"`
	OnHand       string `json:"on_hand"`
	Reserved     string `json:"reserved"`
	Available    string `json:"available"`
}

func (s Store) ListReservations(ctx context.Context, tenantID string) ([]Reservation, error) {
	v := []Reservation{}
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, e := tx.Query(ctx, `SELECT r.id::text,r.reservation_number,r.item_id::text,i.sku,i.name,r.location_id::text,l.code,r.quantity::text,r.reference,r.notes,r.status,r.expires_at,r.released_at,r.created_at FROM inventory.reservations r JOIN inventory.items i ON i.tenant_id=r.tenant_id AND i.id=r.item_id JOIN inventory.locations l ON l.tenant_id=r.tenant_id AND l.id=r.location_id ORDER BY r.created_at DESC,r.id DESC LIMIT 100`)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var x Reservation
			if e := rows.Scan(&x.ID, &x.ReservationNumber, &x.ItemID, &x.ItemSKU, &x.ItemName, &x.LocationID, &x.LocationCode, &x.Quantity, &x.Reference, &x.Notes, &x.Status, &x.ExpiresAt, &x.ReleasedAt, &x.CreatedAt); e != nil {
				return e
			}
			v = append(v, x)
		}
		return rows.Err()
	})
	return v, err
}
func (s Store) ListAvailability(ctx context.Context, tenantID string) ([]Availability, error) {
	v := []Availability{}
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		rows, e := tx.Query(ctx, `SELECT i.id::text,i.sku,i.name,l.id::text,l.code,onhand.quantity::numeric(18,3)::text,COALESCE(r.quantity,0)::numeric(18,3)::text,GREATEST(onhand.quantity-COALESCE(r.quantity,0),0)::numeric(18,3)::text FROM (SELECT item_id,location_id,SUM(quantity_delta) quantity FROM inventory.stock_movement_lines GROUP BY item_id,location_id) onhand JOIN inventory.items i ON i.id=onhand.item_id JOIN inventory.locations l ON l.id=onhand.location_id LEFT JOIN (SELECT item_id,location_id,SUM(quantity) quantity FROM inventory.reservations WHERE status='active' AND (expires_at IS NULL OR expires_at>now()) GROUP BY item_id,location_id) r ON r.item_id=onhand.item_id AND r.location_id=onhand.location_id ORDER BY i.sku,l.code`)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var x Availability
			if e := rows.Scan(&x.ItemID, &x.ItemSKU, &x.ItemName, &x.LocationID, &x.LocationCode, &x.OnHand, &x.Reserved, &x.Available); e != nil {
				return e
			}
			v = append(v, x)
		}
		return rows.Err()
	})
	return v, err
}
func (s Store) CreateReservation(ctx context.Context, tenantID, actorID string, in CreateReservationInput) (Reservation, error) {
	in.ItemID = strings.TrimSpace(in.ItemID)
	in.LocationID = strings.TrimSpace(in.LocationID)
	in.Quantity = strings.TrimSpace(in.Quantity)
	in.Reference = strings.TrimSpace(in.Reference)
	in.Notes = strings.TrimSpace(in.Notes)
	var v Reservation
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		key := tenantID + ":" + in.ItemID + ":" + in.LocationID
		if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, key); e != nil {
			return e
		}
		if e := tx.QueryRow(ctx, `SELECT i.id::text,i.sku,i.name,l.id::text,l.code FROM inventory.items i JOIN inventory.locations l ON l.tenant_id=i.tenant_id WHERE i.id=$1::uuid AND i.status='active' AND l.id=$2::uuid AND l.status='active'`, in.ItemID, in.LocationID).Scan(&v.ItemID, &v.ItemSKU, &v.ItemName, &v.LocationID, &v.LocationCode); e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return ErrReservationReferenceNotFound
			}
			return e
		}
		var ok bool
		if e := tx.QueryRow(ctx, `SELECT $3::numeric <= (COALESCE((SELECT SUM(quantity_delta) FROM inventory.stock_movement_lines WHERE item_id=$1::uuid AND location_id=$2::uuid),0)-COALESCE((SELECT SUM(quantity) FROM inventory.reservations WHERE item_id=$1::uuid AND location_id=$2::uuid AND status='active' AND (expires_at IS NULL OR expires_at>now())),0))`, in.ItemID, in.LocationID, in.Quantity).Scan(&ok); e != nil {
			return e
		}
		if !ok {
			return ErrReservationInsufficientAvailable
		}
		if e := tx.QueryRow(ctx, `INSERT INTO inventory.reservations(tenant_id,reservation_number,item_id,location_id,quantity,reference,notes,expires_at,created_by) VALUES($1,'RS-'||upper(substr(replace(gen_random_uuid()::text,'-',''),1,12)),$2::uuid,$3::uuid,$4::numeric,$5,$6,$7,$8::uuid) RETURNING id::text,reservation_number,quantity::text,reference,notes,status,expires_at,released_at,created_at`, tenantID, v.ItemID, v.LocationID, in.Quantity, in.Reference, in.Notes, in.ExpiresAt, actorID).Scan(&v.ID, &v.ReservationNumber, &v.Quantity, &v.Reference, &v.Notes, &v.Status, &v.ExpiresAt, &v.ReleasedAt, &v.CreatedAt); e != nil {
			return e
		}
		if e := audit.Insert(ctx, tx, audit.Entry{TenantID: tenantID, ActorType: "tenant_user", ActorID: actorID, Action: "inventory.reservation.create", TargetType: "inventory.reservation", TargetID: v.ID, Metadata: map[string]any{"item_id": v.ItemID, "location_id": v.LocationID, "quantity": v.Quantity, "reference": v.Reference}}); e != nil {
			return e
		}
		return outbox.Insert(ctx, tx, outbox.Event{TenantID: tenantID, EventType: "inventory.stock.reserved.v1", EventVersion: 1, AggregateType: "inventory.reservation", AggregateID: v.ID, Payload: map[string]any{"reservation_id": v.ID, "item_id": v.ItemID, "location_id": v.LocationID, "quantity": v.Quantity, "reference": v.Reference}})
	})
	return v, err
}
func (s Store) ReleaseReservation(ctx context.Context, tenantID, actorID, id string) (Reservation, error) {
	var v Reservation
	err := platformdb.WithTenantTx(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		if e := tx.QueryRow(ctx, `SELECT r.id::text,r.reservation_number,r.item_id::text,i.sku,i.name,r.location_id::text,l.code,r.quantity::text,r.reference,r.notes,r.status,r.expires_at,r.released_at,r.created_at FROM inventory.reservations r JOIN inventory.items i ON i.tenant_id=r.tenant_id AND i.id=r.item_id JOIN inventory.locations l ON l.tenant_id=r.tenant_id AND l.id=r.location_id WHERE r.id=$1::uuid FOR UPDATE OF r`, id).Scan(&v.ID, &v.ReservationNumber, &v.ItemID, &v.ItemSKU, &v.ItemName, &v.LocationID, &v.LocationCode, &v.Quantity, &v.Reference, &v.Notes, &v.Status, &v.ExpiresAt, &v.ReleasedAt, &v.CreatedAt); e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return ErrReservationNotFound
			}
			return e
		}
		if v.Status != "active" {
			return ErrReservationNotActive
		}
		key := tenantID + ":" + v.ItemID + ":" + v.LocationID
		if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, key); e != nil {
			return e
		}
		if e := tx.QueryRow(ctx, `UPDATE inventory.reservations SET status='released',released_by=$2::uuid,released_at=now() WHERE id=$1::uuid RETURNING status,released_at`, v.ID, actorID).Scan(&v.Status, &v.ReleasedAt); e != nil {
			return e
		}
		if e := audit.Insert(ctx, tx, audit.Entry{TenantID: tenantID, ActorType: "tenant_user", ActorID: actorID, Action: "inventory.reservation.release", TargetType: "inventory.reservation", TargetID: v.ID, Metadata: map[string]any{"item_id": v.ItemID, "location_id": v.LocationID, "quantity": v.Quantity}}); e != nil {
			return e
		}
		return outbox.Insert(ctx, tx, outbox.Event{TenantID: tenantID, EventType: "inventory.stock.released.v1", EventVersion: 1, AggregateType: "inventory.reservation", AggregateID: v.ID, Payload: map[string]any{"reservation_id": v.ID, "item_id": v.ItemID, "location_id": v.LocationID, "quantity": v.Quantity}})
	})
	return v, err
}

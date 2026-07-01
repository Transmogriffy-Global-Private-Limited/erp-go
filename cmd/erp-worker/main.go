package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/db"
	"github.com/Transmogriffy-Global-Private-Limited/erp-go/internal/platform/outbox"
)

func main() {
	once := flag.Bool("once", false, "process one batch and exit")
	limit := flag.Int("limit", envInt("ERP_WORKER_BATCH_SIZE", 10), "maximum events to process per batch")
	interval := flag.Duration("interval", 5*time.Second, "poll interval")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	databaseURL := env("ERP_DATABASE_URL", "")

	pool, err := db.Open(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	store := outbox.NewStore(pool)

	log.Printf("erp-worker started once=%v limit=%d interval=%s", *once, *limit, interval.String())

	if *once {
		if err := processBatch(ctx, store, *limit); err != nil {
			log.Fatal(err)
		}
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		if err := processBatch(ctx, store, *limit); err != nil {
			log.Printf("process batch failed: %v", err)
		}

		select {
		case <-ctx.Done():
			log.Println("erp-worker shutting down")
			return
		case <-ticker.C:
		}
	}
}

func processBatch(ctx context.Context, store outbox.Store, limit int) error {
	events, err := store.ClaimPending(ctx, limit)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		log.Println("outbox: no events to process")
		return nil
	}

	for _, event := range events {
		log.Printf(
			"outbox: publishing id=%s tenant=%s type=%s aggregate=%s/%s payload=%s",
			event.ID,
			event.TenantID,
			event.EventType,
			event.AggregateType,
			event.AggregateID,
			string(event.Payload),
		)

		if err := store.MarkPublished(ctx, event.ID); err != nil {
			_ = store.MarkFailed(ctx, event.ID)
			return err
		}

		log.Printf("outbox: published id=%s", event.ID)
	}

	return nil
}

func env(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/akordium-id/mergaite-core/internal/core/domain/event"
)

type Config struct {
	PollInterval time.Duration
	BatchSize    int32
	MaxRetries   int32
}

func DefaultConfig() Config {
	return Config{
		PollInterval: 1 * time.Second,
		BatchSize:    50,
		MaxRetries:   5,
	}
}

// OutboxWorker is a background service that continuously polls and dispatches pending events from PostgreSQL.
type OutboxWorker struct {
	outboxRepo  event.OutboxRepository
	bus         event.Bus
	cfg         Config
	triggerChan chan struct{}
}

// NewOutboxWorker constructs a new OutboxWorker.
func NewOutboxWorker(outboxRepo event.OutboxRepository, bus event.Bus, cfg Config) *OutboxWorker {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 1 * time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 5
	}

	return &OutboxWorker{
		outboxRepo:  outboxRepo,
		bus:         bus,
		cfg:         cfg,
		triggerChan: make(chan struct{}, 100),
	}
}

// Trigger wakes up the worker immediately to process new events without waiting for the next ticker tick.
func (w *OutboxWorker) Trigger() {
	select {
	case w.triggerChan <- struct{}{}:
	default:
		// Trigger queue full, ticker or ongoing batch will pick it up
	}
}

// Start runs the worker event processing loop until ctx is cancelled.
func (w *OutboxWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	slog.Info("outbox worker started", slog.Duration("poll_interval", w.cfg.PollInterval))

	for {
		select {
		case <-ctx.Done():
			slog.Info("outbox worker stopping due to context cancellation")
			return
		case <-ticker.C:
			w.ProcessBatch(ctx)
		case <-w.triggerChan:
			w.ProcessBatch(ctx)
		}
	}
}

// ProcessBatch reads and publishes one batch of pending outbox events.
func (w *OutboxWorker) ProcessBatch(ctx context.Context) (int, error) {
	events, err := w.outboxRepo.FetchPending(ctx, w.cfg.MaxRetries, w.cfg.BatchSize)
	if err != nil {
		slog.Error("failed to fetch pending outbox events", slog.Any("error", err))
		return 0, err
	}

	if len(events) == 0 {
		return 0, nil
	}

	processedCount := 0
	for _, oEvt := range events {
		pubErr := w.bus.Publish(ctx, oEvt.ToEvent())
		now := time.Now().UTC()

		if pubErr != nil {
			slog.Warn("failed to dispatch outbox event to bus",
				slog.String("event_id", oEvt.ID.String()),
				slog.String("event_type", oEvt.EventType),
				slog.Any("error", pubErr),
			)
			_ = w.outboxRepo.MarkFailed(ctx, oEvt.ID, pubErr.Error())
		} else {
			if err := w.outboxRepo.MarkPublished(ctx, oEvt.ID, now); err != nil {
				slog.Error("failed to mark outbox event as published",
					slog.String("event_id", oEvt.ID.String()),
					slog.Any("error", err),
				)
			} else {
				processedCount++
			}
		}
	}

	return processedCount, nil
}

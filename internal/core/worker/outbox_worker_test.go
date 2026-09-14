package worker_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/akordium-id/mergaite-core/internal/core/domain/event"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
	"github.com/akordium-id/mergaite-core/internal/core/worker"
	"github.com/akordium-id/mergaite-core/pkg/eventbus"
)

type mockOutboxRepo struct {
	mu        sync.Mutex
	events    map[string]*event.OutboxEvent
	published map[string]time.Time
	failed    map[string]string
}

func newMockOutboxRepo() *mockOutboxRepo {
	return &mockOutboxRepo{
		events:    make(map[string]*event.OutboxEvent),
		published: make(map[string]time.Time),
		failed:    make(map[string]string),
	}
}

func (m *mockOutboxRepo) Create(ctx context.Context, evt *event.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events[evt.ID.String()] = evt
	return nil
}

func (m *mockOutboxRepo) FetchPending(ctx context.Context, maxRetries, limit int32) ([]event.OutboxEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []event.OutboxEvent
	for _, e := range m.events {
		if (e.Status == event.OutboxStatusPending || e.Status == event.OutboxStatusFailed) && e.RetryCount < maxRetries {
			result = append(result, *e)
			if int32(len(result)) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockOutboxRepo) MarkPublished(ctx context.Context, id shared.ID, publishedAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.events[id.String()]; ok {
		e.Status = event.OutboxStatusPublished
		e.PublishedAt = &publishedAt
		m.published[id.String()] = publishedAt
	}
	return nil
}

func (m *mockOutboxRepo) MarkFailed(ctx context.Context, id shared.ID, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.events[id.String()]; ok {
		e.Status = event.OutboxStatusFailed
		e.RetryCount++
		e.ErrorMessage = errMsg
		m.failed[id.String()] = errMsg
	}
	return nil
}

func TestOutboxWorker_ProcessBatch_SuccessAndFailure(t *testing.T) {
	repo := newMockOutboxRepo()
	bus := eventbus.NewInMemoryBus()

	var successEvtCount int32
	bus.Subscribe("order.created", func(ctx context.Context, evt event.Event) error {
		atomic.AddInt32(&successEvtCount, 1)
		return nil
	})

	bus.Subscribe("order.failing", func(ctx context.Context, evt event.Event) error {
		return errors.New("external delivery failed")
	})

	tenantID, _ := shared.NewID()
	doc1ID, _ := shared.NewID()
	doc2ID, _ := shared.NewID()

	evt1ID, _ := shared.NewID()
	evt2ID, _ := shared.NewID()

	// 1. Success event
	_ = repo.Create(context.Background(), &event.OutboxEvent{
		ID:            evt1ID,
		TenantID:      tenantID,
		EventType:     "order.created",
		AggregateType: "order",
		AggregateID:   doc1ID,
		Status:        event.OutboxStatusPending,
		CreatedAt:     time.Now().UTC(),
	})

	// 2. Failing event
	_ = repo.Create(context.Background(), &event.OutboxEvent{
		ID:            evt2ID,
		TenantID:      tenantID,
		EventType:     "order.failing",
		AggregateType: "order",
		AggregateID:   doc2ID,
		Status:        event.OutboxStatusPending,
		CreatedAt:     time.Now().UTC(),
	})

	w := worker.NewOutboxWorker(repo, bus, worker.Config{
		PollInterval: 100 * time.Millisecond,
		BatchSize:    10,
		MaxRetries:   3,
	})

	count, err := w.ProcessBatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error processing batch: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 successfully processed event, got %d", count)
	}
	if atomic.LoadInt32(&successEvtCount) != 1 {
		t.Errorf("expected handler called once, got %d", successEvtCount)
	}

	// Verify status in repo
	if repo.events[evt1ID.String()].Status != event.OutboxStatusPublished {
		t.Errorf("expected evt1 status published, got %s", repo.events[evt1ID.String()].Status)
	}
	if repo.events[evt2ID.String()].Status != event.OutboxStatusFailed {
		t.Errorf("expected evt2 status failed, got %s", repo.events[evt2ID.String()].Status)
	}
	if repo.events[evt2ID.String()].RetryCount != 1 {
		t.Errorf("expected evt2 retry count 1, got %d", repo.events[evt2ID.String()].RetryCount)
	}
}

func TestOutboxWorker_StartAndTrigger(t *testing.T) {
	repo := newMockOutboxRepo()
	bus := eventbus.NewInMemoryBus()

	var triggerCount int32
	bus.Subscribe("immediate.ping", func(ctx context.Context, evt event.Event) error {
		atomic.AddInt32(&triggerCount, 1)
		return nil
	})

	tenantID, _ := shared.NewID()
	pingID, _ := shared.NewID()
	evtID, _ := shared.NewID()

	_ = repo.Create(context.Background(), &event.OutboxEvent{
		ID:            evtID,
		TenantID:      tenantID,
		EventType:     "immediate.ping",
		AggregateType: "ping",
		AggregateID:   pingID,
		Status:        event.OutboxStatusPending,
		CreatedAt:     time.Now().UTC(),
	})

	w := worker.NewOutboxWorker(repo, bus, worker.Config{
		PollInterval: 10 * time.Second, // long ticker, relies on Trigger()
		BatchSize:    10,
		MaxRetries:   3,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)

	// Trigger immediate execution
	w.Trigger()

	// Wait briefly for worker to pick up
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&triggerCount) != 1 {
		t.Errorf("expected immediate trigger to process event, got %d", triggerCount)
	}
}

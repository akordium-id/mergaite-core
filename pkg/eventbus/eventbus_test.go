package eventbus_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/akordium-id/mergaite-core/internal/core/domain/event"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
	"github.com/akordium-id/mergaite-core/pkg/eventbus"
)

func TestInMemoryBus_ExactAndWildcardSubscriptions(t *testing.T) {
	bus := eventbus.NewInMemoryBus()

	var exactCount int32
	var prefixCount int32
	var globalCount int32

	bus.Subscribe("document.created", func(ctx context.Context, evt event.Event) error {
		atomic.AddInt32(&exactCount, 1)
		return nil
	})

	bus.Subscribe("document.*", func(ctx context.Context, evt event.Event) error {
		atomic.AddInt32(&prefixCount, 1)
		return nil
	})

	bus.Subscribe("*", func(ctx context.Context, evt event.Event) error {
		atomic.AddInt32(&globalCount, 1)
		return nil
	})

	tenantID, _ := shared.NewID()
	docID, _ := shared.NewID()

	// 1. Publish document.created -> should trigger all 3 handlers
	evt1 := event.NewBaseEvent(tenantID, "document.created", "document", docID, map[string]any{"num": "SO-01"})
	if err := bus.Publish(context.Background(), evt1); err != nil {
		t.Fatalf("failed to publish evt1: %v", err)
	}

	if atomic.LoadInt32(&exactCount) != 1 {
		t.Errorf("expected exactCount=1, got %d", exactCount)
	}
	if atomic.LoadInt32(&prefixCount) != 1 {
		t.Errorf("expected prefixCount=1, got %d", prefixCount)
	}
	if atomic.LoadInt32(&globalCount) != 1 {
		t.Errorf("expected globalCount=1, got %d", globalCount)
	}

	// 2. Publish document.transitioned -> should trigger prefixCount and globalCount only
	evt2 := event.NewBaseEvent(tenantID, "document.transitioned", "document", docID, map[string]any{"status": "approved"})
	if err := bus.Publish(context.Background(), evt2); err != nil {
		t.Fatalf("failed to publish evt2: %v", err)
	}

	if atomic.LoadInt32(&exactCount) != 1 {
		t.Errorf("expected exactCount=1, got %d", exactCount)
	}
	if atomic.LoadInt32(&prefixCount) != 2 {
		t.Errorf("expected prefixCount=2, got %d", prefixCount)
	}
	if atomic.LoadInt32(&globalCount) != 2 {
		t.Errorf("expected globalCount=2, got %d", globalCount)
	}

	// 3. Publish party.created -> should trigger globalCount only
	evt3 := event.NewBaseEvent(tenantID, "party.created", "party", docID, nil)
	if err := bus.Publish(context.Background(), evt3); err != nil {
		t.Fatalf("failed to publish evt3: %v", err)
	}

	if atomic.LoadInt32(&exactCount) != 1 {
		t.Errorf("expected exactCount=1, got %d", exactCount)
	}
	if atomic.LoadInt32(&prefixCount) != 2 {
		t.Errorf("expected prefixCount=2, got %d", prefixCount)
	}
	if atomic.LoadInt32(&globalCount) != 3 {
		t.Errorf("expected globalCount=3, got %d", globalCount)
	}
}

func TestInMemoryBus_PanicRecoveryAndErrorReporting(t *testing.T) {
	bus := eventbus.NewInMemoryBus()

	bus.Subscribe("fail.event", func(ctx context.Context, evt event.Event) error {
		panic("unexpected crash in listener")
	})

	bus.Subscribe("fail.event", func(ctx context.Context, evt event.Event) error {
		return errors.New("business rule validation error")
	})

	tenantID, _ := shared.NewID()
	aggID, _ := shared.NewID()
	evt := event.NewBaseEvent(tenantID, "fail.event", "test", aggID, nil)

	err := bus.Publish(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

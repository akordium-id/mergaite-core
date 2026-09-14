package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/akordium-id/mergaite-core/internal/core/domain/event"
)

type inMemoryBus struct {
	mu       sync.RWMutex
	handlers map[string][]event.Handler
}

// NewInMemoryBus constructs a thread-safe in-memory event bus.
func NewInMemoryBus() event.Bus {
	return &inMemoryBus{
		handlers: make(map[string][]event.Handler),
	}
}

func (b *inMemoryBus) Subscribe(eventType string, handler event.Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

func (b *inMemoryBus) Publish(ctx context.Context, evt event.Event) error {
	b.mu.RLock()
	var targets []event.Handler

	// Exact match
	if hList, ok := b.handlers[evt.EventType()]; ok {
		targets = append(targets, hList...)
	}

	// Wildcard matches (e.g. "document.*" or "*")
	for pattern, hList := range b.handlers {
		if pattern == "*" {
			targets = append(targets, hList...)
		} else if before, ok := strings.CutSuffix(pattern, ".*"); ok {
			prefix := before
			if strings.HasPrefix(evt.EventType(), prefix+".") {
				targets = append(targets, hList...)
			}
		}
	}
	b.mu.RUnlock()

	var errs []error
	for _, h := range targets {
		if err := executeHandlerSafely(ctx, h, evt); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("event handling errors: %d handler(s) failed", len(errs))
	}
	return nil
}

func executeHandlerSafely(ctx context.Context, h event.Handler, evt event.Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("event handler panicked", slog.Any("panic", r), slog.String("event", evt.EventType()))
			err = fmt.Errorf("handler panic: %v", r)
		}
	}()
	return h(ctx, evt)
}

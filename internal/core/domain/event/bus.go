package event

import (
	"context"
)

// Handler represents an asynchronous or synchronous event consumer.
type Handler func(ctx context.Context, evt Event) error

// Bus defines the event publish/subscribe contract.
type Bus interface {
	Publish(ctx context.Context, evt Event) error
	Subscribe(eventType string, handler Handler)
}

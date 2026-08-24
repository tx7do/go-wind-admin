package eventbus

import (
	"context"
	"fmt"
	"sync"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
)

// EventBus manages event subscriptions and publishing
type EventBus interface {
	// Subscribe registers a handler for a specific event type
	Subscribe(eventType string, handler Handler) error

	// SubscribeAsync registers an async handler for a specific event type
	SubscribeAsync(eventType string, handler Handler) error

	// SubscribeOnce registers a handler that will be called only once
	SubscribeOnce(eventType string, handler Handler) error

	// Unsubscribe removes a handler for a specific event type
	Unsubscribe(eventType string, handler Handler) error

	// Publish publishes an event to all subscribed handlers
	Publish(ctx context.Context, event *Event) error

	// PublishAsync publishes an event asynchronously
	PublishAsync(ctx context.Context, event *Event) error

	// Close closes the event bus and cleans up resources
	Close() error
}

// DefaultEventBus is the default implementation of EventBus
type DefaultEventBus struct {
	mu           sync.RWMutex
	handlers     map[string][]Handler
	onceHandlers map[string][]Handler
	logger       *bLogger.Helper
	closed       bool
}

// NewEventBus creates a new event bus
func NewEventBus(logger bLogger.Logger) EventBus {
	return &DefaultEventBus{
		handlers:     make(map[string][]Handler),
		onceHandlers: make(map[string][]Handler),
		logger:       bLogger.NewHelper(logger.With("module", "eventbus")),
	}
}

// Subscribe registers a handler for a specific event type
func (eb *DefaultEventBus) Subscribe(eventType string, handler Handler) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return fmt.Errorf("event bus is closed")
	}

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	// Removed verbose debug log
	return nil
}

// SubscribeAsync registers an async handler for a specific event type
func (eb *DefaultEventBus) SubscribeAsync(eventType string, handler Handler) error {
	asyncHandler := NewAsyncHandler(handler)
	return eb.Subscribe(eventType, asyncHandler)
}

// SubscribeOnce registers a handler that will be called only once
func (eb *DefaultEventBus) SubscribeOnce(eventType string, handler Handler) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return fmt.Errorf("event bus is closed")
	}

	eb.onceHandlers[eventType] = append(eb.onceHandlers[eventType], handler)
	// Removed verbose debug log
	return nil
}

// Unsubscribe removes a handler for a specific event type
func (eb *DefaultEventBus) Unsubscribe(eventType string, handler Handler) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if handlers, exists := eb.handlers[eventType]; exists {
		for i, h := range handlers {
			if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
				eb.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
				// Removed verbose debug log
				return nil
			}
		}
	}

	return fmt.Errorf("handler not found for event type: %s", eventType)
}

// Publish publishes an event to all subscribed handlers
func (eb *DefaultEventBus) Publish(ctx context.Context, event *Event) error {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if eb.closed {
			eb.logger.Warnf(ctx, "❌ Event bus is closed, cannot publish event: %s", event.Type)
		return fmt.Errorf("event bus is closed")
	}

	// Get handlers for this event type
	handlers := eb.handlers[event.Type]
	onceHandlers := eb.onceHandlers[event.Type]

	totalHandlers := len(handlers) + len(onceHandlers)
	if totalHandlers == 0 {
		// No handlers registered - this is normal, don't log
		return nil
	}

	// Execute regular handlers
	for _, handler := range handlers {
		if err := handler.Handle(ctx, event); err != nil {
				eb.logger.Errorf(ctx, "Handler error for event %s: %v", event.Type, err)
			// Continue with other handlers even if one fails
		}
	}

	// Execute once handlers
	for _, handler := range onceHandlers {
		if err := handler.Handle(ctx, event); err != nil {
				eb.logger.Errorf(ctx, "Once handler error for event %s: %v", event.Type, err)
		}
	}

	// Remove once handlers after execution
	if len(onceHandlers) > 0 {
		eb.mu.RUnlock()
		eb.mu.Lock()
		delete(eb.onceHandlers, event.Type)
		eb.mu.Unlock()
		eb.mu.RLock()
	}

	// Removed verbose debug log - events publish successfully by default
	return nil
}

// PublishAsync publishes an event asynchronously
// Uses a fresh background context so the goroutine isn't affected by caller context cancellation
func (eb *DefaultEventBus) PublishAsync(ctx context.Context, event *Event) error {
	go func() {
		// Use fresh Background context with timeout to prevent indefinite blocking
		// while ensuring the async operation isn't cancelled when caller returns
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := eb.Publish(bgCtx, event); err != nil {
			eb.logger.Errorf(bgCtx, "Async publish error for event %s: %v", event.Type, err)
		}
	}()
	return nil
}

// Close closes the event bus and cleans up resources
func (eb *DefaultEventBus) Close() error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return fmt.Errorf("event bus already closed")
	}

	eb.closed = true
	eb.handlers = make(map[string][]Handler)
	eb.onceHandlers = make(map[string][]Handler)
	eb.logger.Info(context.Background(), "Event bus closed")

	return nil
}

// GetSubscriberCount returns the number of subscribers for an event type
func (eb *DefaultEventBus) GetSubscriberCount(eventType string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	return len(eb.handlers[eventType]) + len(eb.onceHandlers[eventType])
}

// GetEventTypes returns all event types that have subscribers
func (eb *DefaultEventBus) GetEventTypes() []string {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	types := make(map[string]struct{})
	for eventType := range eb.handlers {
		types[eventType] = struct{}{}
	}
	for eventType := range eb.onceHandlers {
		types[eventType] = struct{}{}
	}

	result := make([]string, 0, len(types))
	for eventType := range types {
		result = append(result, eventType)
	}

	return result
}

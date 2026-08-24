package eventbus_test

import (
	"context"
	"fmt"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"go-wind-admin/pkg/eventbus"
)

// Example_basic demonstrates basic event bus usage
func Example_basic() {
	// Create event bus
	bus := eventbus.NewEventBus(bLogger.NopLogger())
	defer bus.Close()

	// Subscribe to events
	bus.Subscribe("user.created", eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		fmt.Printf("User created: %v\n", event.Data)
		return nil
	}))

	// Publish event
	ctx := context.Background()
	event := eventbus.NewEvent("user.created", map[string]interface{}{
		"username": "john_doe",
		"email":    "john@example.com",
	})

	bus.Publish(ctx, event)
}

// Example_typedEvents demonstrates using typed event data
func Example_typedEvents() {
	bus := eventbus.NewEventBus(bLogger.NopLogger())
	defer bus.Close()

	// Subscribe with typed data
	bus.Subscribe(eventbus.EventEmailReceived, eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		var emailData eventbus.EmailReceivedEvent
		if err := event.GetData(&emailData); err != nil {
			return err
		}

		fmt.Printf("Email received from %s: %s\n", emailData.From, emailData.Subject)
		return nil
	}))

	// Publish typed event
	ctx := context.Background()
	emailEvent := eventbus.NewEvent(eventbus.EventEmailReceived, eventbus.EmailReceivedEvent{
		EmailID:   "email-123",
		From:      "sender@example.com",
		To:        "receiver@example.com",
		Subject:   "Hello World",
		Mailbox:   "INBOX",
		AccountID: "account-456",
	})

	bus.Publish(ctx, emailEvent)
}

// Example_middleware demonstrates using middleware
func Example_middleware() {
	logger := bLogger.NewHelper(bLogger.NopLogger())
	bus := eventbus.NewEventBus(bLogger.NopLogger())
	defer bus.Close()

	// Create handler with middleware
	baseHandler := eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		fmt.Printf("Processing event: %s\n", event.Type)
		return nil
	})

	// Chain middlewares
	middleware := eventbus.Chain(
		eventbus.LoggingMiddleware(logger),
		eventbus.RecoveryMiddleware(logger),
		eventbus.TimeoutMiddleware(5*time.Second),
	)

	wrappedHandler := middleware(baseHandler)
	bus.Subscribe("task.started", wrappedHandler)

	// Publish event
	ctx := context.Background()
	event := eventbus.NewEvent("task.started", map[string]interface{}{
		"task_id": "task-123",
	})

	bus.Publish(ctx, event)
}

// Example_manager demonstrates using the event bus manager
func Example_manager() {
	manager := eventbus.NewManager(bLogger.NopLogger())
	defer manager.Close()

	// Subscribe to global bus
	manager.SubscribeGlobal("system.started", eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		fmt.Println("System started!")
		return nil
	}))

	// Use named bus for email events
	emailBus := manager.GetBus("email")
	emailBus.Subscribe(eventbus.EventEmailReceived, eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		fmt.Printf("Email event: %v\n", event.Data)
		return nil
	}))

	// Publish to different buses
	ctx := context.Background()

	systemEvent := eventbus.NewEvent("system.started", nil)
	manager.PublishGlobal(ctx, systemEvent)

	emailEvent := eventbus.NewEvent(eventbus.EventEmailReceived, map[string]interface{}{
		"from": "test@example.com",
	})
	manager.Publish(ctx, "email", emailEvent)

	// Get statistics
	stats := manager.GetStats()
	fmt.Printf("Manager stats: %v\n", stats)
}

// Example_asyncHandlers demonstrates async event handling
func Example_asyncHandlers() {
	bus := eventbus.NewEventBus(bLogger.NopLogger())
	defer bus.Close()

	// Subscribe async handler
	bus.SubscribeAsync("notification.send", eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		// This will run in a separate goroutine
		time.Sleep(100 * time.Millisecond)
		fmt.Println("Notification sent asynchronously")
		return nil
	}))

	// Publish event (returns immediately)
	ctx := context.Background()
	event := eventbus.NewEvent("notification.send", map[string]interface{}{
		"message": "Hello!",
	})

	bus.Publish(ctx, event)
	fmt.Println("Event published, continuing...")

	// Wait a bit for async handler to complete
	time.Sleep(200 * time.Millisecond)
}

// Example_filterHandler demonstrates filtering events
func Example_filterHandler() {
	bus := eventbus.NewEventBus(bLogger.NopLogger())
	defer bus.Close()

	// Only process high-priority events
	filterHandler := eventbus.NewFilterHandler(
		func(event *eventbus.Event) bool {
			return event.Priority >= 5
		},
		eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
			fmt.Printf("Handling high-priority event: %s\n", event.Type)
			return nil
		}),
	)

	bus.Subscribe("alert", filterHandler)

	ctx := context.Background()

	// Low priority - won't be handled
	lowPriorityEvent := eventbus.NewEvent("alert", "Low priority alert").WithPriority(2)
	bus.Publish(ctx, lowPriorityEvent)

	// High priority - will be handled
	highPriorityEvent := eventbus.NewEvent("alert", "Critical alert!").WithPriority(10)
	bus.Publish(ctx, highPriorityEvent)
}

// Example_subscribeOnce demonstrates one-time event handlers
func Example_subscribeOnce() {
	bus := eventbus.NewEventBus(bLogger.NopLogger())
	defer bus.Close()

	counter := 0

	// This handler will only be called once
	bus.SubscribeOnce("init", eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		counter++
		fmt.Printf("Initialization handler called (count: %d)\n", counter)
		return nil
	}))

	ctx := context.Background()

	// Publish multiple times
	for i := 0; i < 3; i++ {
		event := eventbus.NewEvent("init", nil)
		bus.Publish(ctx, event)
	}

	fmt.Printf("Handler was called %d time(s)\n", counter)
}

// Example_chainHandlers demonstrates chaining multiple handlers
func Example_chainHandlers() {
	bus := eventbus.NewEventBus(bLogger.NopLogger())
	defer bus.Close()

	// Create multiple handlers
	handler1 := eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		fmt.Println("Step 1: Validate")
		return nil
	})

	handler2 := eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		fmt.Println("Step 2: Process")
		return nil
	})

	handler3 := eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		fmt.Println("Step 3: Notify")
		return nil
	})

	// Chain them together
	chainHandler := eventbus.NewChainHandler(handler1, handler2, handler3)
	bus.Subscribe("order.created", chainHandler)

	// Publish event - all handlers will execute in sequence
	ctx := context.Background()
	event := eventbus.NewEvent("order.created", map[string]interface{}{
		"order_id": "order-123",
	})

	bus.Publish(ctx, event)
}

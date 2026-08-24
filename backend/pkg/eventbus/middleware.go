package eventbus

import (
	"context"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
)

// Middleware is a function that wraps a Handler
type Middleware func(Handler) Handler

// LoggingMiddleware logs event handling
func LoggingMiddleware(logger *bLogger.Helper) Middleware {
	return func(next Handler) Handler {
		return EventHandlerFunc(func(ctx context.Context, event *Event) error {
			logger.Infof(ctx, "Handling event: type=%s, id=%s, source=%s", event.Type, event.ID, event.Source)
			err := next.Handle(ctx, event)
			if err != nil {
				logger.Errorf(ctx, "Error handling event %s: %v", event.ID, err)
			} else {
				logger.Debugf(ctx, "Successfully handled event: %s", event.ID)
			}
			return err
		})
	}
}

// RecoveryMiddleware recovers from panics in handlers
func RecoveryMiddleware(logger *bLogger.Helper) Middleware {
	return func(next Handler) Handler {
		return EventHandlerFunc(func(ctx context.Context, event *Event) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf(ctx, "Panic recovered in event handler for %s: %v", event.Type, r)
					err = &PanicError{Value: r}
				}
			}()
			return next.Handle(ctx, event)
		})
	}
}

// TimeoutMiddleware adds timeout to event handling
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next Handler) Handler {
		return EventHandlerFunc(func(ctx context.Context, event *Event) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				done <- next.Handle(ctx, event)
			}()

			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				return &TimeoutError{
					EventID:   event.ID,
					EventType: event.Type,
					Timeout:   timeout,
				}
			}
		})
	}
}

// RetryMiddleware retries failed event handling
func RetryMiddleware(maxRetries int, delay time.Duration) Middleware {
	return func(next Handler) Handler {
		return EventHandlerFunc(func(ctx context.Context, event *Event) error {
			var err error
			for i := 0; i <= maxRetries; i++ {
				err = next.Handle(ctx, event)
				if err == nil {
					return nil
				}

				if i < maxRetries {
					time.Sleep(delay)
				}
			}
			return err
		})
	}
}

// MetricsMiddleware collects metrics for event handling
func MetricsMiddleware(logger *bLogger.Helper) Middleware {
	return func(next Handler) Handler {
		return EventHandlerFunc(func(ctx context.Context, event *Event) error {
			start := time.Now()
			err := next.Handle(ctx, event)
			duration := time.Since(start)

			logger.Infof(ctx, "Event handling metrics: type=%s, duration=%s, success=%v",
				event.Type, duration, err == nil)

			return err
		})
	}
}

// Chain chains multiple middlewares together
func Chain(middlewares ...Middleware) Middleware {
	return func(handler Handler) Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}
		return handler
	}
}

// PanicError represents a panic that occurred during event handling
type PanicError struct {
	Value interface{}
}

func (e *PanicError) Error() string {
	return "panic in event handler"
}

// TimeoutError represents a timeout during event handling
type TimeoutError struct {
	EventID   string
	EventType string
	Timeout   time.Duration
}

func (e *TimeoutError) Error() string {
	return "event handling timeout"
}

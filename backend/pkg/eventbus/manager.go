package eventbus

import (
	"context"
	"sync"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
)

// Manager manages multiple event buses and provides a global interface
type Manager struct {
	mu       sync.RWMutex
	buses    map[string]EventBus
	global   EventBus
	logger   *bLogger.Helper
}

// NewManager creates a new event bus manager
func NewManager(logger bLogger.Logger) *Manager {
	l := bLogger.NewHelper(logger.With("module", "eventbus/manager"))
	return &Manager{
		buses:  make(map[string]EventBus),
		global: NewEventBus(logger),
		logger: l,
	}
}

// GetBus returns an event bus by name, creates it if it doesn't exist
func (m *Manager) GetBus(name string) EventBus {
	m.mu.Lock()
	defer m.mu.Unlock()

	if bus, exists := m.buses[name]; exists {
		return bus
	}

	// Create new bus
	bus := NewEventBus(bLogger.GetLogger())
	m.buses[name] = bus
	m.logger.Infof(context.Background(), "Created new event bus: %s", name)

	return bus
}

// Global returns the global event bus
func (m *Manager) Global() EventBus {
	return m.global
}

// Publish publishes an event to a specific bus
func (m *Manager) Publish(ctx context.Context, busName string, event *Event) error {
	bus := m.GetBus(busName)
	return bus.Publish(ctx, event)
}

// PublishGlobal publishes an event to the global bus
func (m *Manager) PublishGlobal(ctx context.Context, event *Event) error {
	return m.global.Publish(ctx, event)
}

// Subscribe subscribes to events on a specific bus
func (m *Manager) Subscribe(busName, eventType string, handler Handler) error {
	bus := m.GetBus(busName)
	return bus.Subscribe(eventType, handler)
}

// SubscribeGlobal subscribes to events on the global bus
func (m *Manager) SubscribeGlobal(eventType string, handler Handler) error {
	return m.global.Subscribe(eventType, handler)
}

// Close closes all event buses
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, bus := range m.buses {
		if err := bus.Close(); err != nil {
			m.logger.Errorf(context.Background(), "Error closing bus %s: %v", name, err)
		}
	}

	if err := m.global.Close(); err != nil {
		m.logger.Errorf(context.Background(), "Error closing global bus: %v", err)
	}

	m.buses = make(map[string]EventBus)
	m.logger.Info(context.Background(), "Event bus manager closed")

	return nil
}

// GetStats returns statistics for all buses
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_buses"] = len(m.buses)

	busStats := make(map[string]interface{})
	for name, bus := range m.buses {
		if defaultBus, ok := bus.(*DefaultEventBus); ok {
			busStats[name] = map[string]interface{}{
				"event_types": defaultBus.GetEventTypes(),
			}
		}
	}
	stats["buses"] = busStats

	if defaultBus, ok := m.global.(*DefaultEventBus); ok {
		stats["global_bus"] = map[string]interface{}{
			"event_types": defaultBus.GetEventTypes(),
		}
	}

	return stats
}

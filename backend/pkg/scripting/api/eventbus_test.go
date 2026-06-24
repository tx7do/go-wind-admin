package api

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	lua "github.com/yuin/gopher-lua"

	"go-wind-admin/pkg/eventbus"
)

func TestEventBusAPI_PublishSubscribe(t *testing.T) {
	manager := eventbus.NewManager(log.DefaultLogger)
	defer manager.Close()

	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	RegisterLogger(L, logger)
	RegisterEventBus(L, manager, logger)

	script := `
		local log = require("kratos_logger")
		local eventbus = require("kratos_eventbus")

		-- Track received events
		local received_events = {}

		-- Subscribe to event
		eventbus.subscribe("user.created", function(event)
			log.info("Received event: " .. event.type)
			table.insert(received_events, event)
		end)

		-- Publish event
		eventbus.publish("user.created", {
			user_id = 123,
			username = "alice"
		})

		-- Wait a bit for event to be processed
		-- (In real scenario, events are processed synchronously)

		return #received_events
	`

	// Give time for async processing
	time.Sleep(100 * time.Millisecond)

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	result := L.Get(-1)
	if result.String() != "1" {
		t.Errorf("Expected 1 event received, got %s", result.String())
	}

	t.Log("✓ Publish/Subscribe test passed")
}

func TestEventBusAPI_MultipleSubscribers(t *testing.T) {
	manager := eventbus.NewManager(log.DefaultLogger)
	defer manager.Close()

	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	RegisterLogger(L, logger)
	RegisterEventBus(L, manager, logger)

	script := `
		local log = require("kratos_logger")
		local eventbus = require("kratos_eventbus")

		local count1 = 0
		local count2 = 0

		-- First subscriber
		eventbus.subscribe("test.event", function(event)
			log.info("Handler 1 received: " .. event.type)
			count1 = count1 + 1
		end)

		-- Second subscriber
		eventbus.subscribe("test.event", function(event)
			log.info("Handler 2 received: " .. event.type)
			count2 = count2 + 1
		end)

		-- Publish event
		eventbus.publish("test.event", {message = "hello"})

		return count1 + count2
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	result := L.Get(-1)
	if result.String() != "2" {
		t.Errorf("Expected 2 handlers to execute, got %s", result.String())
	}

	t.Log("✓ Multiple subscribers test passed")
}

func TestEventBusAPI_SubscribeOnce(t *testing.T) {
	manager := eventbus.NewManager(log.DefaultLogger)
	defer manager.Close()

	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	RegisterLogger(L, logger)
	RegisterEventBus(L, manager, logger)

	script := `
		local log = require("kratos_logger")
		local eventbus = require("kratos_eventbus")

		local call_count = 0

		-- Subscribe once
		eventbus.subscribe_once("once.event", function(event)
			log.info("Once handler called")
			call_count = call_count + 1
		end)

		-- Publish multiple times
		eventbus.publish("once.event", {msg = "first"})
		eventbus.publish("once.event", {msg = "second"})
		eventbus.publish("once.event", {msg = "third"})

		return call_count
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	result := L.Get(-1)
	if result.String() != "1" {
		t.Errorf("Expected handler to be called once, got %s times", result.String())
	}

	t.Log("✓ Subscribe once test passed")
}

func TestEventBusAPI_EventData(t *testing.T) {
	manager := eventbus.NewManager(log.DefaultLogger)
	defer manager.Close()

	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	RegisterLogger(L, logger)
	RegisterEventBus(L, manager, logger)

	script := `
		local log = require("kratos_logger")
		local eventbus = require("kratos_eventbus")

		local received_data = nil

		eventbus.subscribe("data.test", function(event)
			received_data = event.data
			log.info("Received user: " .. event.data.name)
		end)

		-- Publish with complex data
		eventbus.publish("data.test", {
			id = 42,
			name = "John Doe",
			email = "john@example.com",
			roles = {"admin", "user"}
		})

		if received_data == nil then
			error("No data received")
		end

		if received_data.name ~= "John Doe" then
			error("Expected name='John Doe', got: " .. tostring(received_data.name))
		end

		if received_data.id ~= 42 then
			error("Expected id=42, got: " .. tostring(received_data.id))
		end

		return true
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	t.Log("✓ Event data test passed")
}

func TestEventBusAPI_FullEventObject(t *testing.T) {
	manager := eventbus.NewManager(log.DefaultLogger)
	defer manager.Close()

	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	RegisterLogger(L, logger)
	RegisterEventBus(L, manager, logger)

	script := `
		local eventbus = require("kratos_eventbus")

		local received_event = nil

		eventbus.subscribe("full.event", function(event)
			received_event = event
		end)

		-- Publish full event object
		eventbus.publish({
			type = "full.event",
			data = {value = 123},
			source = "lua-test",
			priority = 5,
			metadata = {
				user_id = "user123",
				trace_id = "trace456"
			}
		})

		if received_event == nil then
			error("No event received")
		end

		if received_event.type ~= "full.event" then
			error("Wrong event type")
		end

		if received_event.source ~= "lua-test" then
			error("Wrong event source")
		end

		if received_event.priority ~= 5 then
			error("Wrong priority: " .. tostring(received_event.priority))
		end

		if received_event.metadata.user_id ~= "user123" then
			error("Wrong metadata")
		end

		return true
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	t.Log("✓ Full event object test passed")
}

func TestEventBusAPI_NamedBuses(t *testing.T) {
	manager := eventbus.NewManager(log.DefaultLogger)
	defer manager.Close()

	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	RegisterLogger(L, logger)
	RegisterEventBus(L, manager, logger)

	script := `
		local eventbus = require("kratos_eventbus")

		local email_count = 0
		local sms_count = 0

		-- Subscribe to email bus
		eventbus.subscribe("notification.sent", function(event)
			email_count = email_count + 1
		end, "email")

		-- Subscribe to SMS bus
		eventbus.subscribe("notification.sent", function(event)
			sms_count = sms_count + 1
		end, "sms")

		-- Publish to email bus
		eventbus.publish("notification.sent", {to = "user@example.com"}, "email")

		-- Publish to SMS bus
		eventbus.publish("notification.sent", {to = "+1234567890"}, "sms")

		-- Verify each bus received only its own events
		if email_count ~= 1 then
			error("Email bus should have received 1 event, got: " .. email_count)
		end

		if sms_count ~= 1 then
			error("SMS bus should have received 1 event, got: " .. sms_count)
		end

		return true
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	t.Log("✓ Named buses test passed")
}

func TestEventBusAPI_AsyncPublish(t *testing.T) {
	manager := eventbus.NewManager(log.DefaultLogger)
	defer manager.Close()

	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	RegisterLogger(L, logger)
	RegisterEventBus(L, manager, logger)

	var wg sync.WaitGroup
	wg.Add(1)

	// Subscribe from Go to verify async publish
	manager.Global().Subscribe("async.test", eventbus.EventHandlerFunc(func(ctx context.Context, event *eventbus.Event) error {
		defer wg.Done()
		logger.Info("Go handler received async event")
		return nil
	}))

	script := `
		local log = require("kratos_logger")
		local eventbus = require("kratos_eventbus")

		-- Publish async
		eventbus.publish_async("async.test", {message = "async"})
		log.info("Async publish returned immediately")
		return true
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	// Wait for async event
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("✓ Async publish test passed")
	case <-time.After(1 * time.Second):
		t.Error("Async event handler was not called")
	}
}

func TestEventBusAPI_CreateEvent(t *testing.T) {
	manager := eventbus.NewManager(log.DefaultLogger)
	defer manager.Close()

	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	RegisterLogger(L, logger)
	RegisterEventBus(L, manager, logger)

	script := `
		local eventbus = require("kratos_eventbus")

		-- Create event using helper
		local event = eventbus.create_event("test.create", {value = 42})

		-- Verify event structure
		if event.type ~= "test.create" then
			error("Wrong event type")
		end

		if event.data.value ~= 42 then
			error("Wrong data value")
		end

		if event.source ~= "lua" then
			error("Wrong source")
		end

		-- Modify event
		event.priority = 10
		event.metadata.custom = "value"

		-- Publish modified event
		eventbus.publish(event)

		return true
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	t.Log("✓ Create event helper test passed")
}

func TestEventBusAPI_CrossBusIsolation(t *testing.T) {
	manager := eventbus.NewManager(log.DefaultLogger)
	defer manager.Close()

	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	RegisterLogger(L, logger)
	RegisterEventBus(L, manager, logger)

	script := `
		local eventbus = require("kratos_eventbus")

		local global_count = 0
		local custom_count = 0

		-- Subscribe to global bus
		eventbus.subscribe("test.isolation", function(event)
			global_count = global_count + 1
		end, "global")

		-- Subscribe to custom bus
		eventbus.subscribe("test.isolation", function(event)
			custom_count = custom_count + 1
		end, "custom")

		-- Publish only to custom bus
		eventbus.publish("test.isolation", {}, "custom")

		-- Global should not receive the event
		if global_count ~= 0 then
			error("Global bus should not have received event, got: " .. global_count)
		end

		-- Custom should receive it
		if custom_count ~= 1 then
			error("Custom bus should have received 1 event, got: " .. custom_count)
		end

		return true
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	t.Log("✓ Cross-bus isolation test passed")
}

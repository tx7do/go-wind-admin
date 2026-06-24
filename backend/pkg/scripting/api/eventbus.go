package api

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	lua "github.com/yuin/gopher-lua"

	"go-wind-admin/pkg/eventbus"
	"go-wind-admin/pkg/scripting/internal/convert"
)

// LuaEventHandler wraps a Lua function as an eventbus.Handler
type LuaEventHandler struct {
	L        *lua.LState
	function *lua.LFunction
	logger   *log.Helper
	mu       sync.Mutex
}

// Handle executes the Lua function when an event is received
func (h *LuaEventHandler) Handle(ctx context.Context, event *eventbus.Event) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// ConvertCode event to Lua table
	eventTable := h.L.NewTable()
	eventTable.RawSetString("id", lua.LString(event.ID))
	eventTable.RawSetString("type", lua.LString(event.Type))
	eventTable.RawSetString("source", lua.LString(event.Source))
	eventTable.RawSetString("priority", lua.LNumber(event.Priority))
	eventTable.RawSetString("timestamp", lua.LNumber(event.Timestamp.Unix()))

	// ConvertCode event data
	if event.Data != nil {
		dataLua := convert.ToLuaValue(h.L, event.Data)
		eventTable.RawSetString("data", dataLua)
	}

	// ConvertCode metadata
	if event.Metadata != nil {
		metaTable := h.L.NewTable()
		for k, v := range event.Metadata {
			metaTable.RawSetString(k, lua.LString(v))
		}
		eventTable.RawSetString("metadata", metaTable)
	}

	// Call Lua function
	h.L.Push(h.function)
	h.L.Push(eventTable)

	if err := h.L.PCall(1, 0, nil); err != nil {
		h.logger.Errorf("Lua event handler error: %v", err)
		return fmt.Errorf("lua handler error: %w", err)
	}

	return nil
}

// RegisterEventBus registers the EventBus API for Lua
func RegisterEventBus(L *lua.LState, manager *eventbus.Manager, logger *log.Helper) {
	// Store manager in registry for later access (kept as internal global)
	L.SetGlobal("_eventbus_manager", lua.LString("internal"))

	// Register in package.preload so it can be required
	L.PreloadModule("kratos_eventbus", LoaderEventBus(manager, logger))
}

// LoaderEventBus 返回 eventbus 模块（kratos_eventbus）的 loader，供 go-scripts 引擎 RegisterModule 使用。
// manager 为 nil 时返回空模块。
func LoaderEventBus(manager *eventbus.Manager, logger *log.Helper) lua.LGFunction {
	return func(L *lua.LState) int {
		if manager == nil {
			L.Push(L.NewTable())
			return 1
		}
		// Create eventbus module
		eventbusModule := L.NewTable()

		// Helper to get or create bus
		getBus := func(L *lua.LState, busName string) eventbus.EventBus {
			if busName == "" || busName == "global" {
				return manager.Global()
			}
			return manager.GetBus(busName)
		}

		// eventbus.subscribe(event_type, handler_function, [bus_name])
		eventbusModule.RawSetString("subscribe", L.NewFunction(func(L *lua.LState) int {
			eventType := L.CheckString(1)
			handlerFunc := L.CheckFunction(2)
			busName := L.OptString(3, "global")

			bus := getBus(L, busName)

			// Create Lua handler wrapper
			handler := &LuaEventHandler{
				L:        L,
				function: handlerFunc,
				logger:   logger,
			}

			err := bus.Subscribe(eventType, handler)
			if err != nil {
				logger.Errorf("eventbus.subscribe error: %v", err)
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			logger.Debugf("Subscribed Lua handler for event type: %s (bus: %s)", eventType, busName)
			L.Push(lua.LBool(true))
			return 1
		}))

		// eventbus.subscribe_async(event_type, handler_function, [bus_name])
		eventbusModule.RawSetString("subscribe_async", L.NewFunction(func(L *lua.LState) int {
			eventType := L.CheckString(1)
			handlerFunc := L.CheckFunction(2)
			busName := L.OptString(3, "global")

			bus := getBus(L, busName)

			handler := &LuaEventHandler{
				L:        L,
				function: handlerFunc,
				logger:   logger,
			}

			err := bus.SubscribeAsync(eventType, handler)
			if err != nil {
				logger.Errorf("eventbus.subscribe_async error: %v", err)
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			logger.Debugf("Subscribed async Lua handler for event type: %s (bus: %s)", eventType, busName)
			L.Push(lua.LBool(true))
			return 1
		}))

		// eventbus.subscribe_once(event_type, handler_function, [bus_name])
		eventbusModule.RawSetString("subscribe_once", L.NewFunction(func(L *lua.LState) int {
			eventType := L.CheckString(1)
			handlerFunc := L.CheckFunction(2)
			busName := L.OptString(3, "global")

			bus := getBus(L, busName)

			handler := &LuaEventHandler{
				L:        L,
				function: handlerFunc,
				logger:   logger,
			}

			err := bus.SubscribeOnce(eventType, handler)
			if err != nil {
				logger.Errorf("eventbus.subscribe_once error: %v", err)
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			logger.Debugf("Subscribed once Lua handler for event type: %s (bus: %s)", eventType, busName)
			L.Push(lua.LBool(true))
			return 1
		}))

		// eventbus.publish(event_type, data, [bus_name])
		// eventbus.publish({type="...", data=..., source="...", priority=1, metadata={}}, [bus_name])
		eventbusModule.RawSetString("publish", L.NewFunction(func(L *lua.LState) int {
			firstArg := L.Get(1)
			busName := L.OptString(3, "global")

			var event *eventbus.Event

			// Check if first argument is a table (full event object)
			if table, ok := firstArg.(*lua.LTable); ok {
				// Extract event properties
				eventType := table.RawGetString("type")
				if eventType == lua.LNil {
					L.RaiseError("event.type is required")
					return 0
				}

				event = eventbus.NewEvent(eventType.String(), nil)

				// Extract data
				if dataVal := table.RawGetString("data"); dataVal != lua.LNil {
					event.Data = convert.ToGoValue(dataVal)
				}

				// Extract source
				if sourceVal := table.RawGetString("source"); sourceVal != lua.LNil {
					event.Source = sourceVal.String()
				}

				// Extract priority
				if priorityVal := table.RawGetString("priority"); priorityVal != lua.LNil {
					if num, ok := priorityVal.(lua.LNumber); ok {
						event.Priority = int(num)
					}
				}

				// Extract metadata
				if metaVal := table.RawGetString("metadata"); metaVal != lua.LNil {
					if metaTable, ok := metaVal.(*lua.LTable); ok {
						metaTable.ForEach(func(k, v lua.LValue) {
							if key, ok := k.(lua.LString); ok {
								event.WithMetadata(string(key), v.String())
							}
						})
					}
				}
			} else {
				// Simple form: publish(event_type, data, [bus_name])
				eventType := L.CheckString(1)
				dataVal := L.Get(2)

				event = eventbus.NewEvent(eventType, nil)
				if dataVal != lua.LNil {
					event.Data = convert.ToGoValue(dataVal)
				}
			}

			bus := getBus(L, busName)
			err := bus.Publish(context.Background(), event)
			if err != nil {
				logger.Errorf("eventbus.publish error: %v", err)
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			logger.Debugf("Published event: %s (bus: %s)", event.Type, busName)
			L.Push(lua.LBool(true))
			return 1
		}))

		// eventbus.publish_async(event_type, data, [bus_name])
		eventbusModule.RawSetString("publish_async", L.NewFunction(func(L *lua.LState) int {
			eventType := L.CheckString(1)
			dataVal := L.Get(2)
			busName := L.OptString(3, "global")

			event := eventbus.NewEvent(eventType, nil)
			if dataVal != lua.LNil {
				event.Data = convert.ToGoValue(dataVal)
			}

			bus := getBus(L, busName)
			err := bus.PublishAsync(context.Background(), event)
			if err != nil {
				logger.Errorf("eventbus.publish_async error: %v", err)
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			logger.Debugf("Published async event: %s (bus: %s)", event.Type, busName)
			L.Push(lua.LBool(true))
			return 1
		}))

		// eventbus.create_event(type, data) - Helper to create event object
		eventbusModule.RawSetString("create_event", L.NewFunction(func(L *lua.LState) int {
			eventType := L.CheckString(1)
			dataVal := L.Get(2)

			eventTable := L.NewTable()
			eventTable.RawSetString("type", lua.LString(eventType))

			if dataVal != lua.LNil {
				eventTable.RawSetString("data", dataVal)
			}

			eventTable.RawSetString("source", lua.LString("lua"))
			eventTable.RawSetString("priority", lua.LNumber(0))
			eventTable.RawSetString("metadata", L.NewTable())

			L.Push(eventTable)
			return 1
		}))

		L.Push(eventbusModule)
		return 1
	}
}

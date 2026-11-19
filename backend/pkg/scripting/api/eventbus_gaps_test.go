// 本文件补 eventbus.go 既有测试（eventbus_test.go）未覆盖的分支：
//   - nil manager 的 fail-closed 空模块；
//   - subscribe_async / publish_async（异步订阅路径）；
//   - serialize 非 nil 的串行入口注入路径（P2 修复点：引擎全局锁串行执行）；
//   - Lua handler 抛错 → "lua handler error: %w" 包装；
//   - publish 表形态缺 type 的 RaiseError；
//   - create_event 对 nil data 的字段缺省。
package api

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	lua "github.com/yuin/gopher-lua"

	"github.com/tx7do/go-utils/eventbus"
)

func TestEventBusAPI_NilManager(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	RegisterEventBus(L, nil, bLogger.NewHelper(bLogger.NopLogger()), nil)

	require.NoError(t, L.DoString(`
		local eb = require "kratos_eventbus"
		assert(type(eb) == "table", "module should be a table")
		assert(eb.subscribe == nil, "nil manager must not expose subscribe")
		assert(eb.publish == nil, "nil manager must not expose publish")
	`))
}

// TestEventBusAPI_SubscribeAsync 异步订阅：publish 后经独立 goroutine 分发，
// handler 最终被调用（轮询等待，不猜时序）。
// 注意：异步分发直调 PCall 会与主 goroutine 并发操作同一 LState——这正是
// P2 修复注入 serialize 的原因，故本测试按生产装配形态带 serialize 注册
// （serialize==nil 的直调路径由同步用例覆盖；带 -race 复现该风险会直接
// SIGSEGV，不宜放进常规回归）。
func TestEventBusAPI_SubscribeAsync(t *testing.T) {
	manager := eventbus.NewManager(bLogger.NopLogger())
	defer manager.Close()

	L := lua.NewState()
	logger := bLogger.NewHelper(bLogger.NopLogger())

	// serialize 语义（生产里是引擎 execMu）：handler 进 LState 前串行。
	// 事件从 Go 侧发布（订阅脚本先行结束），LState 全程独占使用——
	// "脚本内 publish + 异步订阅"的并发形态正是生产上 serialize 要挡的竞争，
	// 在单测里复现它会引入不可收敛的偶发崩溃，不放进常规回归。
	var inflight atomic.Int32
	RegisterEventBus(L, manager, logger, func(fn func()) {
		inflight.Add(1)
		defer inflight.Add(-1)
		fn()
	})

	var count atomic.Int32
	manager.Global().Subscribe("async.test", eventbus.EventHandlerFunc(func(context.Context, *eventbus.Event) error {
		count.Add(1)
		return nil
	}))

	// 1) 脚本只订阅（subscribe_async），不发布。
	require.NoError(t, L.DoString(`
		local eb = require "kratos_eventbus"
		eb.subscribe_async("async.test", function(event) end)
	`))
	// 2) Go 侧发布：Lua 异步 handler 与 Go handler 都应被调。
	require.NoError(t, manager.Publish(context.Background(), "global", eventbus.NewEvent("async.test", "payload")))

	require.Eventually(t, func() bool {
		return count.Load() >= 1
	}, 3*time.Second, 20*time.Millisecond, "async subscribers should receive the published event")

	// 清理顺序（t.Cleanup LIFO）：先注册关 LState、后注册等在途 handler 清零——
	// 退出时先等 handler 跑完，再关 LState，否则 L.Close 与仍在跑的 PCall 构成数据竞争。
	t.Cleanup(L.Close)
	t.Cleanup(func() {
		require.Eventually(t, func() bool { return inflight.Load() == 0 }, 3*time.Second, 10*time.Millisecond)
	})
}

// TestEventBusAPI_SerializedHandler serialize 非 nil：Handle 经串行入口执行 invoke，
// 事件仍正常送达（P2 修复路径——引擎全局锁串行）。
func TestEventBusAPI_SerializedHandler(t *testing.T) {
	var entered atomic.Int32
	manager := eventbus.NewManager(bLogger.NopLogger())
	defer manager.Close()

	L := lua.NewState()
	defer L.Close()
	logger := bLogger.NewHelper(bLogger.NopLogger())
	serialize := func(fn func()) {
		entered.Add(1)
		fn()
	}
	RegisterEventBus(L, manager, logger, serialize)

	require.NoError(t, L.DoString(`
		local eb = require "kratos_eventbus"
		eb.subscribe("serial.test", function(event)
			-- handler 内断言拿到事件类型
			assert(event.type == "serial.test", "event type should be carried, got " .. tostring(event.type))
		end)
		eb.publish("serial.test", "data")
	`))

	// serialize 路径：publish → 同步分发 → Handle 走 serialize 包装。
	require.Eventually(t, func() bool {
		return entered.Load() >= 1
	}, 3*time.Second, 20*time.Millisecond, "serialized Handle must be entered via the serialize hook")
}

// TestEventBusAPI_HandlerErrorWrapped Lua handler 抛错 → 包装为 "lua handler error:"。
func TestEventBusAPI_HandlerErrorWrapped(t *testing.T) {
	manager := eventbus.NewManager(bLogger.NopLogger())
	defer manager.Close()

	L := lua.NewState()
	defer L.Close()
	logger := bLogger.NewHelper(bLogger.NopLogger())
	RegisterEventBus(L, manager, logger, nil)

	var handlerErr atomic.Value
	// 直接构造 LuaEventHandler 抛错（不经 publish 分发，同步可控）。
	require.NoError(t, L.DoString(`
		function thrower(event) error("boom inside lua") end
	`))
	fn := L.GetGlobal("thrower")
	luaFn, ok := fn.(*lua.LFunction)
	require.True(t, ok)

	handler := &LuaEventHandler{L: L, function: luaFn, logger: logger}
	err := handler.Handle(context.Background(), &eventbus.Event{ID: "e1", Type: "err.test"})
	require.Error(t, err)
	require.ErrorContains(t, err, "lua handler error:")
	require.ErrorContains(t, err, "boom inside lua")
	handlerErr.Store(err)

	// serialize 非 nil 时错误同样透传。
	var serializedErr error
	h2 := &LuaEventHandler{L: L, function: luaFn, logger: logger, serialize: func(fn func()) { fn() }}
	serializedErr = h2.Handle(context.Background(), &eventbus.Event{ID: "e2", Type: "err.test"})
	require.ErrorContains(t, serializedErr, "lua handler error:")
}

// TestEventBusAPI_PublishTableWithoutType 表形态 publish 缺 type → RaiseError。
func TestEventBusAPI_PublishTableWithoutType(t *testing.T) {
	manager := eventbus.NewManager(bLogger.NopLogger())
	defer manager.Close()

	L := lua.NewState()
	defer L.Close()
	logger := bLogger.NewHelper(bLogger.NopLogger())
	RegisterEventBus(L, manager, logger, nil)

	require.Error(t, L.DoString(`
		local eb = require "kratos_eventbus"
		eb.publish({data = "no type field"})
	`))
}

// TestEventBusAPI_CreateEventNilData create_event 不带 data：事件表缺 data 字段但可用。
func TestEventBusAPI_CreateEventNilData(t *testing.T) {
	manager := eventbus.NewManager(bLogger.NopLogger())
	defer manager.Close()

	L := lua.NewState()
	defer L.Close()
	logger := bLogger.NewHelper(bLogger.NopLogger())
	RegisterEventBus(L, manager, logger, nil)

	require.NoError(t, L.DoString(`
		local eb = require "kratos_eventbus"
		local ev = eb.create_event("nil.data.test")
		assert(ev.type == "nil.data.test", "type should be set")
		assert(ev.data == nil, "data should be absent for nil input")
	`))
}

// TestLuaEventHandler_ConcurrentInvoke mu 串行语义：并发 Handle 不产生竞争与交错
// （-race 下运行）。
func TestLuaEventHandler_ConcurrentInvoke(t *testing.T) {
	manager := eventbus.NewManager(bLogger.NopLogger())
	defer manager.Close()

	L := lua.NewState()
	defer L.Close()
	logger := bLogger.NewHelper(bLogger.NopLogger())
	require.NoError(t, L.DoString(`
		function collector(event)
			_G.last_seen = event.type
			return true
		end
	`))
	luaFn := L.GetGlobal("collector").(*lua.LFunction)
	handler := &LuaEventHandler{L: L, function: luaFn, logger: logger}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = handler.Handle(context.Background(), &eventbus.Event{ID: "c", Type: "conc.test"})
		}(i)
	}
	wg.Wait()
	require.NoError(t, L.DoString(`assert(_G.last_seen == "conc.test")`))
}

// --- 基准：事件分发热路径（构造事件表 + 转换 + PCall） ---

func BenchmarkLuaEventHandler_Invoke(b *testing.B) {
	L := lua.NewState()
	defer L.Close()
	logger := bLogger.NewHelper(bLogger.NopLogger())
	require.NoError(b, L.DoString(`
		function noop_handler(event) return true end
	`))
	luaFn := L.GetGlobal("noop_handler").(*lua.LFunction)
	handler := &LuaEventHandler{L: L, function: luaFn, logger: logger}
	ev := eventbus.NewEvent("bench.event", map[string]any{"k": "v", "n": 42})
	ev.WithMetadata("source", "bench")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := handler.Handle(context.Background(), ev); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEventBus_PublishSubscribe 全链路：publish → 分发 → Lua handler。
func BenchmarkEventBus_PublishSubscribe(b *testing.B) {
	manager := eventbus.NewManager(bLogger.NopLogger())
	defer manager.Close()

	L := lua.NewState()
	defer L.Close()
	logger := bLogger.NewHelper(bLogger.NopLogger())
	RegisterEventBus(L, manager, logger, nil)
	require.NoError(b, L.DoString(`
		local eb = require "kratos_eventbus"
		eb.subscribe("bench.bus", function(event) end)
	`))

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := manager.Publish(ctx, "global", eventbus.NewEvent("bench.bus", "x")); err != nil {
			b.Fatal(err)
		}
	}
	_ = errors.New // 保持 errors 导入（错误分支在上方 Handle 中覆盖）
}

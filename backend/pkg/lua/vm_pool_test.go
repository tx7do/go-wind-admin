package lua

import (
	"sync"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// makeTestFactory 返回一个 VM 工厂，创建最小可用的沙箱 VM。
func makeTestFactory() func() *lua.LState {
	return func() *lua.LState {
		L := lua.NewState(lua.Options{SkipOpenLibs: true})
		lua.OpenBase(L)
		return L
	}
}

func TestVMPool_GetPut(t *testing.T) {
	pool := newVMPool(3, makeTestFactory())
	defer pool.Close()

	vm := pool.Get()
	if vm == nil {
		t.Fatal("Get returned nil VM")
	}
	// 归还后应能再次获取
	pool.Put(vm)
	vm2 := pool.Get()
	if vm2 == nil {
		t.Fatal("second Get returned nil VM")
	}
	pool.Put(vm2)
}

func TestVMPool_GetWhenEmpty(t *testing.T) {
	pool := newVMPool(1, makeTestFactory())
	defer pool.Close()

	// 取出唯一的 VM，池空
	vm1 := pool.Get()
	// 再次 Get 应新建 VM（不阻塞）
	vm2 := pool.Get()
	if vm2 == nil {
		t.Fatal("Get on empty pool returned nil (should create new)")
	}

	pool.Put(vm1)
	pool.Put(vm2) // 池满，vm2 应被关闭
}

func TestVMPool_IsClosed(t *testing.T) {
	pool := newVMPool(2, makeTestFactory())

	if pool.IsClosed() {
		t.Fatal("new pool should not be closed")
	}

	pool.Close()

	if !pool.IsClosed() {
		t.Fatal("pool should be closed after Close")
	}
}

func TestVMPool_CloseIdempotent(t *testing.T) {
	pool := newVMPool(2, makeTestFactory())

	// 多次 Close 不应 panic
	if err := recover(); err != nil {
		t.Fatalf("unexpected panic before Close: %v", err)
	}
	pool.Close()
	pool.Close()
	pool.Close()
}

func TestVMPool_PutAfterClose(t *testing.T) {
	pool := newVMPool(2, makeTestFactory())

	vm := pool.Get()
	pool.Close()

	// 池关闭后 Put 应安全关闭 VM，不 panic
	pool.Put(vm)
}

func TestVMPool_PutNil(t *testing.T) {
	pool := newVMPool(2, makeTestFactory())
	defer pool.Close()

	// Put nil 应为 no-op，不 panic
	pool.Put(nil)
}

func TestVMPool_Replace(t *testing.T) {
	pool := newVMPool(2, makeTestFactory())
	defer pool.Close()

	old := pool.Get()
	newVM := makeTestFactory()()

	// Replace 应关闭旧 VM，归还新 VM
	pool.Replace(old, newVM)

	// 从池中取出的应该是有效 VM
	got := pool.Get()
	if got == nil {
		t.Fatal("Get after Replace returned nil")
	}
}

func TestVMPool_ReplaceNilOld(t *testing.T) {
	pool := newVMPool(2, makeTestFactory())
	defer pool.Close()

	newVM := makeTestFactory()()
	// old 为 nil 不应 panic
	pool.Replace(nil, newVM)
}

func TestVMPool_ConcurrentAccess(t *testing.T) {
	pool := newVMPool(4, makeTestFactory())
	defer pool.Close()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vm := pool.Get()
			// 模拟短暂使用
			vm.SetTop(0)
			pool.Put(vm)
		}()
	}
	wg.Wait()
}

package lua

import (
	"sync"

	lua "github.com/yuin/gopher-lua"
)

// vmPool manages a pool of Lua VMs for reuse.
//
// 借鉴 go-scripts 的 EnginePool（engine_pool.go）健壮性设计：
//   - Get() 在池关闭时返回新 VM，但通过 IsClosed() 允许调用方感知
//   - Put() 增加 recover 防御 send-on-closed panic
//   - Close() 幂等，重复调用安全
//   - Replace() 安全替换因超时/异常而损坏的 VM
type vmPool struct {
	vms     chan *lua.LState
	factory func() *lua.LState
	size    int
	closed  bool
	mu      sync.Mutex
}

// newVMPool 创建一个固定大小的 VM 池，并预创建所有 VM。
func newVMPool(size int, factory func() *lua.LState) *vmPool {
	pool := &vmPool{
		vms:     make(chan *lua.LState, size),
		factory: factory,
		size:    size,
	}

	// 预创建 VM
	for i := 0; i < size; i++ {
		pool.vms <- factory()
	}

	return pool
}

// TryGet 非阻塞地从池中获取一个 VM。
// 池空或已关闭时返回 nil（不新建 VM），用于遍历池内现存 VM。
func (p *vmPool) TryGet() *lua.LState {
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()

	if closed {
		return nil
	}

	select {
	case vm := <-p.vms:
		return vm
	default:
		return nil
	}
}

// Get 从池中获取一个 VM。若池已空则新建；若池已关闭仍返回新建 VM，
// 调用方可通过 IsClosed() 感知关闭态以决定后续行为。
func (p *vmPool) Get() *lua.LState {
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()

	if closed {
		// 池已关闭：返回新 VM（调用方负责关闭，不应再 Put 回来）
		return p.factory()
	}

	select {
	case vm := <-p.vms:
		return vm
	default:
		// 池空，创建新 VM
		return p.factory()
	}
}

// Put 将 VM 归还池中。若池已关闭则直接关闭该 VM。
// 使用 recover 防御并发 Close 导致的 send-on-closed panic。
func (p *vmPool) Put(vm *lua.LState) {
	if vm == nil {
		return
	}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		vm.Close()
		return
	}
	p.mu.Unlock()

	// 重置 VM 状态
	vm.SetTop(0)

	// 防御并发 Close 期间向 chan 发送引发的 panic
	defer func() {
		if r := recover(); r != nil {
			// 池在发送期间被关闭，直接回收 VM
			vm.Close()
		}
	}()

	select {
	case p.vms <- vm:
		// 成功归还
	default:
		// 池满，关闭多余 VM
		vm.Close()
	}
}

// Replace 用新 VM 替换旧 VM。
// 旧 VM 可能因超时/异常而状态损坏（如 Execute 超时后 VM 已不可用），
// 此处安全关闭旧 VM 并将新 VM 归还池。
func (p *vmPool) Replace(old, newVM *lua.LState) {
	// 安全关闭旧 VM（可能已经 Close 过，recover 防御二次关闭 panic）
	if old != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// 旧 VM 已关闭或状态异常，忽略
				}
			}()
			old.Close()
		}()
	}
	p.Put(newVM)
}

// IsClosed 报告池是否已关闭。
func (p *vmPool) IsClosed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closed
}

// Close 关闭池及所有池内 VM。幂等，重复调用安全。
func (p *vmPool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	close(p.vms)
	p.mu.Unlock()

	// 排空并关闭所有 VM
	for vm := range p.vms {
		if vm != nil {
			vm.Close()
		}
	}
}

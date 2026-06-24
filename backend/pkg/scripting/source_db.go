package scripting

import (
	"context"
	"fmt"
	"sync"
	"time"

	gsSource "github.com/tx7do/go-scripts/source"
)

// ScriptLoader 是从数据库加载脚本源码的回调函数类型。
//
// 因 pkg/scripting 不能反向依赖 internal/data（会形成循环依赖），
// 由 app 层在 Wire 装配时注入真正的 DB 查询函数。
//
// 参数 key 通常是脚本的 id 或 name，返回脚本源码字符串。
type ScriptLoader func(ctx context.Context, key string) (string, error)

// ScriptHasher 计算 key 对应脚本的版本指纹，用于检测变更以触发热更新。
// 若返回空字符串，则回退到对源码字符串本身的比对。
type ScriptHasher func(ctx context.Context, key string) (string, error)

// DBSource 从数据库加载脚本，实现 source.ReadWatcher 接口。
//
// 典型场景：脚本存储在数据库表中，通过管理后台 UI 增删改，
// DBSource 负责按 key 加载并提供基于轮询的热更新能力。
//
// 通过注入 ScriptLoader/ScriptHasher 回调解耦 data 层，避免循环依赖。
type DBSource struct {
	loader ScriptLoader // 脚本加载回调（必填）
	hasher ScriptHasher // 脚本指纹回调（可选，用于高效变更检测）

	cache        map[string]string // key -> 源码缓存（避免每次 Load 都查库）
	cacheMu      sync.RWMutex
	pollInterval time.Duration // 轮询间隔（默认 5s）
}

// DBOption 配置 DBSource。
type DBOption func(*DBSource)

// WithScriptHasher 设置脚本指纹回调，用于热更新时的高效变更检测。
func WithScriptHasher(h ScriptHasher) DBOption {
	return func(d *DBSource) { d.hasher = h }
}

// WithPollInterval 设置热更新轮询间隔，默认 5 秒。
func WithPollInterval(d time.Duration) DBOption {
	return func(s *DBSource) { s.pollInterval = d }
}

// NewDBSource 创建数据库脚本源。
// loader 为脚本加载回调（必填，nil 将 panic）。
func NewDBSource(loader ScriptLoader, opts ...DBOption) *DBSource {
	if loader == nil {
		panic("DBSource: loader must not be nil")
	}
	s := &DBSource{
		loader:       loader,
		cache:        make(map[string]string),
		pollInterval: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Compile-time assertion: *DBSource 实现 source.ReadWatcher。
var _ gsSource.ReadWatcher = (*DBSource)(nil)

// Load 按 key 从数据库加载脚本源码。带内存缓存，避免重复查库。
func (d *DBSource) Load(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	// 缓存命中
	d.cacheMu.RLock()
	if code, ok := d.cache[key]; ok {
		d.cacheMu.RUnlock()
		return code, nil
	}
	d.cacheMu.RUnlock()

	// 回源加载
	code, err := d.loader(ctx, key)
	if err != nil {
		return "", fmt.Errorf("db source: load %q: %w", key, err)
	}

	d.cacheMu.Lock()
	d.cache[key] = code
	d.cacheMu.Unlock()

	return code, nil
}

// Close 释放资源（当前无外部资源）。
func (d *DBSource) Close() error {
	d.cacheMu.Lock()
	d.cache = make(map[string]string)
	d.cacheMu.Unlock()
	return nil
}

// Invalidate 使指定 key 的缓存失效，下次 Load 将强制回源。
func (d *DBSource) Invalidate(key string) {
	d.cacheMu.Lock()
	delete(d.cache, key)
	d.cacheMu.Unlock()
}

// InvalidateAll 清空所有缓存。
func (d *DBSource) InvalidateAll() {
	d.cacheMu.Lock()
	d.cache = make(map[string]string)
	d.cacheMu.Unlock()
}

// Watch 基于轮询的变更检测。
// 每 pollInterval 检查一次：优先用 hasher 指纹比对，否则比对源码字符串。
// 检测到变更时通过 channel 发送信号，ctx 取消时关闭 channel。
func (d *DBSource) Watch(ctx context.Context, key string) (<-chan struct{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ch := make(chan struct{}, 1)

	// 记录初始指纹
	lastSig := d.signature(ctx, key)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer close(ch)

		ticker := time.NewTicker(d.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				curSig := d.signature(ctx, key)
				if curSig != lastSig {
					lastSig = curSig
					// 变更发生，使缓存失效以便下次 Load 取新值
					d.Invalidate(key)
					select {
					case ch <- struct{}{}:
					default:
					}
				}
			}
		}
	}()

	return ch, nil
}

// signature 计算 key 对应脚本的变更指纹。
// 优先用 hasher（高效，如返回 updated_at 或版本号），否则返回源码字符串本身。
func (d *DBSource) signature(ctx context.Context, key string) string {
	if d.hasher != nil {
		if sig, err := d.hasher(ctx, key); err == nil {
			return sig
		}
	}
	// 回退：比对源码
	code, err := d.loader(ctx, key)
	if err != nil {
		return "" // 加载失败，指纹不变，避免误触发
	}
	return code
}

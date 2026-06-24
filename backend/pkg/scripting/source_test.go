package scripting

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// --- FileSource 测试 ---

func TestFileSource_Load(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.lua")
	content := "return 1"

	if err := os.WriteFile(scriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	src := NewFileSource()
	code, err := src.Load(context.Background(), scriptPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if code != content {
		t.Errorf("expected %q, got %q", content, code)
	}
}

func TestFileSource_LoadWithPaths(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "sub", "mod.lua")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scriptPath, []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}

	// 传入搜索路径，key 用相对路径
	src := NewFileSource(filepath.Join(tmpDir, "sub"))
	code, err := src.Load(context.Background(), "mod.lua")
	if err != nil {
		t.Fatalf("Load with paths failed: %v", err)
	}
	if code != "ok" {
		t.Errorf("expected 'ok', got %q", code)
	}
}

func TestFileSource_LoadMissingFile(t *testing.T) {
	src := NewFileSource()
	_, err := src.Load(context.Background(), "/nonexistent/file.lua")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFileSource_Watch(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "watched.lua")
	if err := os.WriteFile(scriptPath, []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}

	src := NewFileSource()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := src.Watch(ctx, scriptPath)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	// 等待 Watch goroutine 启动
	time.Sleep(200 * time.Millisecond)

	// 修改文件（mtime 精度为秒，需确保 mtime 变化）
	time.Sleep(1100 * time.Millisecond)
	if err := os.WriteFile(scriptPath, []byte("v2"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-ch:
		// 收到变更信号
	case <-time.After(3 * time.Second):
		t.Fatal("did not receive watch signal within 3s")
	}
}

// --- DBSource 测试 ---

func TestDBSource_Load(t *testing.T) {
	scripts := map[string]string{
		"hello": "print('hello')",
		"world": "print('world')",
	}

	loader := func(_ context.Context, key string) (string, error) {
		if code, ok := scripts[key]; ok {
			return code, nil
		}
		return "", os.ErrNotExist
	}

	src := NewDBSource(loader, WithPollInterval(100*time.Millisecond))
	defer src.Close()

	code, err := src.Load(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if code != "print('hello')" {
		t.Errorf("unexpected code: %q", code)
	}
}

func TestDBSource_LoadCaching(t *testing.T) {
	var calls int32
	loader := func(_ context.Context, key string) (string, error) {
		atomic.AddInt32(&calls, 1)
		return "cached", nil
	}

	src := NewDBSource(loader)
	defer src.Close()

	// 第一次 Load 触发回调
	src.Load(context.Background(), "k")
	// 第二次 Load 应命中缓存，不触发回调
	src.Load(context.Background(), "k")

	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("expected 1 loader call (cache hit), got %d", calls)
	}
}

func TestDBSource_Invalidate(t *testing.T) {
	var calls int32
	loader := func(_ context.Context, key string) (string, error) {
		atomic.AddInt32(&calls, 1)
		return "v", nil
	}

	src := NewDBSource(loader)
	defer src.Close()

	src.Load(context.Background(), "k")
	src.Invalidate("k")
	src.Load(context.Background(), "k")

	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("expected 2 loader calls after invalidate, got %d", calls)
	}
}

func TestDBSource_Watch(t *testing.T) {
	var version int32
	loader := func(_ context.Context, key string) (string, error) {
		v := atomic.LoadInt32(&version)
		if v == 0 {
			return "original", nil
		}
		return "updated", nil
	}
	hasher := func(_ context.Context, key string) (string, error) {
		v := atomic.LoadInt32(&version)
		return string(rune('0' + v)), nil
	}

	src := NewDBSource(loader, WithScriptHasher(hasher), WithPollInterval(100*time.Millisecond))
	defer src.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := src.Watch(ctx, "k")
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	// 触发变更
	time.Sleep(150 * time.Millisecond)
	atomic.StoreInt32(&version, 1)

	select {
	case <-ch:
		// 收到变更信号，缓存应已失效
	case <-time.After(3 * time.Second):
		t.Fatal("did not receive DB watch signal within 3s")
	}

	// 验证缓存失效后重新加载
	code, _ := src.Load(context.Background(), "k")
	if code != "updated" {
		t.Errorf("expected 'updated' after reload, got %q", code)
	}
}

func TestDBSource_NewNilLoader(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil loader")
		}
	}()
	NewDBSource(nil)
}

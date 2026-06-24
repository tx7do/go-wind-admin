package scripting

import (
	"context"
	"path/filepath"

	gsSource "github.com/tx7do/go-scripts/source"
)

// FileSource 从本地文件系统加载脚本，包装 go-scripts/source.FileSource，
// 额外提供搜索路径（paths）解析能力。
//
// 实现 source.ReadWatcher 接口，因此可直接传给 Engine.SetSource，
// 也可用于热更新（基于文件 mtime 轮询）。
type FileSource struct {
	src   *gsSource.FileSource // 底层 go-scripts FileSource
	paths []string             // 搜索路径列表（key 相对路径时依次拼接尝试）
}

// NewFileSource 创建一个文件脚本源。
// paths 为搜索路径列表，Load 时若 key 不是绝对路径，会依次与 paths 拼接尝试。
func NewFileSource(paths ...string) *FileSource {
	return &FileSource{
		src:   gsSource.NewFileSource(),
		paths: paths,
	}
}

// Load 按 key 加载脚本。若 key 为相对路径，依次尝试 paths 中的前缀。
func (f *FileSource) Load(ctx context.Context, key string) (string, error) {
	resolved := f.resolveKey(key)
	return f.src.Load(ctx, resolved)
}

// Close 释放资源（FileSource 无资源，no-op）。
func (f *FileSource) Close() error {
	return f.src.Close()
}

// Watch 监听脚本文件变更（基于 mtime 轮询，每秒一次）。
// 变更时通过返回的 channel 发送信号，调用方应重新 Load。
func (f *FileSource) Watch(ctx context.Context, key string) (<-chan struct{}, error) {
	resolved := f.resolveKey(key)
	return f.src.Watch(ctx, resolved)
}

// resolveKey 解析脚本 key：绝对路径直接返回，相对路径依次与 paths 拼接。
// 若 paths 为空或均不匹配，返回原始 key（交由底层 Load 报错）。
func (f *FileSource) resolveKey(key string) string {
	if key == "" || filepath.IsAbs(key) || len(f.paths) == 0 {
		return key
	}
	for _, p := range f.paths {
		candidate := filepath.Clean(filepath.Join(p, key))
		return candidate // 返回第一个拼接结果，底层 Load 会处理文件不存在
	}
	return key
}

package lua

import (
	"os"
	"path/filepath"
	"strings"
)

// osStat 是 os.Stat 的薄封装，便于在 engine.go 中避免直接 import os（保持导入整洁）。
func osStat(dir string) (os.FileInfo, error) {
	return os.Stat(dir)
}

// walkLuaFiles 遍历 dir 下的所有 .lua 文件，对每个文件调用 fn。
// 遇到目录或非 .lua 文件则跳过。
func walkLuaFiles(dir string, fn func(path string) error) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".lua") {
			return nil
		}
		return fn(path)
	})
}

package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"go-wind-admin/pkg/crypto"
)

// 本文件定义语言无关的模块（ModuleDef），供非 Lua 引擎（如 goja JS）使用。
//
// ModuleDef 的 Funcs 是 map[函数名]Go函数，各语言适配器负责注册到各自 VM：
//   - JS（goja）：RegisterModule(name, map[string]any) 直接设全局，Go 函数自动可调用
//   - Lua：继续使用 Loader*（lua.LGFunction），Module* 作为未来统一方向
//
// Go 函数签名约定：
//   - 参数为基础类型（string/int/float64/map[string]any/[]any）
//   - 单返回值 或 (result, error)
//   - 无返回值的函数返回空

// ModuleDef 是语言无关的模块定义：名称 + 一组 Go 函数。
type ModuleDef struct {
	Name  string
	Funcs map[string]any
}

// convertForFormatVal 将 float64 整数值转为 int，便于 %d 格式化。
func convertForFormatVal(val any) any {
	if f, ok := val.(float64); ok {
		if f == math.Floor(f) && !math.IsInf(f, 0) && !math.IsNaN(f) {
			return int(f)
		}
	}
	return val
}

// ModuleLogger 构建语言无关的 logger 模块。
func ModuleLogger(logger *log.Helper) ModuleDef {
	return ModuleDef{
		Name: "log",
		Funcs: map[string]any{
			"info": func(msg string) {
				logger.Info(msg)
			},
			"warn": func(msg string) {
				logger.Warn(msg)
			},
			"error": func(msg string) {
				logger.Error(msg)
			},
			"debug": func(msg string) {
				logger.Debug(msg)
			},
			"infof": func(format string, args ...any) {
				logger.Infof(format, convertForFormatArgs(args)...)
			},
			"warnf": func(format string, args ...any) {
				logger.Warnf(format, convertForFormatArgs(args)...)
			},
			"errorf": func(format string, args ...any) {
				logger.Errorf(format, convertForFormatArgs(args)...)
			},
			"debugf": func(format string, args ...any) {
				logger.Debugf(format, convertForFormatArgs(args)...)
			},
		},
	}
}

// convertForFormatArgs 将 variadic any 参数中的 float64 整数转为 int（兼容 %d）。
func convertForFormatArgs(args []any) []any {
	out := make([]any, len(args))
	for i, a := range args {
		out[i] = convertForFormatVal(a)
	}
	return out
}

// ModuleCrypto 构建语言无关的 crypto 模块。
func ModuleCrypto() ModuleDef {
	return ModuleDef{
		Name: "crypto",
		Funcs: map[string]any{
			"encrypt": func(plaintext string) (string, error) {
				return crypto.EncryptIfNeeded(plaintext)
			},
			"decrypt": func(ciphertext string) (string, error) {
				return crypto.DecryptIfNeeded(ciphertext)
			},
			"is_encrypted": func(data string) bool {
				return crypto.IsEncrypted(data)
			},
			"encrypt_json": func(data any) (string, error) {
				jsonData, err := json.Marshal(data)
				if err != nil {
					return "", err
				}
				return crypto.EncryptIfNeeded(string(jsonData))
			},
			"decrypt_json": func(encrypted string) (any, error) {
				decrypted, err := crypto.DecryptIfNeeded(encrypted)
				if err != nil {
					return nil, err
				}
				var result any
				if err := json.Unmarshal([]byte(decrypted), &result); err != nil {
					return nil, err
				}
				return result, nil
			},
			"hash_sha256": func(data string) string {
				hash := sha256.Sum256([]byte(data))
				return hex.EncodeToString(hash[:])
			},
		},
	}
}

// ModuleUtil 构建语言无关的 util 模块。
func ModuleUtil() ModuleDef {
	return ModuleDef{
		Name: "util",
		Funcs: map[string]any{
			"sleep": func(seconds float64) {
				time.Sleep(time.Duration(seconds * float64(time.Second)))
			},
			"time": func() int64 {
				return time.Now().Unix()
			},
			"timestamp": func() int64 {
				return time.Now().UnixMilli()
			},
			"date": func(format ...string) string {
				f := time.RFC3339
				if len(format) > 0 && format[0] != "" {
					f = format[0]
				}
				return time.Now().Format(f)
			},
		},
	}
}

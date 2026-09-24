package api

// 本文件针对 cache.go（kratos_cache 模块）做单元测试，覆盖 RegisterCache / LoaderCache 全部分支：
//   - 字符串与 JSON（对象/数组/嵌套）的 set/get 往返；
//   - get/hget 命中 redis.Nil（键或字段不存在）返回 nil；
//   - incr/decr/incrby、exists/delete、set 带 TTL 与不带 TTL；
//   - ttl/expire（含无过期键 -1、不存在键 -2）；
//   - keys 模式匹配；
//   - hset/hget/hgetall；
//   - miniredis.SetError 全局错误注入覆盖全部错误返回分支；
//   - set 对含 NaN 的表做 JSON 序列化失败的分支；
//   - rdb 为 nil 时 loader 返回空模块。
//
// Redis 侧使用 miniredis 内存服务器，无需外部 Redis。

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	lua "github.com/yuin/gopher-lua"
)

// newCacheLuaState 构造绑定 miniredis 的、注册了 kratos_cache 模块的 LState。
// 返回的清理函数关闭 Redis 客户端（miniredis 服务器由 RunT 自动回收）。
func newCacheLuaState(t *testing.T) (*lua.LState, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	L := lua.NewState()
	t.Cleanup(L.Close)
	RegisterCache(L, rdb, bLogger.NewHelper(bLogger.NopLogger()))
	return L, mr
}

// TestCacheAPI_SetGetString 字符串 set/get 往返（带 TTL 与不带 TTL 两个 set 分支），
// 非键读取返回 nil（redis.Nil 分支）。
func TestCacheAPI_SetGetString(t *testing.T) {
	L, _ := newCacheLuaState(t)

	err := L.DoString(`
		local cache = require "kratos_cache"

		-- 带 TTL 的 set
		assert(cache.set("plain", "hello-value", 60) == true, "set with ttl should succeed")
		local v = cache.get("plain")
		assert(v == "hello-value", "get should round-trip the string, got " .. tostring(v))

		-- 不带 TTL 的 set（默认不过期）
		assert(cache.set("plain2", "x") == true, "set without ttl should succeed")
		local v2 = cache.get("plain2")
		assert(v2 == "x", "get should round-trip, got " .. tostring(v2))

		-- nil 值：按空字符串存储（ToGoValue 对 LNil 返回 nil）
		assert(cache.set("nilval", nil, 60) == true, "set nil should succeed")
		local nv = cache.get("nilval")
		assert(nv == "", "nil value should store empty string, got " .. tostring(nv))

		-- 不存在的键 → nil
		local miss = cache.get("does-not-exist")
		assert(miss == nil, "missing key should return nil")
	`)
	require.NoError(t, err)
}

// TestCacheAPI_JSONRoundTrip 表经 JSON 序列化存储，读取时反序列化为表；
// 直接存原始 JSON 字符串同样在 get 侧走 JSON 解码分支。
func TestCacheAPI_JSONRoundTrip(t *testing.T) {
	L, _ := newCacheLuaState(t)

	err := L.DoString(`
		local cache = require "kratos_cache"

		-- 表 → JSON → 表（含数组与嵌套）
		local obj = {name = "Alice", tags = {"a", "b"}, nested = {x = 1}}
		assert(cache.set("jsonobj", obj, 60) == true, "set table should succeed")
		local back = cache.get("jsonobj")
		assert(type(back) == "table", "round-trip should yield table, got " .. type(back))
		assert(back.name == "Alice", "name field mismatch")
		assert(#back.tags == 2, "array length mismatch")
		assert(back.tags[1] == "a" and back.tags[2] == "b", "array contents mismatch")
		assert(back.nested.x == 1, "nested field mismatch")

		-- 原始 JSON 字符串：get 侧走 JSON 解码分支
		assert(cache.set("rawjson", '{"n":5}', 60) == true)
		local dec = cache.get("rawjson")
		assert(type(dec) == "table", "raw json should decode to table")
		assert(dec.n == 5, "decoded field mismatch")
	`)
	require.NoError(t, err)
}

// TestCacheAPI_SetTableWithNaN 含 NaN 的表 JSON 序列化失败：
// 返回 (false, 错误字符串)。
func TestCacheAPI_SetTableWithNaN(t *testing.T) {
	L, _ := newCacheLuaState(t)

	err := L.DoString(`
		local cache = require "kratos_cache"
		local ok, e = cache.set("nan", {x = 0/0})
		assert(ok == false, "set with NaN table should fail")
		assert(type(e) == "string", "marshal error should surface as string, got " .. type(e))
	`)
	require.NoError(t, err)
}

// TestCacheAPI_IncrDecrIncrBy 计数器家族返回值。
func TestCacheAPI_IncrDecrIncrBy(t *testing.T) {
	L, _ := newCacheLuaState(t)

	err := L.DoString(`
		local cache = require "kratos_cache"
		assert(cache.incr("c") == 1, "first incr should be 1")
		assert(cache.incr("c") == 2, "second incr should be 2")
		assert(cache.decr("c") == 1, "decr should be 1")
		assert(cache.incrby("c", 10) == 11, "incrby 10 should be 11")
	`)
	require.NoError(t, err)
}

// TestCacheAPI_ExistsDelete 存在性检查与删除。
func TestCacheAPI_ExistsDelete(t *testing.T) {
	L, _ := newCacheLuaState(t)

	err := L.DoString(`
		local cache = require "kratos_cache"
		assert(cache.exists("k") == false, "missing key should not exist")
		assert(cache.set("k", "v") == true)
		assert(cache.exists("k") == true, "existing key should exist")
		assert(cache.delete("k") == true, "delete should succeed")
		assert(cache.exists("k") == false, "deleted key should not exist")
	`)
	require.NoError(t, err)
}

// TestCacheAPI_TTLExpire TTL 读取与 expire 更新：
// 无过期键报告 -1、不存在键报告 -2、对不存在键 expire 返回 false。
func TestCacheAPI_TTLExpire(t *testing.T) {
	L, _ := newCacheLuaState(t)

	err := L.DoString(`
		local cache = require "kratos_cache"

		-- 带 TTL 的键：剩余时长应在设置值附近
		assert(cache.set("temp", "v", 300) == true)
		local ttl = cache.ttl("temp")
		assert(ttl >= 290 and ttl <= 300, "ttl out of range: " .. tostring(ttl))

		-- expire 更新 TTL
		assert(cache.expire("temp", 600) == true, "expire should succeed")
		local ttl2 = cache.ttl("temp")
		assert(ttl2 >= 590 and ttl2 <= 600, "updated ttl out of range: " .. tostring(ttl2))

		-- 不带 TTL 的键：报告 -1
		assert(cache.set("perm", "v") == true)
		assert(cache.ttl("perm") == -1, "no-expiry key should report -1")

		-- 不存在的键：报告 -2
		assert(cache.ttl("nokey") == -2, "missing key should report -2")

		-- 对不存在的键 expire 返回 false
		assert(cache.expire("nokey", 5) == false, "expire on missing key should be false")
	`)
	require.NoError(t, err)
}

// TestCacheAPI_KeysPattern keys 模式匹配返回匹配键数组（仅断言数量，顺序不稳定）。
func TestCacheAPI_KeysPattern(t *testing.T) {
	L, _ := newCacheLuaState(t)

	err := L.DoString(`
		local cache = require "kratos_cache"
		cache.set("user:1", "a")
		cache.set("user:2", "b")
		cache.set("user:3", "c")
		cache.set("post:1", "d")
		local keys = cache.keys("user:*")
		assert(#keys == 3, "expected 3 user keys, got " .. tostring(#keys))
	`)
	require.NoError(t, err)
}

// TestCacheAPI_HashOps 哈希家族：hset/hget/hgetall，缺失字段返回 nil，缺失键 hgetall 返回空表。
func TestCacheAPI_HashOps(t *testing.T) {
	L, _ := newCacheLuaState(t)

	err := L.DoString(`
		local cache = require "kratos_cache"
		assert(cache.hset("h", "f1", "v1") == true, "hset should succeed")
		assert(cache.hset("h", "f2", "v2") == true, "hset should succeed")

		local f1 = cache.hget("h", "f1")
		assert(f1 == "v1", "hget mismatch: " .. tostring(f1))

		local all = cache.hgetall("h")
		assert(all.f1 == "v1", "hgetall f1 mismatch")
		assert(all.f2 == "v2", "hgetall f2 mismatch")

		-- 缺失字段 → nil
		local miss = cache.hget("h", "nope")
		assert(miss == nil, "missing field should be nil")

		-- 缺失键的 hgetall → 空表
		local empty = cache.hgetall("missing-key")
		assert(type(empty) == "table" and #empty == 0, "missing key hgetall should be empty table")
	`)
	require.NoError(t, err)
}

// TestCacheAPI_ErrorInjection miniredis 全局错误注入：
// 所有命令返回错误，各 API 以 (nil/false, 错误字符串) 形式返回。
func TestCacheAPI_ErrorInjection(t *testing.T) {
	L, mr := newCacheLuaState(t)
	mr.SetError("forced error")

	err := L.DoString(`
		local cache = require "kratos_cache"

		local v, e = cache.get("k")
		assert(v == nil, "get should return nil on error")
		assert(type(e) == "string", "get error should be string")

		local ok, e2 = cache.set("k", "v")
		assert(ok == false, "set should return false on error")
		assert(type(e2) == "string", "set error should be string")

		local ok3, e3 = cache.delete("k")
		assert(ok3 == false, "delete should return false on error")
		assert(type(e3) == "string", "delete error should be string")

		assert(cache.exists("k") == false, "exists should return false on error")

		local ok4, e4 = cache.expire("k", 10)
		assert(ok4 == false, "expire should return false on error")
		assert(type(e4) == "string", "expire error should be string")

		local iv, e5 = cache.incr("k")
		assert(iv == nil, "incr should return nil on error")
		assert(type(e5) == "string", "incr error should be string")

		local dv, e6 = cache.decr("k")
		assert(dv == nil, "decr should return nil on error")
		assert(type(e6) == "string", "decr error should be string")

		local bv, e7 = cache.incrby("k", 1)
		assert(bv == nil, "incrby should return nil on error")
		assert(type(e7) == "string", "incrby error should be string")

		assert(cache.ttl("k") == -2, "ttl should report -2 on error")

		local kv, e8 = cache.keys("*")
		assert(kv == nil, "keys should return nil on error")
		assert(type(e8) == "string", "keys error should be string")

		local hv, e9 = cache.hget("k", "f")
		assert(hv == nil, "hget should return nil on error")
		assert(type(e9) == "string", "hget error should be string")

		local hs, e10 = cache.hset("k", "f", "v")
		assert(hs == false, "hset should return false on error")
		assert(type(e10) == "string", "hset error should be string")

		local ha, e11 = cache.hgetall("k")
		assert(ha == nil, "hgetall should return nil on error")
		assert(type(e11) == "string", "hgetall error should be string")
	`)
	require.NoError(t, err)
}

// TestRegisterCache_NilClient rdb 为 nil 时 loader 返回空模块。
func TestRegisterCache_NilClient(t *testing.T) {
	L := lua.NewState()
	t.Cleanup(L.Close)
	RegisterCache(L, nil, bLogger.NewHelper(bLogger.NopLogger()))

	err := L.DoString(`
		local c = require "kratos_cache"
		assert(type(c) == "table", "nil client should still yield a table")
		assert(c.get == nil, "empty module should have no get")
		assert(c.set == nil, "empty module should have no set")
	`)
	require.NoError(t, err)
}

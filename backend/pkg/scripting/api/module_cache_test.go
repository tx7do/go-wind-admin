// 本文件针对 module_cache.go（JS 桥接版 cache 模块）做单元测试。
// Lua 版（cache.go）已有 cache_module_test.go 全面覆盖；本文件验证 JS 版函数
// 与 Lua 版语义对齐：命名空间归一、JSON 往返、redis.Nil → (nil, nil)、
// 值限额、TTL 分支，以及 nil rdb 的 fail-closed 空模块。
// Redis 侧同样使用 miniredis，无需外部 Redis。
package api

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
)

// newJSCacheFuncs 构造绑定 miniredis 的 JS 版 cache 模块函数表。
func newJSCacheFuncs(t *testing.T) (map[string]any, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	mod := ModuleCache(rdb, bLogger.NewHelper(bLogger.NopLogger()))
	return mod.Funcs, mr
}

func jsFunc[T any](t *testing.T, funcs map[string]any, name string) T {
	t.Helper()
	fn, ok := funcs[name].(T)
	require.Truef(t, ok, "funcs[%q] should be of expected type", name)
	return fn
}

func TestModuleCache_NilClient(t *testing.T) {
	mod := ModuleCache(nil, nil)
	require.Equal(t, "cache", mod.Name)
	require.Empty(t, mod.Funcs, "nil rdb must produce an empty (fail-closed) module")
}

func TestModuleCache_StringRoundTrip(t *testing.T) {
	funcs, mr := newJSCacheFuncs(t)
	set := jsFunc[func(string, any, ...int) (bool, error)](t, funcs, "set")
	get := jsFunc[func(string) (any, error)](t, funcs, "get")

	ok, err := set("greeting", "hello", 60)
	require.NoError(t, err)
	require.True(t, ok)

	// 键经 script: 命名空间归一落库。
	v, err := mr.Get("script:greeting")
	require.NoError(t, err)
	require.Equal(t, "hello", v)

	got, err := get("greeting")
	require.NoError(t, err)
	require.Equal(t, "hello", got)

	// 缺键 → (nil, nil)，不报错（JS 侧拿到 undefined）。
	miss, err := get("missing-key")
	require.NoError(t, err)
	require.Nil(t, miss)
}

func TestModuleCache_JSONRoundTrip(t *testing.T) {
	funcs, _ := newJSCacheFuncs(t)
	set := jsFunc[func(string, any, ...int) (bool, error)](t, funcs, "set")
	get := jsFunc[func(string) (any, error)](t, funcs, "get")

	obj := map[string]any{"name": "Alice", "n": float64(3)}
	ok, err := set("obj", obj)
	require.NoError(t, err)
	require.True(t, ok)

	got, err := get("obj")
	require.NoError(t, err)
	require.Equal(t, map[string]any{"name": "Alice", "n": float64(3)}, got)
}

func TestModuleCache_SetNilAndTTLVariants(t *testing.T) {
	funcs, mr := newJSCacheFuncs(t)
	set := jsFunc[func(string, any, ...int) (bool, error)](t, funcs, "set")
	ttl := jsFunc[func(string) (int64, error)](t, funcs, "ttl")

	// nil → 空串。
	ok, err := set("nilval", nil)
	require.NoError(t, err)
	require.True(t, ok)
	v, err := mr.Get("script:nilval")
	require.NoError(t, err)
	require.Equal(t, "", v)

	// 正 TTL 生效。
	ok, err = set("withttl", "x", 120)
	require.NoError(t, err)
	require.True(t, ok)
	d := mr.TTL("script:withttl")
	require.Equal(t, 120*time.Second, d)

	// TTL ≤ 0 → 不过期（miniredis 对无过期键报 0s，真实 Redis 语义为 -1；
	// 这里只钉"没有设置到期时间"这一事实）。
	ok, err = set("nottl", "x", 0)
	require.NoError(t, err)
	require.True(t, ok)
	d = mr.TTL("script:nottl")
	require.LessOrEqual(t, int64(d), int64(0), "no expiry expected for ttl<=0")

	ok, err = set("negttl", "x", -5)
	require.NoError(t, err)
	require.True(t, ok)
	d = mr.TTL("script:negttl")
	require.LessOrEqual(t, int64(d), int64(0))

	got, err := ttl("withttl")
	require.NoError(t, err)
	require.Positive(t, got)
}

func TestModuleCache_ValueSizeLimit(t *testing.T) {
	funcs, _ := newJSCacheFuncs(t)
	set := jsFunc[func(string, any, ...int) (bool, error)](t, funcs, "set")
	hset := jsFunc[func(string, string, string) (bool, error)](t, funcs, "hset")

	big := make([]byte, 64<<10+1)
	ok, err := set("big", string(big))
	require.Error(t, err, "value above 64KB must be rejected")
	require.False(t, ok)

	ok, err = hset("h", "f", string(big))
	require.Error(t, err)
	require.False(t, ok)
}

func TestModuleCache_EmptyKeyRejected(t *testing.T) {
	funcs, _ := newJSCacheFuncs(t)
	set := jsFunc[func(string, any, ...int) (bool, error)](t, funcs, "set")
	get := jsFunc[func(string) (any, error)](t, funcs, "get")
	del := jsFunc[func(string) (bool, error)](t, funcs, "delete")
	exists := jsFunc[func(string) (bool, error)](t, funcs, "exists")
	incr := jsFunc[func(string) (int64, error)](t, funcs, "incr")

	_, err := set("", "x")
	require.Error(t, err)
	_, err = get("")
	require.Error(t, err)
	_, err = del("")
	require.Error(t, err)
	_, err = exists("")
	require.Error(t, err)
	_, err = incr("")
	require.Error(t, err)
}

func TestModuleCache_ExistsDeleteExpire(t *testing.T) {
	funcs, _ := newJSCacheFuncs(t)
	set := jsFunc[func(string, any, ...int) (bool, error)](t, funcs, "set")
	exists := jsFunc[func(string) (bool, error)](t, funcs, "exists")
	del := jsFunc[func(string) (bool, error)](t, funcs, "delete")
	expire := jsFunc[func(string, int) (bool, error)](t, funcs, "expire")

	_, err := set("ed", "v")
	require.NoError(t, err)

	ok, err := exists("ed")
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = expire("ed", 30)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = expire("missing", 30)
	require.NoError(t, err)
	require.False(t, ok, "expire on missing key returns false")

	ok, err = del("ed")
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = exists("ed")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestModuleCache_IncrDecr(t *testing.T) {
	funcs, _ := newJSCacheFuncs(t)
	incr := jsFunc[func(string) (int64, error)](t, funcs, "incr")
	decr := jsFunc[func(string) (int64, error)](t, funcs, "decr")
	incrby := jsFunc[func(string, int) (int64, error)](t, funcs, "incrby")

	n, err := incr("counter")
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	n, err = incr("counter")
	require.NoError(t, err)
	require.Equal(t, int64(2), n)

	n, err = decr("counter")
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	n, err = incrby("counter", 10)
	require.NoError(t, err)
	require.Equal(t, int64(11), n)

	n, err = incrby("counter", -5)
	require.NoError(t, err)
	require.Equal(t, int64(6), n)
}

func TestModuleCache_TTLTriState(t *testing.T) {
	funcs, _ := newJSCacheFuncs(t)
	set := jsFunc[func(string, any, ...int) (bool, error)](t, funcs, "set")
	ttl := jsFunc[func(string) (int64, error)](t, funcs, "ttl")

	got, err := ttl("no-such-key")
	require.NoError(t, err)
	require.EqualValues(t, -2, got, "missing key → -2")

	_, err = set("forever", "x")
	require.NoError(t, err)
	got, err = ttl("forever")
	require.NoError(t, err)
	require.EqualValues(t, -1, got, "no expiry → -1")

	_, err = set("timed", "x", 300)
	require.NoError(t, err)
	got, err = ttl("timed")
	require.NoError(t, err)
	require.Positive(t, got)
	require.LessOrEqual(t, got, int64(300))
}

func TestModuleCache_KeysAndHashes(t *testing.T) {
	funcs, mr := newJSCacheFuncs(t)
	set := jsFunc[func(string, any, ...int) (bool, error)](t, funcs, "set")
	keys := jsFunc[func(string) ([]string, error)](t, funcs, "keys")
	hset := jsFunc[func(string, string, string) (bool, error)](t, funcs, "hset")
	hget := jsFunc[func(string, string) (any, error)](t, funcs, "hget")
	hgetall := jsFunc[func(string) (map[string]string, error)](t, funcs, "hgetall")

	// 平台资产键（不在 script: 命名空间）不得被 keys(pattern) 看到。
	require.NoError(t, mr.Set("at:token-1", "secret"))
	require.NoError(t, mr.Set("mfa:secret-1", "secret"))

	for _, k := range []string{"user:1", "user:2", "other:3"} {
		ok, err := set(k, "v")
		require.NoError(t, err)
		require.True(t, ok)
	}

	found, err := keys("user:*")
	require.NoError(t, err)
	// 返回的是带 script: 前缀的全名（脚本侧可见形态）。
	require.ElementsMatch(t, []string{"script:user:1", "script:user:2"}, found)

	// 全量模式同样扫不出命名空间外的平台资产键。
	all, err := keys("*")
	require.NoError(t, err)
	require.NotEmpty(t, all)
	for _, k := range all {
		require.True(t, len(k) >= len("script:"), "keys must stay inside the script: namespace")
		require.False(t, k == "at:token-1" || k == "mfa:secret-1", "platform asset keys must never leak")
	}

	ok, err := hset("myhash", "f1", "v1")
	require.NoError(t, err)
	require.True(t, ok)

	got, err := hget("myhash", "f1")
	require.NoError(t, err)
	require.Equal(t, "v1", got)

	got, err = hget("myhash", "no-field")
	require.NoError(t, err)
	require.Nil(t, got)

	hashAll, err := hgetall("myhash")
	require.NoError(t, err)
	require.Equal(t, map[string]string{"f1": "v1"}, hashAll)

	hashAll, err = hgetall("no-such-hash")
	require.NoError(t, err)
	require.Empty(t, hashAll)
}

func TestModuleCache_KeysTruncatesAtLimit(t *testing.T) {
	funcs, _ := newJSCacheFuncs(t)
	set := jsFunc[func(string, any, ...int) (bool, error)](t, funcs, "set")
	keys := jsFunc[func(string) ([]string, error)](t, funcs, "keys")

	for i := 0; i < 250; i++ {
		ok, err := set("bulk:"+string(rune('a'+i%26))+string(rune(i)), "v")
		require.NoError(t, err)
		require.True(t, ok)
	}
	found, err := keys("bulk:*")
	require.NoError(t, err)
	require.Len(t, found, 200, "keys must be capped at scriptCacheMaxKeys=200 to avoid blocking the instance")
}

func TestModuleCache_RedisErrorPropagates(t *testing.T) {
	funcs, mr := newJSCacheFuncs(t)
	set := jsFunc[func(string, any, ...int) (bool, error)](t, funcs, "set")
	get := jsFunc[func(string) (any, error)](t, funcs, "get")

	mr.SetError("forced error")
	_, err := set("k", "v")
	require.Error(t, err)
	_, err = get("k")
	require.Error(t, err)
}

// --- 基准：脚本缓存最高频路径 = set/get 序列化往返 ---

func BenchmarkModuleCache_SetGetString(b *testing.B) {
	mr := miniredis.RunT(&testing.T{})
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()
	mod := ModuleCache(rdb, nil)
	set := mod.Funcs["set"].(func(string, any, ...int) (bool, error))
	get := mod.Funcs["get"].(func(string) (any, error))

	b.Run("string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := set("bench:str", "payload-value"); err != nil {
				b.Fatal(err)
			}
			if _, err := get("bench:str"); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("json-object", func(b *testing.B) {
		obj := map[string]any{"name": "Alice", "score": 42, "tags": []any{"a", "b", "c"}}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := set("bench:obj", obj); err != nil {
				b.Fatal(err)
			}
			if _, err := get("bench:obj"); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkModuleCache_ScriptCacheKey(b *testing.B) {
	mr := miniredis.RunT(&testing.T{})
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()
	mod := ModuleCache(rdb, nil)
	exists := mod.Funcs["exists"].(func(string) (bool, error))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := exists("bench:key"); err != nil {
			b.Fatal(err)
		}
	}
}

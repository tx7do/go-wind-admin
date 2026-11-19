// 本文件针对 oss.go（Lua 版 kratos_oss 模块）与 module_oss.go（JS 桥接版）做
// 【离线可测子集】的单元测试。真网络路径（list/delete/ensure 的远端交互）不在此列；
// 离线可测的三类：
//   1. 桶白名单（P2 安全修复：脚本不得触碰 backups 等平台系统桶）；
//   2. 参数校验错误路径（缺 content_type / bucket_name）；
//   3. PresignedPutObject 是本地 HMAC 签名，不触网，可测成功路径与结果表四字段。
// 另钉住 JS 版 uploadUrl 当前【未做】桶白名单校验的现状（与 Lua 版语义不一致，
// 若后续补上白名单，TestModuleOSS_UploadUrlNoWhitelist_PinnedCurrentBehavior 的
// NoError 断言要同步翻转为 Error）。
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	lua "github.com/yuin/gopher-lua"

	"go-wind-admin/pkg/oss"
)

// newOfflineOSSClient 构造指向本地 httptest 桩的 MinIOClient。
// minio-go 的 PresignedPutObject 会先做一次桶 location 查询（GET /bucket/?location=），
// 这里用 httptest 返回固定 XML 应答，使预签名路径完全离线；签名本身仍是本地 HMAC。
func newOfflineOSSClient(t *testing.T) *oss.MinIOClient {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><LocationConstraint>us-east-1</LocationConstraint>`))
	}))
	t.Cleanup(server.Close)

	// httptest 地址形如 127.0.0.1:port，minio-go 要求 endpoint 不带 scheme。
	endpoint := strings.TrimPrefix(strings.TrimPrefix(server.URL, "http://"), "https://")
	client := oss.NewMinIoClient(&conf.Bootstrap{
		Oss: &conf.OSS{
			Minio: &conf.OSS_MinIO{
				Endpoint:  endpoint,
				AccessKey: "test",
				SecretKey: "test",
			},
		},
	}, bLogger.NopLogger())
	return client
}

// --- 桶白名单（checkScriptBucket，Lua/JS 共用这道闸的只有 Lua 版） ---

func TestCheckScriptBucket(t *testing.T) {
	for _, allowed := range []string{"images", "videos", "audios", "docs", "files"} {
		require.NoErrorf(t, checkScriptBucket(allowed), "business bucket %q must be allowed", allowed)
	}
	for _, denied := range []string{"", "backups", "system", "images-backup", "Images", "other"} {
		err := checkScriptBucket(denied)
		require.Errorf(t, err, "bucket %q must be denied", denied)
		require.Contains(t, err.Error(), "is not allowed for scripts")
	}
}

// --- Lua 版离线路径 ---

func TestLoaderOSS_NilClient(t *testing.T) {
	L := lua.NewState()
	t.Cleanup(L.Close)
	L.PreloadModule("kratos_oss", LoaderOSS(nil, nil))
	require.NoError(t, L.DoString(`
		local oss = require "kratos_oss"
		assert(type(oss) == "table", "module should be a table")
		assert(oss.upload_url == nil, "nil client must not expose upload_url")
		assert(oss.list_files == nil, "nil client must not expose list_files")
	`))
}

func TestOSSAPI_LuaUploadUrlValidation(t *testing.T) {
	L := lua.NewState()
	t.Cleanup(L.Close)
	L.PreloadModule("kratos_oss", LoaderOSS(newOfflineOSSClient(t), bLogger.NewHelper(bLogger.NopLogger())))

	// 现状钉住：缺 content_type 不报错——RawGetString 对缺省键返回 LNil，
	// 其 String() 是字面 "nil"（非空），于是按未知类型降级落 files 桶继续预签名。
	// 若后续把这里改为显式报错，请同步更新本断言。
	require.NoError(t, L.DoString(`require "kratos_oss".upload_url({file_name = "a.jpg"})`))
	// 显式空串 content_type 才会进 "content_type is required" 报错分支。
	require.Error(t, L.DoString(`require "kratos_oss".upload_url({content_type = ""})`))
	// 显式给平台系统桶 → 白名单拒绝。
	require.Error(t, L.DoString(`require "kratos_oss".upload_url({content_type = "image/jpeg", bucket_name = "backups"})`))
	// 未知的 content type 降级落 files 桶（白名单内），不会被拒——钉住该降级语义。
	require.NoError(t, L.DoString(`require "kratos_oss".upload_url({content_type = "weird/type"})`))
}

func TestOSSAPI_LuaUploadUrlPresign(t *testing.T) {
	L := lua.NewState()
	t.Cleanup(L.Close)
	L.PreloadModule("kratos_oss", LoaderOSS(newOfflineOSSClient(t), bLogger.NewHelper(bLogger.NopLogger())))

	require.NoError(t, L.DoString(`
		local oss = require "kratos_oss"

		local res = oss.upload_url({content_type = "image/jpeg", file_name = "photo.jpg", file_path = "uploads"})
		assert(type(res) == "table", "result should be a table")
		assert(type(res.upload_url) == "string" and #res.upload_url > 0, "upload_url should be a presigned URL string")
		assert(res.bucket_name == "images", "image/jpeg maps to images bucket, got " .. tostring(res.bucket_name))
		assert(res.download_url == "/images/" .. res.object_name, "download_url should be /<bucket>/<object>")
		assert(type(res.object_name) == "string" and #res.object_name > 0, "object_name should be generated")
		assert(string.find(res.object_name, "photo%.jpg", 1, false) ~= nil, "file name should appear in object name")
	`))
}

func TestOSSAPI_LuaGetBucketForType(t *testing.T) {
	L := lua.NewState()
	t.Cleanup(L.Close)
	L.PreloadModule("kratos_oss", LoaderOSS(newOfflineOSSClient(t), nil))

	require.NoError(t, L.DoString(`
		local oss = require "kratos_oss"
		assert(oss.get_bucket_for_type("image/jpeg") == "images")
		assert(oss.get_bucket_for_type("video/mp4") == "videos")
		assert(oss.get_bucket_for_type("audio/mpeg") == "audios")
		assert(oss.get_bucket_for_type("application/pdf") == "docs")
	`))
}

// --- JS 桥接版离线路径 ---

func TestModuleOSS_NilClient(t *testing.T) {
	mod := ModuleOSS(nil, nil)
	require.Equal(t, "oss", mod.Name)
	require.Empty(t, mod.Funcs, "nil client must produce an empty (fail-closed) module")
}

func TestModuleOSS_UploadUrlValidation(t *testing.T) {
	funcs := ModuleOSS(newOfflineOSSClient(t), nil).Funcs
	uploadUrl := funcs["uploadUrl"].(func(map[string]any) (map[string]any, error))

	_, err := uploadUrl(map[string]any{})
	require.ErrorContains(t, err, "content_type is required")

	_, err = uploadUrl(nil)
	require.ErrorContains(t, err, "content_type is required", "nil options map must not panic")
}

func TestModuleOSS_UploadUrlPresign(t *testing.T) {
	funcs := ModuleOSS(newOfflineOSSClient(t), nil).Funcs
	uploadUrl := funcs["uploadUrl"].(func(map[string]any) (map[string]any, error))

	res, err := uploadUrl(map[string]any{"content_type": "image/jpeg", "file_name": "photo.jpg"})
	require.NoError(t, err)
	require.Equal(t, "images", res["bucket_name"])
	require.NotEmpty(t, res["upload_url"])
	require.NotEmpty(t, res["object_name"])
	require.Equal(t, "/images/"+res["object_name"].(string), res["download_url"])
}

func TestModuleOSS_UploadUrlNoWhitelist_PinnedCurrentBehavior(t *testing.T) {
	// 现状钉住：JS 版 uploadUrl 未做 checkScriptBucket 校验（Lua 版每个函数都校验）。
	// 平台系统桶同样能拿到预签名 URL。若后续补上白名单，请把本断言改为 require.Error。
	funcs := ModuleOSS(newOfflineOSSClient(t), nil).Funcs
	uploadUrl := funcs["uploadUrl"].(func(map[string]any) (map[string]any, error))

	res, err := uploadUrl(map[string]any{"content_type": "image/jpeg", "bucket_name": "backups"})
	require.NoError(t, err, "current JS implementation has no bucket whitelist on uploadUrl")
	require.Equal(t, "backups", res["bucket_name"])
}

func TestModuleOSS_GetBucketForType(t *testing.T) {
	funcs := ModuleOSS(newOfflineOSSClient(t), nil).Funcs
	getBucketForType := funcs["getBucketForType"].(func(string) string)

	require.Equal(t, "images", getBucketForType("image/jpeg"))
	require.Equal(t, "videos", getBucketForType("video/mp4"))
	require.Equal(t, "audios", getBucketForType("audio/mpeg"))
	require.Equal(t, "docs", getBucketForType("application/pdf"))
}

func TestTrimEtag(t *testing.T) {
	require.Equal(t, "abc123", trimEtag(`"abc123"`))
	require.Equal(t, "abc123", trimEtag("abc123"))
	require.Equal(t, `"`, trimEtag(`"`), "single quote is not a wrapped etag")
	require.Empty(t, trimEtag(""), "empty etag stays empty (len<2 guard)")
	require.Equal(t, ``, trimEtag(`""`), "empty quoted etag strips to empty")
}

// --- 基准 ---

// BenchmarkModuleOSS_UploadUrlPresign JoinObjectName + HMAC 签名 + URL 构造，分配密集。
// 复用 httptest location 桩（minio-go 预签名前的桶查询），保持基准离线。
func BenchmarkModuleOSS_UploadUrlPresign(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<LocationConstraint>us-east-1</LocationConstraint>`))
	}))
	defer server.Close()
	endpoint := strings.TrimPrefix(strings.TrimPrefix(server.URL, "http://"), "https://")
	client := oss.NewMinIoClient(&conf.Bootstrap{
		Oss: &conf.OSS{
			Minio: &conf.OSS_MinIO{Endpoint: endpoint, AccessKey: "test", SecretKey: "test"},
		},
	}, bLogger.NopLogger())
	uploadUrl := ModuleOSS(client, nil).Funcs["uploadUrl"].(func(map[string]any) (map[string]any, error))
	opts := map[string]any{"content_type": "image/jpeg", "file_name": "bench.jpg"}
	// 预热一次，让桶 location 查询结果进入 minio-go 的进程内缓存。
	if _, err := uploadUrl(opts); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := uploadUrl(opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCheckScriptBucket(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = checkScriptBucket("images")
		_ = checkScriptBucket("backups")
	}
}

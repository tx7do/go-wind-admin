// FileTransferService 的 SQLite 内存库 + httptest 测试（白盒，包内测试）。
//
// 上传/下载链路的中段（MinIO 对象读写）依赖真实 MinIO，打桩不到；按约定不硬凑，
// 测试火力集中在"打桩不到也不该放过去"的部分：
//   - 纯函数口径：parseKey/joinObjectPath 是元数据落库与按元数据重建 object key 的
//     唯一路径拼装点，此前缺 "/" 分隔的 bug 就出在这一层（下载永远 404）；
//   - 直传守卫链：缺存储对象/缺文件/缺 MIME/缺原始文件名/超 50MiB/内容嗅探不在
//     白名单（HTML 伪装）/路径穿越目录，全部挡在触碰 MinIO 之前；
//   - 套餐 STORAGE 硬限制：租户用户上传前配额校验真实生效（当前占用+本次 > 上限拒绝）；
//   - ServeImageHandler 签名闸：缺参 400、过期 403、无全局密钥时验签恒拒
//     （签名即凭证，密钥未配置时 fail-closed）；
//   - DownloadFile 分发：未知选择器拒绝；FileId 走元数据（不存在即失败，不触 MinIO）；
//     DownloadUrl 走 SSRF 防护（空 URL、非法 scheme、内网 IP 各自拦截）。
package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/ossutil"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	storageV1 "go-wind-admin/api/gen/go/storage/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/planquota"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newFileTransferServiceForTest 白盒装配 FileTransferService：log 换 NopLogger、
// 元数据 repo 走 SQLite 内存库、mc 置 nil（被测分支全部在触碰 MinIO 之前返回；
// 越过守卫即 nil panic，反而是守卫失效的警报）。
func newFileTransferServiceForTest(t *testing.T) (*FileTransferService, *entCrud.EntClient[*ent.Client]) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &FileTransferService{
		log:               bLogger.NewHelper(bLogger.NopLogger()),
		fileServiceClient: data.NewFileRepoForTest(entClient),
	}, entClient
}

// pngContent 构造带 PNG 魔头的最小内容（内容嗅探必判 image/png，命中上传白名单）。
func pngContent(extra int) []byte {
	b := make([]byte, 0, 8+extra)
	b = append(b, "\x89PNG\r\n\x1a\n"...)
	for i := 0; i < extra; i++ {
		b = append(b, 0x00)
	}
	return b
}

// ftUploadReq 构造一份能通过前置守卫的直传请求（内容/目录可覆盖）。
func ftUploadReq(content []byte, directory string) *storageV1.UploadFileRequest {
	return &storageV1.UploadFileRequest{
		StorageObject:  &storageV1.StorageObject{FileDirectory: trans.Ptr(directory)},
		Source:         &storageV1.UploadFileRequest_File{File: content},
		SourceFileName: trans.Ptr("photo.png"),
		Mime:           trans.Ptr("image/png"),
	}
}

// TestFileTransfer_ParseKeyAndJoinObjectPath 验证元数据键位拆装口径：
// parseKey 拆目录/文件名/小写扩展名（点文件与多层扩展名各有归属）；
// joinObjectPath 补 "/" 且不产生重复斜杠——下载/删除按元数据重建 key 全靠它。
func TestFileTransfer_ParseKeyAndJoinObjectPath(t *testing.T) {
	cases := []struct {
		key               string
		folder, name, ext string
	}{
		{key: "", folder: "", name: "", ext: ""},
		{key: "a.txt", folder: "", name: "a", ext: "txt"},
		{key: "dir/b/c.PNG", folder: "dir/b", name: "c", ext: "png"},
		{key: "/x/y", folder: "x", name: "y", ext: ""},
		{key: "folder/", folder: "folder", name: "", ext: ""},
		{key: ".env", folder: "", name: ".env", ext: ""},
		{key: "archive.tar.gz", folder: "", name: "archive.tar", ext: "gz"},
		{key: "noext", folder: "", name: "noext", ext: ""},
	}
	for _, c := range cases {
		folder, name, ext := parseKey(c.key)
		require.Equal(t, c.folder, folder, "parseKey(%q) 目录部分", c.key)
		require.Equal(t, c.name, name, "parseKey(%q) 文件名部分", c.key)
		require.Equal(t, c.ext, ext, "parseKey(%q) 扩展名部分（应小写）", c.key)
	}

	require.Equal(t, "dir/f.jpg", joinObjectPath("dir", "f.jpg"))
	require.Equal(t, "f.jpg", joinObjectPath("", "f.jpg"), "目录为空时不加前缀斜杠")
	require.Equal(t, "dir/f.jpg", joinObjectPath("dir/", "/f.jpg"), "不得产生重复斜杠")
	require.Equal(t, "dir/f.jpg", joinObjectPath("/dir/", "f.jpg"))
}

// TestFileTransfer_UploadFileGuards 验证直传守卫链的每个闸门：
// 全部在触碰 MinIO（此处为 nil）之前返回，越闸即 panic 是守卫失效的警报。
func TestFileTransfer_UploadFileGuards(t *testing.T) {
	svc, _ := newFileTransferServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	// 分发器：未知来源 / 预签名路径已禁用
	_, err := svc.UploadFile(ctx, &storageV1.UploadFileRequest{})
	require.Error(t, err, "未知上传来源应被拒绝")
	require.Contains(t, err.Error(), "unknown upload source")
	_, err = svc.UploadFile(ctx, &storageV1.UploadFileRequest{
		Source: &storageV1.UploadFileRequest_Presign{Presign: &storageV1.PresignOption{}},
	})
	require.Error(t, err, "预签名路径应拒绝（元数据无法可靠落库）")
	require.Contains(t, err.Error(), "presigned upload is not implemented")

	// 直传守卫链（顺序即实现顺序，逐个闸门验证）
	_, err = svc.UploadFile(ctx, &storageV1.UploadFileRequest{
		Source: &storageV1.UploadFileRequest_File{File: pngContent(4)},
	})
	require.Error(t, err, "缺存储对象应被拒绝")
	require.Contains(t, err.Error(), "unknown storageObject")

	_, err = svc.UploadFile(ctx, &storageV1.UploadFileRequest{
		StorageObject: &storageV1.StorageObject{},
		Source:        &storageV1.UploadFileRequest_File{},
	})
	require.Error(t, err, "缺文件内容应被拒绝")
	require.Contains(t, err.Error(), "unknown fileData")

	noMime := ftUploadReq(pngContent(4), "")
	noMime.Mime = nil
	_, err = svc.UploadFile(ctx, noMime)
	require.Error(t, err, "缺 MIME 应被拒绝")
	require.Contains(t, err.Error(), "unknown mime type")

	noName := ftUploadReq(pngContent(4), "")
	noName.SourceFileName = nil
	_, err = svc.UploadFile(ctx, noName)
	require.Error(t, err, "缺原始文件名应被拒绝")
	require.Contains(t, err.Error(), "unknown source file name")

	oversize := ftUploadReq(make([]byte, ossutil.MaxUploadSize+1), "")
	_, err = svc.UploadFile(ctx, oversize)
	require.Error(t, err, "超 50MiB 应被拒绝")
	require.Contains(t, err.Error(), "exceeds max upload size")

	// 内容嗅探挡伪装：声明 image/png 的 HTML 内容按真实类型 text/html 拒绝
	htmlDisguise := ftUploadReq([]byte("<html><body>xss</body></html>"), "")
	_, err = svc.UploadFile(ctx, htmlDisguise)
	require.Error(t, err, "HTML 内容应被白名单拒绝（防伪装进可渲染 bucket）")
	require.Contains(t, err.Error(), "is not allowed")

	// 路径穿越：内容合法但目录带 .. 必须拒绝（防 OSS 命名空间注入）
	traversal := ftUploadReq(pngContent(4), "../evil")
	_, err = svc.UploadFile(ctx, traversal)
	require.Error(t, err, "路径穿越目录应被拒绝")
	require.Contains(t, err.Error(), "invalid file directory")
}

// TestFileTransfer_UploadFileStorageQuota 验证套餐 STORAGE 硬限制：
// 租户用户上传前做配额校验，当前占用 + 本次大小超上限即拒绝（套餐闸在 MinIO 之前）。
func TestFileTransfer_UploadFileStorageQuota(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	// 种子：租户 → 套餐（STORAGE 配额 150 字节）→ 已占用 100 字节的文件行
	plan, err := entClient.Client().Plan.Create().SetName("基础套餐").Save(ctx)
	require.NoError(t, err, "播种套餐应成功")
	require.NoError(t, entClient.Client().PlanQuota.Create().
		SetPlan(plan).
		SetQuotaType(planquota.QuotaTypeStorage).
		SetQuotaValue(150).
		Exec(ctx), "播种存储配额应成功")
	tenant, err := entClient.Client().Tenant.Create().
		SetName("租户七").
		SetCode("t7").
		SetPlan(plan).
		Save(ctx)
	require.NoError(t, err, "播种租户应成功")
	_, err = entClient.Client().File.Create().
		SetTenantID(tenant.ID).
		SetSize(100).
		Save(ctx)
	require.NoError(t, err, "播种已占用文件行应成功")

	svc := &FileTransferService{
		log:               bLogger.NewHelper(bLogger.NopLogger()),
		fileServiceClient: data.NewFileRepoForTest(entClient),
		usageRepo:         data.NewTenantUsageRepoForTest(entClient, nil),
	}
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{
		UserId:   9,
		TenantId: &tenant.ID,
	})

	// 已占用 100 + 本次 200 > 配额 150 → 拒绝
	_, err = svc.UploadFile(opCtx, ftUploadReq(pngContent(200-8), ""))
	require.Error(t, err, "超套餐存储配额的上传应被拒绝")
	require.Contains(t, err.Error(), "plan storage quota", "拒绝文案要能被前端识别为配额原因")
}

// TestFileTransfer_ServeImageHandlerGuards 验证签名图片代理的鉴权闸：
// 缺参 400、过期 403、无全局密钥时验签恒拒（fail-closed，签名即凭证不能有默认口令）。
func TestFileTransfer_ServeImageHandlerGuards(t *testing.T) {
	svc, _ := newFileTransferServiceForTest(t)

	// 缺参数
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/v1/file/image", nil)
	require.NoError(t, svc.ServeImageHandler(rec, req))
	require.Equal(t, http.StatusBadRequest, rec.Code, "缺参数应回 400")

	// 已过期
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet,
		"/admin/v1/file/image?path=img/a.png&expires=1000&sig=deadbeef", nil)
	require.NoError(t, svc.ServeImageHandler(rec, req))
	require.Equal(t, http.StatusForbidden, rec.Code, "过期 URL 应回 403")
	require.Contains(t, rec.Body.String(), "url expired")

	// 参数齐全但签名不可验（全局密钥未配置：SignData/VerifyData fail-closed）
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet,
		"/admin/v1/file/image?path=img/a.png&expires=99999999999&sig=deadbeef", nil)
	require.NoError(t, svc.ServeImageHandler(rec, req))
	require.Equal(t, http.StatusForbidden, rec.Code, "验签失败应回 403，未配置密钥时不得放行")
	require.Contains(t, rec.Body.String(), "invalid signature")
}

// TestFileTransfer_DownloadFileDispatcher 验证下载分发与 SSRF 防护：
// 未知选择器拒绝；FileId 走元数据（不存在即失败，不触 MinIO）；DownloadUrl 的
// 空 URL / 非法 scheme / 内网 IP 各自拦截。
func TestFileTransfer_DownloadFileDispatcher(t *testing.T) {
	svc, entClient := newFileTransferServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	// 未知选择器
	_, err := svc.DownloadFile(ctx, &storageV1.DownloadFileRequest{})
	require.Error(t, err, "未知下载选择器应被拒绝")
	require.Contains(t, err.Error(), "unknown download selector")

	// FileId：元数据不存在即失败（在触碰 MinIO 之前）
	_, err = svc.DownloadFile(ctx, &storageV1.DownloadFileRequest{
		Selector: &storageV1.DownloadFileRequest_FileId{FileId: 987654},
	})
	require.Error(t, err, "文件元数据不存在应失败")
	require.Contains(t, err.Error(), "file not found")

	// DownloadUrl 分支已收归平台管理员：普通操作者先被闸门拒绝
	plainCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 42})
	_, err = svc.DownloadFile(plainCtx, &storageV1.DownloadFileRequest{
		Selector: &storageV1.DownloadFileRequest_DownloadUrl{},
	})
	require.Error(t, err, "非平台管理员的代取 URL 下载应被闸门拒绝")

	// DownloadUrl 各校验分支（平台管理员上下文）
	adminCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{
		UserId:          1,
		IsPlatformAdmin: trans.Ptr(true),
	})

	// DownloadUrl：空 URL
	_, err = svc.DownloadFile(adminCtx, &storageV1.DownloadFileRequest{
		Selector: &storageV1.DownloadFileRequest_DownloadUrl{},
	})
	require.Error(t, err, "空下载地址应被拒绝")
	require.Contains(t, err.Error(), "empty download url")

	// DownloadUrl：非 http(s) scheme
	_, err = svc.DownloadFile(adminCtx, &storageV1.DownloadFileRequest{
		Selector: &storageV1.DownloadFileRequest_DownloadUrl{DownloadUrl: "ftp://example.com/file"},
	})
	require.Error(t, err, "非法 scheme 应被拒绝")
	require.Contains(t, err.Error(), "invalid download url")

	// DownloadUrl：带用户信息的混淆 URL
	_, err = svc.DownloadFile(adminCtx, &storageV1.DownloadFileRequest{
		Selector: &storageV1.DownloadFileRequest_DownloadUrl{DownloadUrl: "http://user:pass@example.com/file"},
	})
	require.Error(t, err, "携带用户信息的 URL 应被拒绝")
	require.Contains(t, err.Error(), "invalid download url")

	// DownloadUrl：内网回环地址（SSRF 闸，无需真实网络）
	_, err = svc.DownloadFile(adminCtx, &storageV1.DownloadFileRequest{
		Selector: &storageV1.DownloadFileRequest_DownloadUrl{DownloadUrl: "http://127.0.0.1/secret"},
	})
	require.Error(t, err, "内网地址应被 SSRF 闸拦截")
	require.Contains(t, err.Error(), "blocked download host")

	// FileId 命中元数据后必然触达 MinIO（nil）——主路径不覆盖，见文件头说明
	require.NoError(t, entClient.Client().File.Create().
		SetBucketName("bucket-a").
		SetFileDirectory("dir").
		SetSaveFileName("a.png").
		Exec(ctx), "播种文件元数据应成功")
}

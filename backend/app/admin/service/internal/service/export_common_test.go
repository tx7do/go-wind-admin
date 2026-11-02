// export_common 的单元测试（白盒，包内测试）。
//
// 手动导出路由不穿 kratos 中间件链，auth/ent 两层上下文全靠 authorizeExportRequest
// 手工重建——这是"鉴权缺席就全站裸奔"的关键缝，测试盯三件事：
//   - 鉴权闸：缺 Bearer 前缀 / 缺 token / 校验失败一律 401；平台管理员重建
//     SystemContext、租户用户重建 UserContext（viewer 租户号必须与操作人一致，
//     这是 repo 层租户收窄的种子）；
//   - 参数解析：format 缺省 xlsx、非法值 400；maxRows 只能在资源硬顶内往下调；
//     query 原样透传（"导出的就是当前搜索看到的"）；
//   - 产物字节：CSV 手写 RFC 4180 转义 + UTF-8 BOM（Excel 中文识别）；XLSX 可被
//     excelize 原样读回（表头 + 行数据逐格相等）。
package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	"github.com/xuri/excelize/v2"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-crud/viewer"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"go-wind-admin/pkg/middleware/auth"
)

// newExportNopLogger 造导出用的静默日志器。
func newExportNopLogger() *bLogger.Helper {
	return bLogger.NewHelper(bLogger.NopLogger())
}

// fakeExportTokenChecker 用函数适配 auth.AccessTokenChecker（导出路由的 token 校验位）。
type fakeExportTokenChecker struct {
	valid    bool
	payload  *authenticationV1.UserTokenPayload
	blockErr bool
}

func (f *fakeExportTokenChecker) IsValidAccessToken(_ context.Context, _ string, _ bool) (bool, *authenticationV1.UserTokenPayload) {
	if f.blockErr {
		return false, nil
	}
	return f.valid, f.payload
}

func (f *fakeExportTokenChecker) IsBlockedAccessToken(_ context.Context, _ string) bool {
	return false
}

// TestExportCommon_AuthorizeAndPlatformGate 验证导出路由的鉴权闸与平台管理员闸：
// 缺 Bearer 前缀 / 缺 token / 校验失败均 401；viewer 上下文按操作人租户重建；
// 非平台管理员过不了 requireExportPlatformAdmin。
func TestExportCommon_AuthorizeAndPlatformGate(t *testing.T) {
	newChecker := func(valid bool, payload *authenticationV1.UserTokenPayload) auth.AccessTokenChecker {
		return &fakeExportTokenChecker{valid: valid, payload: payload}
	}

	// 缺 Authorization 头
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/export", nil)
	_, _, ok := authorizeExportRequest(rec, req, newChecker(true, nil))
	require.False(t, ok, "缺 Authorization 头应拒绝")
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	// 有头但无 Bearer 前缀（token == authzHeader 分支）
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/export", nil)
	req.Header.Set("Authorization", "raw-token-without-bearer")
	_, _, ok = authorizeExportRequest(rec, req, newChecker(true, nil))
	require.False(t, ok, "非 Bearer 形态的令牌应拒绝")
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	// Bearer 形态但校验失败
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/export", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	_, _, ok = authorizeExportRequest(rec, req, newChecker(false, nil))
	require.False(t, ok, "校验失败的令牌应拒绝")
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	// 平台管理员：viewer 重建为系统上下文（租户 0）
	platformPayload := &authenticationV1.UserTokenPayload{UserId: 1, IsPlatformAdmin: trans.Ptr(true)}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/export", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	ctx, operator, ok := authorizeExportRequest(rec, req, newChecker(true, platformPayload))
	require.True(t, ok, "平台管理员的合法令牌应放行")
	require.Equal(t, uint32(1), operator.GetUserId())
	vc, vcOK := viewer.FromContext(ctx)
	require.True(t, vcOK, "应重建 viewer 上下文（ent 租户收窄的种子）")
	require.Equal(t, uint64(0), vc.TenantID(), "平台管理员应落系统视图（租户 0）")

	// 租户用户：viewer 重建为用户上下文（租户号与操作人一致）
	tenantID := uint32(7)
	tenantPayload := &authenticationV1.UserTokenPayload{UserId: 9, TenantId: &tenantID}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/export", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	ctx, _, ok = authorizeExportRequest(rec, req, newChecker(true, tenantPayload))
	require.True(t, ok)
	vc, vcOK = viewer.FromContext(ctx)
	require.True(t, vcOK)
	require.Equal(t, uint64(7), vc.TenantID(), "viewer 租户号必须与操作人一致：repo 收窄全靠它")

	// 平台管理员闸
	rec = httptest.NewRecorder()
	require.False(t, requireExportPlatformAdmin(rec, tenantPayload), "租户用户过不了平台闸")
	require.Equal(t, http.StatusForbidden, rec.Code)
	rec = httptest.NewRecorder()
	require.True(t, requireExportPlatformAdmin(rec, platformPayload), "平台管理员应放行")
}

// TestExportCommon_ParseExportRequest 验证导出参数解析：
// format 缺省 xlsx、非法值 400；maxRows 只能在硬顶内往下调；query 原样透传。
func TestExportCommon_ParseExportRequest(t *testing.T) {
	// 缺省 format=xlsx、无 maxRows 用硬顶、无 query 不带过滤
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/export", nil)
	format, paging, ok := parseExportRequest(rec, req, 100)
	require.True(t, ok)
	require.Equal(t, "xlsx", format, "缺省格式应为 xlsx")
	require.NotNil(t, paging.GetNoPaging())
	require.True(t, paging.GetNoPaging(), "导出应关分页（一次拉全量）")
	require.Equal(t, uint32(100), paging.GetLimit(), "maxRows 缺省应取硬顶")
	require.Nil(t, paging.GetFilteringType(), "无 query 时不应带过滤")

	// csv 合法
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/export?format=csv", nil)
	format, _, ok = parseExportRequest(rec, req, 100)
	require.True(t, ok)
	require.Equal(t, "csv", format)

	// 大写 FORMAT 归一
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/export?format=CSV", nil)
	format, _, ok = parseExportRequest(rec, req, 100)
	require.True(t, ok)
	require.Equal(t, "csv", format, "format 应大小写归一")

	// 非法格式 400
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/export?format=pdf", nil)
	_, _, ok = parseExportRequest(rec, req, 100)
	require.False(t, ok, "非法格式应拒绝")
	require.Equal(t, http.StatusBadRequest, rec.Code)

	// maxRows 往下调生效
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/export?maxRows=5", nil)
	_, paging, ok = parseExportRequest(rec, req, 100)
	require.True(t, ok)
	require.Equal(t, uint32(5), paging.GetLimit(), "低于硬顶的 maxRows 应生效")

	// maxRows 超硬顶：忽略（硬顶是资源的上限，客户端只能往下调）
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/export?maxRows=500", nil)
	_, paging, ok = parseExportRequest(rec, req, 100)
	require.True(t, ok)
	require.Equal(t, uint32(100), paging.GetLimit(), "超硬顶的 maxRows 不得放大导出量")

	// maxRows 垃圾值：忽略
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/export?maxRows=abc", nil)
	_, paging, ok = parseExportRequest(rec, req, 100)
	require.True(t, ok)
	require.Equal(t, uint32(100), paging.GetLimit())

	// query 原样透传（导出的就是当前搜索看到的）
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, `/export?query=%7B%22action__contains%22%3A%22DELETE%22%7D`, nil)
	_, paging, ok = parseExportRequest(rec, req, 100)
	require.True(t, ok)
	queryFilter, filterOK := paging.GetFilteringType().(*paginationV1.PagingRequest_Query)
	require.True(t, filterOK, "query 应装配为 Query 过滤")
	require.Contains(t, queryFilter.Query, "action__contains", "query 应原样透传给 repo")
}

// TestExportCommon_WriteCSV 验证手写 CSV 的 RFC 4180 口径：
// UTF-8 BOM 打头（Excel 中文识别）、CRLF 行分隔、含逗号/引号/换行的字段加引号转义。
func TestExportCommon_WriteCSV(t *testing.T) {
	rec := httptest.NewRecorder()
	err := writeCSV(rec,
		[]string{"操作人", "action"},
		[][]string{
			{"甲", "DELETE"},
			{"乙,\"注\"", "含,逗号"},
			{"多\n行", "普"},
		},
	)
	require.NoError(t, err)

	body := rec.Body.String()
	require.True(t, strings.HasPrefix(body, "\xEF\xBB\xBF"), "必须以 UTF-8 BOM 打头（Excel 中文识别）")
	require.Contains(t, body, "操作人,action")
	require.Contains(t, body, "\"乙,\"\"注\"\"\"", "含引号/逗号的字段应加引号并双写引号转义")
	require.Contains(t, body, "\"多\n行\"", "含换行的字段应加引号")
	require.NotContains(t, body, "含,逗号\n", "未转义的裸逗号字段不得出现")
	// 行分隔为 CRLF
	require.Contains(t, body, "DELETE\r\n\"乙", "行分隔应为 CRLF")
}

// TestExportCommon_BuildXLSX 验证 XLSX 产物可被 excelize 原样读回：
// 指定表名、表头与行数据逐格相等；空行集只写表头。
func TestExportCommon_BuildXLSX(t *testing.T) {
	headers := []string{"列一", "列二"}
	rows := [][]string{{"甲", "1"}, {"乙", "2"}}

	data, err := buildXLSX("审计导出", headers, rows)
	require.NoError(t, err, "构建 XLSX 应成功")
	require.NotEmpty(t, data)
	require.True(t, strings.HasPrefix(string(data[:2]), "PK"), "xlsx 本质是 zip（PK 魔头）")

	f, err := excelize.OpenReader(strings.NewReader(string(data)))
	require.NoError(t, err, "产物应能被 excelize 读回")
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	require.Contains(t, sheets, "审计导出", "自定义表名应生效")

	gotHeaders, err := f.GetRows("审计导出")
	require.NoError(t, err)
	require.Len(t, gotHeaders, 3, "表头 + 2 行")
	require.Equal(t, headers, gotHeaders[0], "表头逐格相等")
	require.Equal(t, []string{"甲", "1"}, gotHeaders[1])
	require.Equal(t, []string{"乙", "2"}, gotHeaders[2])

	// 空行集：只有表头一行
	emptyData, err := buildXLSX("空表", headers, nil)
	require.NoError(t, err)
	emptyF, err := excelize.OpenReader(strings.NewReader(string(emptyData)))
	require.NoError(t, err)
	defer func() { _ = emptyF.Close() }()
	emptyRows, err := emptyF.GetRows("空表")
	require.NoError(t, err)
	require.Equal(t, [][]string{headers}, emptyRows, "空行集应只有表头")
}

// TestExportCommon_FmtTimeAndBoolStr 验证单元格取值器的 nil 安全与格式：
// 时间 nil 回空串、格式固定到秒（经 TimestamppbToTime → AsTime，为 UTC 挂钟）；
// 布尔 nil 回空串（未记录 ≠ false）。
func TestExportCommon_FmtTimeAndBoolStr(t *testing.T) {
	require.Empty(t, fmtTime(nil), "nil 时间应回空串而不是零值时间")

	ts := timestamppb.New(time.Unix(1773456605, 0))
	require.Equal(t, time.Unix(1773456605, 0).UTC().Format("2006-01-02 15:04:05"), fmtTime(ts),
		"时间格式应固定为 2006-01-02 15:04:05（AsTime 为 UTC 口径）")

	require.Empty(t, boolStr(nil), "nil 布尔回空串（未记录不得显示为 false）")
	require.Equal(t, "true", boolStr(trans.Ptr(true)))
	require.Equal(t, "false", boolStr(trans.Ptr(false)))
}

// TestExportCommon_EmitExportFile 验证产物下发：
// csv 回 text/csv 与 attachment 文件名；xlsx 回 sheet Content-Type 且 body 可解析。
func TestExportCommon_EmitExportFile(t *testing.T) {
	log := newExportNopLogger()
	ctx := context.Background()
	headers := []string{"h1", "h2"}
	rows := [][]string{{"a", "b"}}

	// CSV
	rec := httptest.NewRecorder()
	emitExportFile(rec, log, ctx, "test", "审计", "Sheet", headers, rows, "csv")
	require.Equal(t, "text/csv; charset=utf-8", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Header().Get("Content-Disposition"), `filename="审计-`)
	require.Contains(t, rec.Header().Get("Content-Disposition"), `.csv"`)
	require.True(t, strings.HasPrefix(rec.Body.String(), "\xEF\xBB\xBF"), "csv body 应带 BOM")

	// XLSX（缺省分支）
	rec = httptest.NewRecorder()
	emitExportFile(rec, log, ctx, "test", "审计", "Sheet", headers, rows, "xlsx")
	require.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Header().Get("Content-Disposition"), `.xlsx"`)
	require.NotEmpty(t, rec.Body.Bytes(), "xlsx body 不应为空")
	_, err := excelize.OpenReader(strings.NewReader(rec.Body.String()))
	require.NoError(t, err, "下发 body 应能被 excelize 读回")
}

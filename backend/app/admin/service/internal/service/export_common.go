package service

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/logger"
	"github.com/xuri/excelize/v2"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	"github.com/tx7do/go-crud/viewer"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"go-wind-admin/pkg/middleware/auth"
)

// exportColumn 一列的定义：表头 + 从行 DTO 取值的取值器（返回已格式化的字符串）。
// 各导出资源（审计五类 / AI 用量流水 / 通知投递台账）共用此形态。
type exportColumn[T any] struct {
	header string
	value  func(row T) string
}

func colHeaders[T any](cols []exportColumn[T]) []string {
	headers := make([]string, 0, len(cols))
	for _, c := range cols {
		headers = append(headers, c.header)
	}
	return headers
}

func colValues[T any](cols []exportColumn[*T], row *T) []string {
	values := make([]string, 0, len(cols))
	for _, c := range cols {
		values = append(values, c.value(row))
	}
	return values
}

func writeCSV(w http.ResponseWriter, headers []string, rows [][]string) error {
	// 手写 CSV（RFC 4180）：含引号/逗号/换行的字段加引号转义；UTF-8 BOM 让 Excel 正确识别中文
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	escape := func(v string) string {
		if strings.ContainsAny(v, ",\"\n\r") {
			return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
		}
		return v
	}
	lines := make([]string, 0, len(rows)+1)
	lines = append(lines, joinCSV(headers, escape))
	for _, row := range rows {
		lines = append(lines, joinCSV(row, escape))
	}
	_, err := w.Write([]byte(strings.Join(lines, "\r\n")))
	return err
}

func joinCSV(fields []string, escape func(string) string) string {
	out := make([]byte, 0, 32*len(fields))
	for i, f := range fields {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, escape(f)...)
	}
	return string(out)
}

func buildXLSX(sheet string, headers []string, rows [][]string) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	// NewFile 自带 "Sheet1"，直接改名成目标表名即可；先 NewSheet(sheet) 再改名
	// 会得到两张同名工作表，随后的按名写入会落到解析不唯一的位置——导出产物
	// 变成"表头行数都对不上"的空簿（实测 2026-11）。
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := f.SetSheetRow(sheet, cell, &headers); err != nil {
		return nil, err
	}
	// SetSheetRow 一次只写一行（slice 元素=单元格）：把整个 rows 矩阵塞给它
	// 会把每行当成一个单元格序列化成 "[a b]" 字符串——导出文件全部数据挤在一行乱码里。
	for i, row := range rows {
		cell, _ = excelize.CoordinatesToCellName(1, i+2)
		if err := f.SetSheetRow(sheet, cell, &row); err != nil {
			return nil, err
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func fmtTime(ts *timestamppb.Timestamp) string {
	t := timeutil.TimestamppbToTime(ts)
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func boolStr(v *bool) string {
	if v == nil {
		return ""
	}
	return strconv.FormatBool(*v)
}

// ==== 手动导出路由的公共样板 ====
//
// :export 手动路由不穿 kratos middleware（proto 生成 handler 才走中间件链），
// 因此 auth 中间件的一套（token 校验、操作人注入、ent viewer 注入）对它全部缺席。
// 各导出 handler 在此补齐：token 自校验 + 操作人注入 + viewer 重建。

// authorizeExportRequest 校验 Bearer token 并重建 auth/ent 两层上下文。
// 返回 nil 表示已写好 401 响应，调用方直接 return。
//
// viewer 重建与 asynq handler 的惯例同款：平台管理员 SystemContext，
// 租户用户 UserContext（后续 repo 查询由 mixin 谓词自动收窄到本租户）。
// 是否放行租户用户由各 handler 在拿到 operator 后自行定夺（审计/台账=平台闸）。
func authorizeExportRequest(
	w http.ResponseWriter,
	r *http.Request,
	tokenChecker auth.AccessTokenChecker,
) (context.Context, *authenticationV1.UserTokenPayload, bool) {
	authzHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authzHeader, "Bearer ")
	if token == "" || token == authzHeader {
		http.Error(w, "unauthorized: bearer token required", http.StatusUnauthorized)
		return nil, nil, false
	}
	valid, operator := tokenChecker.IsValidAccessToken(r.Context(), token, false)
	if !valid || operator == nil {
		http.Error(w, "unauthorized: invalid token", http.StatusUnauthorized)
		return nil, nil, false
	}
	ctx := auth.NewContext(r.Context(), operator)

	if operator.GetTenantId() == 0 {
		ctx = viewer.WithSystemContext(ctx)
	} else {
		ctx = viewer.WithContext(ctx, viewer.NewUserContext(
			uint64(operator.GetUserId()), uint64(operator.GetTenantId()), 0, "", nil,
		))
	}
	return ctx, operator, true
}

// requireExportPlatformAdmin 平台管理员闸（语义对齐 requirePlatformAdmin，
// 手动路由拿不到 service 层 ctx，须在 operator 已解析处就地判定）。
func requireExportPlatformAdmin(w http.ResponseWriter, operator *authenticationV1.UserTokenPayload) bool {
	if !operator.GetIsPlatformAdmin() {
		http.Error(w, "forbidden: platform admin only", http.StatusForbidden)
		return false
	}
	return true
}

// requireExportTenantModule 手动导出路由的租户闸（P1 修复补齐）。
//
// :export 手动路由游离在 kratos 中间件链之外，此前套餐模块白名单/租户状态/
// 到期处置/API_CALL 配额四段语义全部失效（套餐不含 AI 模块的租户也能导出）。
// 手动路由自身不在 Api 表（无法以自身 path 判定模块归属），故按其同源列表
// 端点的 (path,method) 复用 HTTP 中间件的 TenantAccessChecker 判定——该行在
// Api 表内，完整带出四段语义。
func requireExportTenantModule(
	w http.ResponseWriter,
	checker auth.TenantAccessChecker,
	ctx context.Context,
	operator *authenticationV1.UserTokenPayload,
	listPath, method string,
) bool {
	if checker == nil {
		// checker 未装配：fail-closed（与中间件"未归类即拒"口径一致）
		http.Error(w, "forbidden: tenant access checker unavailable", http.StatusForbidden)
		return false
	}
	if err := checker.CheckTenantAccess(ctx, operator.GetTenantId(), listPath, method); err != nil {
		http.Error(w, "forbidden: tenant access denied", http.StatusForbidden)
		return false
	}
	return true
}

// parseExportRequest 解析导出参数：format（xlsx/csv，缺省 xlsx）与行数上限
// （cap 为该资源的硬顶，maxRows 只能往下调）。
// query JSON（列表页同源条件）原样透传给 repo List，"导出的就是当前搜索看到的"。
func parseExportRequest(w http.ResponseWriter, r *http.Request, cap int) (string, *paginationV1.PagingRequest, bool) {
	q := r.URL.Query()
	format := strings.ToLower(q.Get("format"))
	if format == "" {
		format = "xlsx"
	}
	if format != "xlsx" && format != "csv" {
		http.Error(w, "format must be xlsx or csv", http.StatusBadRequest)
		return "", nil, false
	}
	maxRows := cap
	if v := q.Get("maxRows"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed < cap {
			maxRows = parsed
		}
	}

	req := &paginationV1.PagingRequest{
		NoPaging: trans.Ptr(true),
		Limit:    trans.Ptr(uint32(maxRows)),
	}
	if query := strings.TrimSpace(q.Get("query")); query != "" {
		req.FilteringType = &paginationV1.PagingRequest_Query{Query: query}
	}
	return format, req, true
}

// emitExportFile 把列化的数据按 format 写成下载响应。
// 查询/构建的原始错误由调用方留日志（仓铁律：不吞错），此处只回笼统文案。
func emitExportFile(
	w http.ResponseWriter,
	log *logger.Helper,
	ctx context.Context,
	tag string,
	fileName string,
	sheet string,
	headers []string,
	rows [][]string,
	format string,
) {
	stamp := time.Now().Format("20060102_150405")

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-%s.csv"`, fileName, stamp))
		if err := writeCSV(w, headers, rows); err != nil {
			log.Errorf(ctx, "export [%s] csv write failed: %s", tag, err.Error())
			return
		}
	default:
		data, err := buildXLSX(sheet, headers, rows)
		if err != nil {
			log.Errorf(ctx, "export [%s] xlsx build failed: %s", tag, err.Error())
			http.Error(w, "build xlsx failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-%s.xlsx"`, fileName, stamp))
		_, _ = w.Write(data)
	}
}

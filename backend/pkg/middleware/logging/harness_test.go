// harness_test.go —— 审计中间件测试的公共假件与进程内 server 挂具。
//
// 为什么需要这层挂具：五个审计中间件的 Handle 均以 kratos 的
// *http.Transport（字段全部未导出、无包外构造器）为入参，包外无法直接构造。
// 这里走 kratos 官方的服务端构造路径：
//
//	khttp.NewServer(khttp.Middleware(Server(...)))  挂上被测中间件
//	srv.Route("/").Handle(...)                      注册与生成代码同形的路由
//	srv.ServeHTTP(recorder, request)                进程内直投，无 socket、无监听
//
// mux 匹配后由 server.filter 构造出携带真实 *http.Request 的 Transport，
// 路由 handler 形态与 protoc 生成的 *_http.pb.go 一致（SetOperation +
// ctx.Middleware），中间件因此按生产语义完整执行。
//
// 桩写入函数（write*Func）经 With* 选项注入，记录每次调用与字段供断言；
// 同时记录写入时 ctx 的审计落库标记（audit.SinkKey，防递归采集）与系统
// viewer 状态——审计写入必须以系统 viewer（绕过租户隔离）+ sink 标记执行，
// 这是审计链路的关键不变量。
package logging

import (
	"context"
	"errors"
	"io"
	"math/big"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	crudviewer "github.com/tx7do/go-crud/viewer"
	authn "github.com/tx7do/kratos-authn/engine"
	"google.golang.org/protobuf/types/known/timestamppb"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
	"go-wind-admin/pkg/audit"
	pkgjwt "go-wind-admin/pkg/jwt"
)

// ---------------------------------------------------------------------------
// 写入桩：记录每条被落库的审计记录与调用上下文
// ---------------------------------------------------------------------------

// auditCallMeta 记录单次写入发生时的上下文特征。
type auditCallMeta struct {
	Sinking      bool // ctx 是否带审计落库标记（防递归采集）
	SystemViewer bool // ctx 是否为系统 viewer（审计写入须绕过租户隔离）
}

// auditCapture 以桩 write 函数身份记录每一条被落库的审计记录及其调用上下文，
// 供测试断言字段来源映射与写入链路不变量。
type auditCapture struct {
	api            []*auditV1.ApiAuditLog
	login          []*auditV1.LoginAuditLog
	operation      []*auditV1.OperationAuditLog
	permission     []*auditV1.PermissionAuditLog
	dataAccess     []*auditV1.DataAccessAuditLog
	apiMeta        []auditCallMeta
	loginMeta      []auditCallMeta
	operationMeta  []auditCallMeta
	permissionMeta []auditCallMeta
	dataAccessMeta []auditCallMeta
}

// metaOf 提取写入时刻 ctx 的 sink 标记与系统 viewer 状态。
func (c *auditCapture) metaOf(ctx context.Context) auditCallMeta {
	v, _ := crudviewer.FromContext(ctx)
	return auditCallMeta{
		Sinking:      audit.IsSinking(ctx),
		SystemViewer: v != nil && v.IsSystemContext(),
	}
}

// total 返回全部桩写入总数，用于"零写入"断言。
func (c *auditCapture) total() int {
	return len(c.api) + len(c.login) + len(c.operation) + len(c.permission) + len(c.dataAccess)
}

// ---------------------------------------------------------------------------
// 进程内 server 挂具
// ---------------------------------------------------------------------------

// auditServerEnv 封装一台进程内 kratos http server 与写入桩。
type auditServerEnv struct {
	t       *testing.T
	srv     *khttp.Server
	capture *auditCapture
	// acc 记录业务闭包注入 SQL 事件时的 accumulator 指针，
	// 请求结束后验证其被 data_access 审计清空（防复用 ctx 重复落库）。
	acc *[]audit.AuditEvent
}

// newAuditServer 构造挂具：五个写入桩默认全部注入（后续 opts 可覆盖，
// 例如显式传 WithWriteApiLogFunc(nil) 以验证空函数分支）。
func newAuditServer(t *testing.T, opts ...Option) *auditServerEnv {
	t.Helper()
	env := &auditServerEnv{t: t, capture: &auditCapture{}}
	all := []Option{
		WithWriteApiLogFunc(func(ctx context.Context, d *auditV1.ApiAuditLog) error {
			env.capture.api = append(env.capture.api, d)
			env.capture.apiMeta = append(env.capture.apiMeta, env.capture.metaOf(ctx))
			return nil
		}),
		WithWriteLoginLogFunc(func(ctx context.Context, d *auditV1.LoginAuditLog) error {
			env.capture.login = append(env.capture.login, d)
			env.capture.loginMeta = append(env.capture.loginMeta, env.capture.metaOf(ctx))
			return nil
		}),
		WithWriteOperationAuditLogFunc(func(ctx context.Context, d *auditV1.OperationAuditLog) error {
			env.capture.operation = append(env.capture.operation, d)
			env.capture.operationMeta = append(env.capture.operationMeta, env.capture.metaOf(ctx))
			return nil
		}),
		WithWritePermissionAuditLogFunc(func(ctx context.Context, d *auditV1.PermissionAuditLog) error {
			env.capture.permission = append(env.capture.permission, d)
			env.capture.permissionMeta = append(env.capture.permissionMeta, env.capture.metaOf(ctx))
			return nil
		}),
		WithWriteDataAccessAuditLogFunc(func(ctx context.Context, d *auditV1.DataAccessAuditLog) error {
			env.capture.dataAccess = append(env.capture.dataAccess, d)
			env.capture.dataAccessMeta = append(env.capture.dataAccessMeta, env.capture.metaOf(ctx))
			return nil
		}),
	}
	all = append(all, opts...)
	env.srv = khttp.NewServer(khttp.Middleware(Server(all...)))
	router := env.srv.Route("/")
	handler := env.chainHandler()
	// 四种写方法 × 两种路径形态（带/不带数字段）注册同一 handler，
	// 供方法分支与路径数字段（ResourceId/TargetId 来源）断言复用。
	for _, m := range []string{
		nethttp.MethodPost, nethttp.MethodGet,
		nethttp.MethodPut, nethttp.MethodDelete,
	} {
		router.Handle(m, "/case", handler)
		router.Handle(m, "/case/{id}", handler)
	}
	return env
}

// chainHandler 返回与 protoc 生成代码同形的路由 handler：
// 先按 X-Test-Operation 设置 operation，再把业务闭包挂进中间件链。
// 业务闭包按 X-Test-Error 返回错误（kratos 错误 / 普通错误），
// 按 X-Test-Data-Access 向 accumulator 注入 SQL 事件（模拟 driver wrapper
// 的采集行为）；X-Test-Reply-Username 模拟 handler 通过响应头回传
// 挑战上下文用户名（MFA 免鉴权流程语义，见 MfaService.VerifyMFAChallenge）。
func (e *auditServerEnv) chainHandler() func(ctx khttp.Context) error {
	return func(ctx khttp.Context) error {
		req := ctx.Request()
		if op := req.Header.Get("X-Test-Operation"); op != "" {
			khttp.SetOperation(ctx, op)
		}
		if v := req.Header.Get("X-Test-Reply-Username"); v != "" {
			ctx.Response().Header().Set("X-Audit-Username", v)
		}
		var innerErr error
		switch req.Header.Get("X-Test-Error") {
		case "kratos":
			innerErr = kerrors.New(403, "TEST_FORBIDDEN", "forbidden for test")
		case "plain":
			innerErr = errors.New("plain test error")
		}
		da := req.Header.Get("X-Test-Data-Access")
		h := ctx.Middleware(func(c context.Context, reqIn interface{}) (interface{}, error) {
			if da == "all" || da == "empty" {
				if acc, ok := audit.FromContext(c); ok {
					e.acc = acc
					if da == "all" {
						*acc = append(*acc, testDAEvents()...)
					}
				}
			}
			return nil, innerErr
		})
		_, _ = h(ctx, nil)
		return nil
	}
}

// fire 向进程内 server 直投一条请求（不走 socket），返回 recorder。
// headers 会原样设到请求头；remoteAddr 留空则用 httptest 默认（公网 192.0.2.1）。
func (e *auditServerEnv) fire(method, path string, headers map[string]string, body, remoteAddr string) *httptest.ResponseRecorder {
	e.t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	rec := httptest.NewRecorder()
	e.srv.ServeHTTP(rec, req)
	return rec
}

// ---------------------------------------------------------------------------
// 测试数据构造
// ---------------------------------------------------------------------------

// testDAEvents 构造四类典型 SQL 事件：读用户表、写角色表、JOIN 双表、
// 无表引用；覆盖 TableName 多表斜线连接、首表数据分类与 AffectedRows
// 取值边界（-1 不落库 / 0 与正数落库）。
func testDAEvents() []audit.AuditEvent {
	return []audit.AuditEvent{
		{
			SqlText:   "SELECT id, username FROM sys_users WHERE id = $1",
			SqlDigest: "digest-select-users", Latency: 12, Dialect: "postgres",
			AffectedRows: -1, DataMasked: true, MaskingRules: "rule-a",
		},
		{
			SqlText:   "INSERT INTO sys_roles (name) VALUES ('x')",
			SqlDigest: "digest-insert-roles", Latency: 34, Dialect: "mysql",
			AffectedRows: 2, DataMasked: false, MaskingRules: "",
		},
		{
			SqlText:   "SELECT * FROM sys_users a JOIN sys_plan_quotas b ON a.id = b.user_id",
			SqlDigest: "digest-join", Latency: 56, Dialect: "postgres",
			AffectedRows: 0, DataMasked: false, MaskingRules: "",
		},
		{
			SqlText:   "SELECT 1",
			SqlDigest: "digest-constant", Latency: 7, Dialect: "postgres",
			AffectedRows: 0, DataMasked: false, MaskingRules: "",
		},
	}
}

// mintTestToken 铸造测试 JWT。解析侧（jwtutil.ParseJWTPayload）不校验签名，
// 任意 HS256 密钥签出的载荷即可映射出 UserTokenPayload
// （sub→Username、uid→UserId、tid→TenantId、cid→ClientId），
// 用于断言审计记录中用户身份字段的令牌来源映射。
func mintTestToken(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		authn.ClaimFieldSubject:   "alice",
		pkgjwt.ClaimFieldUserID:   float64(42),
		pkgjwt.ClaimFieldTenantID: float64(7),
		pkgjwt.ClaimFieldClientID: "test-client-id",
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-signing-secret"))
	require.NoError(t, err)
	return tok
}

// mintTestTokenWithoutSubject 铸造无 sub 声明的令牌：载荷侧 GetSubject
// 报错走日志分支，Username 不落库而数字身份照常映射。
func mintTestTokenWithoutSubject(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		pkgjwt.ClaimFieldUserID:   float64(42),
		pkgjwt.ClaimFieldTenantID: float64(7),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-signing-secret"))
	require.NoError(t, err)
	return tok
}

// mintTestTokenWithBadRoleClaim 铸造角色声明类型非法的令牌：载荷构造器
// 在角色声明类型不符时报错返回，覆盖令牌身份提取的错误分支。
func mintTestTokenWithBadRoleClaim(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		pkgjwt.ClaimFieldRoleCodes: "not-an-array",
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-signing-secret"))
	require.NoError(t, err)
	return tok
}

// ---------------------------------------------------------------------------
// 签名/哈希形状断言
// ---------------------------------------------------------------------------

// parseDERSig 按 encodeDER 的格式（0x30 总长 0x02 rLen r 0x02 sLen s）拆出
// (r, s)，供 ecdsa.Verify 回验签名。
func parseDERSig(t *testing.T, der []byte) (r, s *big.Int) {
	t.Helper()
	require.NotNil(t, der)
	require.GreaterOrEqual(t, len(der), 8)
	require.Equal(t, byte(0x30), der[0])
	require.Equal(t, byte(len(der)-2), der[1])
	idx := 2
	require.Equal(t, byte(0x02), der[idx])
	rLen := int(der[idx+1])
	require.Positive(t, rLen)
	r = new(big.Int).SetBytes(der[idx+2 : idx+2+rLen])
	idx += 2 + rLen
	require.Equal(t, byte(0x02), der[idx])
	sLen := int(der[idx+1])
	require.Positive(t, sLen)
	s = new(big.Int).SetBytes(der[idx+2 : idx+2+sLen])
	idx += 2 + sLen
	require.Equal(t, len(der), idx)
	require.Positive(t, r.Sign())
	require.Positive(t, s.Sign())
	return r, s
}

// requireDERSig 断言签名为结构合法的 DER 序列（可拆出正的 r/s）。
// 声明式包装 parseDERSig，供仅关心形状的断言使用。
func requireDERSig(t *testing.T, sig []byte) {
	t.Helper()
	r, s := parseDERSig(t, sig)
	require.NotNil(t, r)
	require.NotNil(t, s)
}

// requireLogHashHex 断言 LogHash 为 64 位小写十六进制（SHA256）。
func requireLogHashHex(t *testing.T, hash string) {
	t.Helper()
	require.Len(t, hash, 64)
	for i := 0; i < len(hash); i++ {
		c := hash[i]
		require.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"log_hash 必须为十六进制字符: %q", hash)
	}
}

// requireTimestampNearNow 断言时间戳落在当前时间 ±2 分钟内。
func requireTimestampNearNow(t *testing.T, ts *timestamppb.Timestamp) {
	t.Helper()
	require.NotNil(t, ts)
	now := time.Now().Unix()
	require.GreaterOrEqual(t, ts.Seconds, now-120)
	require.LessOrEqual(t, ts.Seconds, now+120)
}

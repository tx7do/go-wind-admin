// utils_extra_test.go —— utils.go 尚未覆盖的取值与纯函数测试
// （utils_test.go 仅覆盖 getIPFromRemoteAddr，本文件覆盖其余全部）。
//
// 覆盖内容：
//  1. getClientRealIP：X-Forwarded-For 多级代理链（含非法项跳过、空白容忍）
//     → X-Real-IP → RemoteAddr 的优先级链；
//  2. getRequestId：X-Request-ID → X-Correlation-ID → x-fc-request-id 的
//     优先级链与无头时的 GUID 生成；
//  3. getClientID：X-Client-ID 头与令牌 ClientId 的来源优先级；
//  4. getStatusCode：nil 错误、kratos 错误（code/reason 透传、<400 记成功）、
//     普通错误（500/空 reason/失败）；
//  5. parseUsernameFromBytes / stripLineBreaks / extractUsernameFromRequest：
//     JSON 与表单体提取、CR/LF 剥离（防日志行注入）、未找到的报错路径、
//     提取后请求体的可重读性；
//  6. clientIpToLocation / fillGeoLocation：私网 IP（局域网归一）、
//     非法 IP（空结果）；
//  7. fillDeviceInfo 的空 Transport 来源（全空设备信息）；
//  8. isPrivateIP：常见私网/链路本地/环回段与 IPv6 ULA 的判定；
//  9. detectPlatformFromUA：原生 App / 桌面混合应用 / 浏览器的启发式分类全分支；
//  10. generateECDSAKeyPair 与 encodeDER（含高位字节 0x00 前缀与零值分支）。
package logging

import (
	"errors"
	"io"
	"math/big"
	nethttp "net/http"
	"strings"
	"testing"

	"crypto/elliptic"
	kerrors "github.com/go-kratos/kratos/v2/errors"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

// newHeaderOnlyRequest 构造仅带指定头的轻量请求（无 body）。
func newHeaderOnlyRequest(headers map[string]string) *nethttp.Request {
	req, _ := nethttp.NewRequest(nethttp.MethodGet, "/probe", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

// TestGetClientRealIP 表驱动验证客户端真实 IP 的来源优先级：
// X-Forwarded-For 首个可解析项（逗号分隔、容忍空白、跳过非法项）→
// X-Real-IP → RemoteAddr（host:port 拆解）。所有来源不可用时归一空串。
func TestGetClientRealIP(t *testing.T) {
	cases := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		want       string
	}{
		{"单级XFF", "1.2.3.4", "", "", "1.2.3.4"},
		{"多级XFF取首个", "1.2.3.4, 5.6.7.8", "", "", "1.2.3.4"},
		{"XFF空白容忍", "  5.6.7.8 , 1.2.3.4", "", "", "5.6.7.8"},
		{"XFF全非法回落XRI", "garbage", "9.9.9.9", "", "9.9.9.9"},
		{"XFF与XRI全非法回落RemoteAddr", "garbage", "alsogarbage", "203.0.113.5:8080", "203.0.113.5"},
		{"仅RemoteAddr", "", "", "203.0.113.5:8080", "203.0.113.5"},
		{"全部缺失", "", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := newHeaderOnlyRequest(nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xff != "" {
				req.Header.Set(HeaderKeyXForwardedFor, tc.xff)
			}
			if tc.xri != "" {
				req.Header.Set(HeaderKeyXRealIP, tc.xri)
			}
			assert.Equal(t, tc.want, getClientRealIP(req))
		})
	}
}

// TestGetClientRealIPNilRequest 验证 nil 请求安全归一为空串。
func TestGetClientRealIPNilRequest(t *testing.T) {
	assert.Empty(t, getClientRealIP(nil))
}

// TestGetRequestId 表驱动验证请求 ID 的来源优先级：
// X-Request-ID → X-Correlation-ID → x-fc-request-id；
// 全部缺失时生成 GUID（保证可追踪而非空串）。
func TestGetRequestId(t *testing.T) {
	t.Run("三级头优先级", func(t *testing.T) {
		req := newHeaderOnlyRequest(map[string]string{
			HeaderKeyXRequestID:     "id-req",
			HeaderKeyXCorrelationID: "id-corr",
			HeaderKeyXFcRequestID:   "id-fc",
		})
		assert.Equal(t, "id-req", getRequestId(req))

		req = newHeaderOnlyRequest(map[string]string{
			HeaderKeyXCorrelationID: "id-corr",
			HeaderKeyXFcRequestID:   "id-fc",
		})
		assert.Equal(t, "id-corr", getRequestId(req))

		req = newHeaderOnlyRequest(map[string]string{
			HeaderKeyXFcRequestID: "id-fc",
		})
		assert.Equal(t, "id-fc", getRequestId(req))
	})

	t.Run("无头生成GUID", func(t *testing.T) {
		req := newHeaderOnlyRequest(nil)
		assert.NotEmpty(t, getRequestId(req), "无请求 ID 头时必须生成 GUID")
	})

	t.Run("nil请求", func(t *testing.T) {
		assert.Empty(t, getRequestId(nil))
	})
}

// TestGetClientID 表驱动验证客户端 ID 的来源：
// X-Client-ID 头优先，其次令牌 ClientId；nil 请求归一空串。
func TestGetClientID(t *testing.T) {
	ut := &authenticationV1.UserTokenPayload{ClientId: trans.Ptr("cli-token")}

	t.Run("nil请求", func(t *testing.T) {
		assert.Empty(t, getClientID(nil, ut))
	})

	t.Run("无来源", func(t *testing.T) {
		assert.Empty(t, getClientID(newHeaderOnlyRequest(nil), nil))
	})

	t.Run("头优先于令牌", func(t *testing.T) {
		req := newHeaderOnlyRequest(map[string]string{HeaderKeyXClientIP: "cli-hdr"})
		assert.Equal(t, "cli-hdr", getClientID(req, ut))
	})

	t.Run("令牌来源", func(t *testing.T) {
		assert.Equal(t, "cli-token", getClientID(newHeaderOnlyRequest(nil), ut))
	})
}

// TestGetStatusCode 表驱动验证错误到状态三元组（code/reason/success）的映射：
// nil → 200/空/成功；kratos 错误透传 code 与 reason、<400 记成功；
// 普通错误按 500/空 reason/失败处理。
func TestGetStatusCode(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantCode   uint32
		wantReason string
		wantOK     bool
	}{
		{"nil错误", nil, 200, "", true},
		{"kratos客户端错误", kerrors.New(403, "TEST_FORBIDDEN", "forbidden for test"), 403, "TEST_FORBIDDEN", false},
		{"kratos重定向记成功", kerrors.New(302, "TEST_FOUND", "moved"), 302, "TEST_FOUND", true},
		{"kratos信息类记成功", kerrors.New(199, "TEST_INFO", "info"), 199, "TEST_INFO", true},
		{"普通错误", errors.New("plain boom"), 500, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, reason, ok := getStatusCode(tc.err)
			assert.Equal(t, tc.wantCode, code)
			assert.Equal(t, tc.wantReason, reason)
			assert.Equal(t, tc.wantOK, ok)
		})
	}
}

// TestParseUsernameFromBytes 表驱动验证用户名提取的两种载体
// （JSON 引号形式含空白容忍、表单编码）与 CR/LF 剥离。
func TestParseUsernameFromBytes(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantName string
		wantErr  bool
	}{
		{"JSON载体", `{"username":"bob"}`, "bob", false},
		{"JSON空白容忍", "{\"username\"  :  \"bob\"}", "bob", false},
		{"表单载体", "username=bob&password=x", "bob", false},
		{"无用户名表单", "a=b", "", true},
		{"非载体文本", "not json at all", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name, err := parseUsernameFromBytes([]byte(tc.body))
			if tc.wantErr {
				assert.Error(t, err, "未找到用户名必须显式报错而非静默空串")
				assert.Empty(t, name)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantName, name)
			}
		})
	}
}

// TestStripLineBreaks 验证 CR/LF 全量剥离（阻断日志行注入）。
func TestStripLineBreaks(t *testing.T) {
	assert.Equal(t, "ab", stripLineBreaks("a\r\nb"))
	assert.Equal(t, "abb", stripLineBreaks("a\r\rb\nb"))
	assert.Empty(t, stripLineBreaks("\r\n"))
	assert.Empty(t, stripLineBreaks(""))
}

// TestExtractUsernameFromRequest 表驱动验证请求级用户名提取与
// 提取后请求体的可重读性（读尽后必须复位，避免影响后续业务处理）。
func TestExtractUsernameFromRequest(t *testing.T) {
	newBodyReq := func(body string) *nethttp.Request {
		req, _ := nethttp.NewRequest(nethttp.MethodPost, "/probe", strings.NewReader(body))
		return req
	}

	t.Run("JSON载体并复位body", func(t *testing.T) {
		req := newBodyReq(`{"username":"bob","password":"x"}`)
		name, err := extractUsernameFromRequest(req)
		assert.NoError(t, err)
		assert.Equal(t, "bob", name)
		restored, rerr := io.ReadAll(req.Body)
		assert.NoError(t, rerr)
		assert.Equal(t, `{"username":"bob","password":"x"}`, string(restored), "提取后 body 必须可重读")
	})

	t.Run("CR/LF剥离", func(t *testing.T) {
		req := newBodyReq("{\"username\":\"bo\r\nb\"}")
		name, err := extractUsernameFromRequest(req)
		assert.NoError(t, err)
		assert.Equal(t, "bob", name, "提取值必须剥离换行防注入")
	})

	t.Run("表单载体", func(t *testing.T) {
		req := newBodyReq("username=bob&password=x")
		name, err := extractUsernameFromRequest(req)
		assert.NoError(t, err)
		assert.Equal(t, "bob", name)
	})

	t.Run("可解析表单无用户名", func(t *testing.T) {
		req := newBodyReq("a=b")
		name, err := extractUsernameFromRequest(req)
		assert.NoError(t, err)
		assert.Empty(t, name)
	})

	t.Run("无等号文本可解析但无用户名", func(t *testing.T) {
		// 该文本可被 ParseQuery 解析（键无值），按空用户名处理而非报错。
		req := newBodyReq("garbage-body")
		name, err := extractUsernameFromRequest(req)
		assert.NoError(t, err)
		assert.Empty(t, name)
	})

	t.Run("非法转义导致解析失败报错", func(t *testing.T) {
		req := newBodyReq("a=%zz&username=bob")
		name, err := extractUsernameFromRequest(req)
		assert.Error(t, err, "两段解析都失败时必须显式报错")
		assert.Empty(t, name)
	})

	t.Run("nil请求报错", func(t *testing.T) {
		name, err := extractUsernameFromRequest(nil)
		assert.Error(t, err)
		assert.Empty(t, name)
	})

	t.Run("读取body失败透传错误", func(t *testing.T) {
		sentinel := errors.New("read-body-failed")
		req := &nethttp.Request{
			Method: nethttp.MethodPost,
			Body:   &errReadCloser{err: sentinel},
		}
		name, err := extractUsernameFromRequest(req)
		assert.ErrorIs(t, err, sentinel, "读 body 失败必须把原始错误透传出去（不吞错）")
		assert.Empty(t, name)
	})
}

// errReadCloser 恒定返回既定错误的可读体，用于覆盖读失败分支。
type errReadCloser struct{ err error }

func (r *errReadCloser) Read(p []byte) (int, error) { return 0, r.err }
func (r *errReadCloser) Close() error               { return nil }

// TestClientIpToLocation 验证地理解析结果的两个分支：
// 私网 IP → 内建局域网结果；非法 IP → nil。
func TestClientIpToLocation(t *testing.T) {
	res := clientIpToLocation("127.0.0.1")
	require.NotNil(t, res)
	assert.Equal(t, "局域网", res.Country)
	assert.Equal(t, "局域网", res.Province)
	assert.Equal(t, "局域网", res.City)

	assert.Nil(t, clientIpToLocation("not-an-ip"))
}

// TestFillGeoLocation 验证地理位置填充：非法/空 IP 得到全空结构体
// （不 panic、不 nil）；私网 IP 填内建局域网字段。
func TestFillGeoLocation(t *testing.T) {
	for _, ip := range []string{"", "not-an-ip"} {
		info := fillGeoLocation(ip)
		require.NotNil(t, info, "地理位置结构必须非 nil")
		assert.Empty(t, info.GetCountryCode())
		assert.Empty(t, info.GetProvince())
		assert.Empty(t, info.GetCity())
		assert.Empty(t, info.GetIsp())
	}

	info := fillGeoLocation("127.0.0.1")
	require.NotNil(t, info)
	assert.Equal(t, "局域网", info.GetCountryCode())
	assert.Equal(t, "局域网", info.GetProvince())
	assert.Equal(t, "局域网", info.GetCity())
}

// TestFillDeviceInfoEmptyTransport 验证空 Transport（无请求头可读）时
// 设备信息为全空解析：UA 空、设备类型 OTHER、平台 Other、ClientId 空。
func TestFillDeviceInfoEmptyTransport(t *testing.T) {
	info := fillDeviceInfo(&khttp.Transport{}, nil)
	require.NotNil(t, info)
	assert.Empty(t, info.GetUserAgent())
	assert.Empty(t, info.GetClientName())
	assert.Equal(t, auditV1.DeviceInfo_OTHER, info.GetDeviceType())
	assert.Empty(t, info.GetBrowserName())
	assert.Empty(t, info.GetBrowserVersion())
	assert.Empty(t, info.GetOsName())
	assert.Empty(t, info.GetOsVersion())
	assert.Equal(t, PlatformOther, info.GetPlatform())
	assert.Empty(t, info.GetClientId())
}

// TestIsPrivateIP 表驱动验证私网/环回/链路本地/IPv6 ULA 判定与
// 空白容忍；公网、非法串、链路本地之外的 IPv6 归否。
func TestIsPrivateIP(t *testing.T) {
	cases := []struct {
		ip   string
		want bool
	}{
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"172.31.255.254", true},
		{"192.168.1.1", true},
		{"127.0.0.1", true},
		{"169.254.1.1", true},
		{"::1", true},
		{"fc00::1", true},
		{"fd12::1", true},
		{"  10.0.0.1  ", true},
		{"8.8.8.8", false},
		{"172.32.0.1", false},
		{"192.169.1.1", false},
		{"fe80::1", false},
		{"2001:db8::1", false},
		{"garbage", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.ip, func(t *testing.T) {
			assert.Equal(t, tc.want, isPrivateIP(tc.ip))
		})
	}
}

// TestDetectPlatformFromUA 表驱动验证平台启发式分类全分支：
// 原生 Android（okhttp/dalvik/webview 标志、android+包名模式）、原生 iOS
// （iphone/ipad/ipod/cfnetwork/darwin）、桌面混合应用（electron/nwjs/nodejs/
// desktop/appname × 各 OS 关键词与 win/mac/linux 兜底）、
// 浏览器（Mozilla+桌面/移动 OS 且无原生标志）、其余归 Other。
func TestDetectPlatformFromUA(t *testing.T) {
	cases := []struct {
		name string
		ua   string
		want string
	}{
		{"空UA", "", PlatformOther},
		{"okhttp", "okhttp/3.0", PlatformAndroidApp},
		{"dalvik", "dalvik/2.1", PlatformAndroidApp},
		{"wv标志", "Mozilla/5.0 (Linux; U; Android 4.4; x; ; wv) Chrome", PlatformAndroidApp},
		{"紧凑wv标志", "Mozilla/5.0 (Linux; Android 4.4;  ;wv) Chrome", PlatformAndroidApp},
		{"build标志", "Mozilla/5.0 (Linux; Android 4.4; build/Xiaomi) Chrome", PlatformAndroidApp},
		{"android包名模式", "some android client com.foo.bar activity", PlatformAndroidApp},
		{"android含wv", "android webview wv", PlatformAndroidApp},
		{"iphone", "iphone", PlatformiOSApp},
		{"ipad", "ipad", PlatformiOSApp},
		{"ipod", "ipod", PlatformiOSApp},
		{"cfnetwork", "cfnetwork", PlatformiOSApp},
		{"darwin", "darwin", PlatformiOSApp},
		{"cpuiphoneos", "cpu iphone os 17", PlatformiOSApp},
		{"electron加windowsnt", "electron on windows nt", PlatformDesktopWindows},
		{"electron加macintosh", "electron on macintosh", PlatformDesktopMac},
		{"electron加x11", "electron on x11", PlatformDesktopLinux},
		{"electron加debian", "electron on debian", PlatformDesktopLinux},
		{"nwjs加win64", "nwjs with win64", PlatformDesktopWindows},
		{"nodejs加macosx", "node.js mac os x", PlatformDesktopMac},
		{"nodejs加ubuntu", "nodejs ubuntu", PlatformDesktopLinux},
		{"desktop标志加windows", "desktop appname windows", PlatformDesktopWindows},
		{"electron仅win兜底", "electron win", PlatformDesktopWindows},
		{"electron仅mac兜底", "electron mac", PlatformDesktopMac},
		{"electron仅osx兜底", "electron os x", PlatformDesktopMac},
		{"electron仅linux兜底", "electron linux", PlatformDesktopLinux},
		{"electron无OS关键词", "electron", PlatformOther},
		{"桌面浏览器windows", "Mozilla/5.0 (Windows NT 10.0) Chrome", PlatformWeb},
		{"桌面浏览器mac", "Mozilla/5.0 (Macintosh) Safari", PlatformWeb},
		{"桌面浏览器linux", "Mozilla/5.0 (X11; Linux x86_64) Firefox", PlatformWeb},
		{"移动浏览器android无原生标志", "Mozilla/5.0 (Android) Chrome", PlatformWeb},
		{"Mozilla含iphone被iOS分支截获", "Mozilla/5.0 (iPhone) Safari", PlatformiOSApp},
		{"Mozilla含okhttp被Android分支截获", "Mozilla/5.0 okhttp Windows NT", PlatformAndroidApp},
		{"爬虫UA", "Googlebot/2.1", PlatformOther},
		{"普通文本", "random text", PlatformOther},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, detectPlatformFromUA(tc.ua))
		})
	}
}

// TestGenerateECDSAKeyPair 验证密钥对生成：secp256r1 曲线、公私钥配套。
func TestGenerateECDSAKeyPair(t *testing.T) {
	priv, pub, err := generateECDSAKeyPair()
	require.NoError(t, err)
	require.NotNil(t, priv)
	require.NotNil(t, pub)
	assert.Equal(t, elliptic.P256(), priv.Curve, "必须使用 secp256r1")
	assert.Same(t, &priv.PublicKey, pub, "公钥必须取自私钥")
}

// TestEncodeDER 表驱动验证 DER 编码：标准两整数序列、高位字节补 0x00
// 前缀、零值空整数。字节布局为外部验证方依赖的稳定契约。
func TestEncodeDER(t *testing.T) {
	cases := []struct {
		name string
		r    int64
		s    int64
		want []byte
	}{
		{"小整数", 1, 2, []byte{0x30, 0x06, 0x02, 0x01, 0x01, 0x02, 0x01, 0x02}},
		{"r高位补零", 0x80, 1, []byte{0x30, 0x07, 0x02, 0x02, 0x00, 0x80, 0x02, 0x01, 0x01}},
		{"s高位补零", 1, 0x80, []byte{0x30, 0x07, 0x02, 0x01, 0x01, 0x02, 0x02, 0x00, 0x80}},
		{"零值", 0, 0, []byte{0x30, 0x04, 0x02, 0x00, 0x02, 0x00}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			der, err := encodeDER(big.NewInt(tc.r), big.NewInt(tc.s))
			require.NoError(t, err)
			assert.Equal(t, tc.want, der)
		})
	}
}

package logging

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net"
	"regexp"
	"strings"

	"net/url"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/go-utils/geoip"
	"github.com/tx7do/go-utils/trans"

	"github.com/mileusna/useragent"
	"github.com/tx7do/go-utils/geoip/geolite"
	"github.com/tx7do/go-utils/jwtutil"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"

	"go-wind-admin/pkg/jwt"
)

var ipClient, _ = geolite.NewClient()

// extractAuthToken 从JWT Token中提取用户信息
func extractAuthToken(htr *http.Transport) *authenticationV1.UserTokenPayload {
	authToken := htr.RequestHeader().Get(HeaderKeyAuthorization)
	if len(authToken) == 0 {
		return nil
	}

	jwtToken := strings.TrimPrefix(authToken, "Bearer ")

	claims, err := jwtutil.ParseJWTPayload(jwtToken)
	if err != nil {
		log.Errorf("extractAuthToken ParseJWTPayload failed: %v", err)
		return nil
	}

	ut, err := jwt.NewUserTokenPayloadWithJwtMapClaims(claims)
	if err != nil {
		log.Errorf("extractAuthToken NewUserTokenPayloadWithJwtMapClaims failed: %v", err)
		return nil
	}

	return ut
}

// getClientRealIP 获取客户端真实IP
func getClientRealIP(request *http.Request) string {
	if request == nil {
		return ""
	}

	// 先检查 X-Forwarded-For 头
	// 由于它可以记录整个代理链中的IP地址，因此适用于多级代理的情况。
	// 当请求经过多个代理服務器时，X-Forwarded-For字段可以完整地记录原始请求的客户端IP地址和所有代理服務器的IP地址。
	// 需要注意：
	// 最外层Nginx配置为：proxy_set_header X-Forwarded-For $remote_addr; 如此做可以覆写掉ip。以防止ip伪造。
	// 里层Nginx配置为：proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
	xff := request.Header.Get(HeaderKeyXForwardedFor)
	if xff != "" {
		// X-Forwarded-For字段的值是一个逗号分隔的IP地址列表，
		// 一般来说，第一个IP地址是原始请求的客户端IP地址（当然，它可以被伪造）。
		ips := strings.Split(xff, ",")

		for _, ip := range ips {
			// 去除空格
			ip = strings.TrimSpace(ip)
			// 检查是否是合法的IP地址
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// 接着检查反向代理的 X-Real-IP 头
	// 通常只在反向代理服務器中使用，并且只记录原始请求的客户端IP地址。
	// 它不适用于多级代理的情况，因为每经过一个代理服務器，X-Real-IP字段的值都会被覆盖为最新的客户端IP地址。
	xri := request.Header.Get(HeaderKeyXRealIP)
	if xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	return getIPFromRemoteAddr(request.RemoteAddr)
}

func getIPFromRemoteAddr(hostAddress string) string {
	// Check if the host address contains a port
	if strings.Contains(hostAddress, ":") {
		// Attempt to split the host address into host and port
		host, _, err := net.SplitHostPort(strings.TrimSpace(hostAddress))
		if err == nil {
			// Validate the host as an IP address
			if net.ParseIP(host) != nil {
				return host
			}
		}
	}
	// Validate the host address as an IP address
	if net.ParseIP(hostAddress) != nil {
		return hostAddress
	}
	return ""
}

// getRequestId 获取请求ID
func getRequestId(request *http.Request) string {
	if request == nil {
		return ""
	}

	// 先检查 X-Request-ID 头
	// 这是比较常见的用于标识请求的自定义头部字段。
	// 例如，在一个微服務架构的系统中，当一个请求从前端应用发送到后端的多个微服務时，
	// 每个微服務都可以在 X-Request-ID 字段中获取到相同的请求标识，从而方便追踪请求在各个服務节点中的处理情况。
	xri := request.Header.Get(HeaderKeyXRequestID)
	if xri != "" {
		return xri
	}

	// 接着检查 X-Correlation-ID 头
	// 它和 X-Request-ID 类似，用于关联一系列相关的请求或者事务。
	// 比如，在一个包含多个子请求的复杂业务流程中，X-Correlation-ID 可以用于跟踪整个业务流程中各个子请求之间的关系。
	xci := request.Header.Get(HeaderKeyXCorrelationID)
	if xci != "" {
		return xci
	}

	// 函数计算的请求ID
	xfcri := request.Header.Get(HeaderKeyXFcRequestID)
	if xfcri != "" {
		return xfcri
	}

	return ""
}

// getClientID 获取客户端ID
func getClientID(request *http.Request, userToken *authenticationV1.UserTokenPayload) string {
	if request == nil {
		return ""
	}

	// 我们可以自定义一个Header叫做：X-Client-ID。
	xci := request.Header.Get(HeaderKeyXClientIP)
	if xci != "" {
		return xci
	}

	// 从JWT Token中获取ClientID也是可行的。
	if userToken != nil {
		return userToken.GetClientId()
	}

	return ""
}

// getStatusCode 状态码
func getStatusCode(err error) (uint32, string, bool) {
	// 1. 信息响应 (100–199)
	// 2. 成功响应 (200–299)
	// 3. 重定向消息 (300–399)
	// 4. 客户端错误响应 (400–499)
	// 5. 服務端错误响应 (500–599)
	if se := errors.FromError(err); se != nil {
		return uint32(se.Code), se.Reason, se.Code < 400
	} else {
		return 200, "", true
	}
}

// printUserAgent 打印User-Agent信息
func printUserAgent(strUserAgent string) {
	ua := useragent.Parse(strUserAgent)

	fmt.Println("User-Agent", ua)
	fmt.Println()
	fmt.Println(ua.String)
	fmt.Println(strings.Repeat("=", len(ua.String)))
	fmt.Println("Name:", ua.Name, "v", ua.Version)
	fmt.Println("OS:", ua.OS, "v", ua.OSVersion)
	fmt.Println("Device:", ua.Device)

	if ua.Mobile {
		fmt.Println("(Mobile)")
	}
	if ua.Tablet {
		fmt.Println("(Tablet)")
	}
	if ua.Desktop {
		fmt.Println("(Desktop)")
	}
	if ua.Bot {
		fmt.Println("(Bot)")
	}
	if ua.URL != "" {
		fmt.Println(ua.URL)
	}
}

var reUsername = regexp.MustCompile(`"username"\s*:\s*"([^"]+)"`)

// parseUsernameFromBytes 从请求体中解析用户名
func parseUsernameFromBytes(body []byte) (string, error) {
	if m := reUsername.FindSubmatch(body); m != nil {
		return string(m[1]), nil
	}
	if values, err := url.ParseQuery(string(body)); err == nil {
		if u := values.Get("username"); u != "" {
			return u, nil
		}
	}
	return "", fmt.Errorf("username not found")
}

// extractUsernameFromRequest 从HTTP请求中提取用户名
func extractUsernameFromRequest(r *http.Request) (username string, err error) {
	if r == nil {
		return "", fmt.Errorf("nil request")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	_ = r.Body.Close()
	// 恢复 Body，避免影响后续处理
	r.Body = io.NopCloser(bytes.NewReader(body))

	if username, err = parseUsernameFromBytes(body); err == nil {
		return username, nil
	}

	if values, err := url.ParseQuery(string(body)); err == nil {
		//fmt.Println("extractUsernameFromRequest Unmarshal Query", err)
		return values.Get("username"), nil
	}

	return "", err
}

// clientIpToLocation 获取客户端IP的地理位置
func clientIpToLocation(ip string) *geoip.Result {
	res, err := ipClient.Query(ip)
	if err != nil {
		return nil
	}
	return &res
}

// TokenHash 返回 token 的 SHA-256 十六进制摘要，用于审计存储（避免存明文）
func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// generateECDSAKeyPair 生成 ECDSA 密钥对（secp256r1 曲线）
func generateECDSAKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate ECDSA key failed: %w", err)
	}
	return privateKey, &privateKey.PublicKey, nil
}

// encodeDER 将 ECDSA 的 r、s 转为 DER 格式字节数组（标准签名字段）
func encodeDER(r, s *big.Int) ([]byte, error) {
	// DER 格式规则：0x30 + 总长度 + 0x02 + r长度 + r值 + 0x02 + s长度 + s值
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// 确保 r/s 是正整数（补 0 前缀）
	if len(rBytes) > 0 && rBytes[0]&0x80 != 0 {
		rBytes = append([]byte{0x00}, rBytes...)
	}
	if len(sBytes) > 0 && sBytes[0]&0x80 != 0 {
		sBytes = append([]byte{0x00}, sBytes...)
	}

	// 拼接 DER 字节
	der := make([]byte, 0)
	der = append(der, 0x30)                              // 序列标签
	der = append(der, byte(2+len(rBytes)+2+len(sBytes))) // 总长度
	der = append(der, 0x02)                              // 整数标签（r）
	der = append(der, byte(len(rBytes)))                 // r 长度
	der = append(der, rBytes...)
	der = append(der, 0x02)              // 整数标签（s）
	der = append(der, byte(len(sBytes))) // s 长度
	der = append(der, sBytes...)

	return der, nil
}

// fillDeviceInfo 填写设备信息
func fillDeviceInfo(htr *http.Transport, ut *authenticationV1.UserTokenPayload) (info *auditV1.DeviceInfo) {
	info = &auditV1.DeviceInfo{}

	userAgent := htr.RequestHeader().Get(HeaderKeyUserAgent)
	ua := useragent.Parse(userAgent)
	info.UserAgent = trans.Ptr(ua.String)

	var deviceName string
	if ua.Device != "" {
		deviceName = ua.Device
	} else {
		if ua.Desktop {
			deviceName = "PC"
		}
	}
	info.ClientName = trans.Ptr(deviceName)

	if ua.Desktop {
		info.DeviceType = trans.Ptr(auditV1.DeviceInfo_DESKTOP)
	} else if ua.Tablet {
		info.DeviceType = trans.Ptr(auditV1.DeviceInfo_TABLET)
	} else if ua.Mobile {
		info.DeviceType = trans.Ptr(auditV1.DeviceInfo_MOBILE)
	} else if ua.Bot {
		info.DeviceType = trans.Ptr(auditV1.DeviceInfo_BOT)
	} else {
		info.DeviceType = trans.Ptr(auditV1.DeviceInfo_OTHER)
	}

	info.BrowserVersion = trans.Ptr(ua.Version)
	info.BrowserName = trans.Ptr(ua.Name)

	info.OsName = trans.Ptr(ua.OS)
	info.OsVersion = trans.Ptr(ua.OSVersion)

	info.Platform = trans.Ptr(detectPlatformFromUA(userAgent))

	info.ClientId = trans.Ptr(getClientID(htr.Request(), ut))

	return
}

// fillGeoLocation 填写地理位置信息
func fillGeoLocation(clientIp string) (info *auditV1.GeoLocation) {
	info = &auditV1.GeoLocation{}

	result := clientIpToLocation(clientIp)
	if result == nil {
		return
	}

	info.CountryCode = trans.Ptr(result.Country)
	info.Province = trans.Ptr(result.Province)
	info.City = trans.Ptr(result.City)
	info.Isp = trans.Ptr(result.ISP)

	return
}

// isPrivateIP 检查 IP 是否属于常见内网或链路本地地址
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return false
	}
	cidrs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
	}
	for _, c := range cidrs {
		_, n, _ := net.ParseCIDR(c)
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

const (
	PlatformAndroidApp = "AndroidApp"
	PlatformiOSApp     = "iOSApp"

	PlatformDesktopWindows = "DesktopWindows"
	PlatformDesktopMac     = "DesktopMac"
	PlatformDesktopLinux   = "DesktopLinux"

	PlatformWeb = "Web"

	PlatformOther = "Other"
)

// detectPlatformFromUA 根据 UA 字符串启发式判断平台类型。
// 说明：仅作为补充判断，优先使用客户端上报字段（platform/os_name）。
func detectPlatformFromUA(ua string) string {
	if ua == "" {
		return PlatformOther
	}
	s := strings.ToLower(strings.TrimSpace(ua))

	// 原生 Android app 指示器
	if strings.Contains(s, "okhttp") || strings.Contains(s, "dalvik") || strings.Contains(s, "; wv") ||
		strings.Contains(s, " ;wv") || strings.Contains(s, "build/") {
		return PlatformAndroidApp
	}
	// 包名模式（com.xxx）且含 android
	if strings.Contains(s, "android") {
		if regexp.MustCompile(`\bcom\.[a-z0-9_.]+`).MatchString(s) || strings.Contains(s, "wv") {
			return PlatformAndroidApp
		}
	}

	// 原生 iOS 指示器
	if strings.Contains(s, "iphone") || strings.Contains(s, "ipad") || strings.Contains(s, "ipod") ||
		strings.Contains(s, "cfnetwork") || strings.Contains(s, "darwin") || strings.Contains(s, "cpu iphone os") {
		return PlatformiOSApp
	}

	// 桌面原生/混合应用（例如 Electron、nwjs、desktop）
	if strings.Contains(s, "electron") || strings.Contains(s, "nwjs") || strings.Contains(s, "node.js") || strings.Contains(s, "nodejs") ||
		strings.Contains(s, "desktop") || strings.Contains(s, "appname") {
		// 根据 UA 中的 OS 关键词区分具体桌面系统
		if strings.Contains(s, "windows nt") || strings.Contains(s, "win64") || strings.Contains(s, "win32") || strings.Contains(s, "windows") {
			return PlatformDesktopWindows
		}
		if strings.Contains(s, "macintosh") || strings.Contains(s, "mac os x") || strings.Contains(s, "darwin") {
			return PlatformDesktopMac
		}
		if strings.Contains(s, "x11") || strings.Contains(s, "linux") || strings.Contains(s, "ubuntu") || strings.Contains(s, "debian") || strings.Contains(s, "fedora") {
			return PlatformDesktopLinux
		}
		// 若未直接包含 OS 关键字，尝试从常见标识推断
		if strings.Contains(s, "win") || strings.Contains(s, "windows") {
			return PlatformDesktopWindows
		}
		if strings.Contains(s, "mac") || strings.Contains(s, "os x") {
			return PlatformDesktopMac
		}
		if strings.Contains(s, "linux") || strings.Contains(s, "x11") {
			return PlatformDesktopLinux
		}
		// 仍无法判定，视作桌面类但未知系统
		return PlatformOther
	}

	// 常见 Web 浏览器识别（Mozilla+桌面/移动系统且无明显原生 app 标志）
	if strings.Contains(s, "mozilla") && (strings.Contains(s, "windows nt") || strings.Contains(s, "macintosh") ||
		strings.Contains(s, "x11") || strings.Contains(s, "linux") || strings.Contains(s, "android") || strings.Contains(s, "iphone")) {
		// 排除明显的原生 app 标志
		if !strings.Contains(s, "okhttp") && !strings.Contains(s, "dalvik") && !strings.Contains(s, "cfnetwork") && !strings.Contains(s, "electron") {
			return PlatformWeb
		}
	}

	return PlatformOther
}

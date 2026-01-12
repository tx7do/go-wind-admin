package logging

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"

	"github.com/mileusna/useragent"

	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
)

// Server is an server logging middleware.
func Server(opts ...Option) middleware.Middleware {
	op := options{}
	for _, o := range opts {
		o(&op)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			startTime := time.Now()

			reply, err = handler(ctx, req)

			// 统计耗时
			latency := time.Since(startTime).Seconds()

			var apiAuditLog *adminV1.ApiAuditLog
			var loginAuditLog *adminV1.LoginAuditLog

			if tr, ok := transport.FromServerContext(ctx); ok {
				var htr *http.Transport
				if htr, ok = tr.(*http.Transport); ok {
					loginAuditLog = fillLoginLog(htr)
					apiAuditLog = fillApiLog(htr)
				}
			}

			// 获取错误码和是否成功
			statusCode, reason, success := getStatusCode(err)

			if apiAuditLog != nil {
				apiAuditLog.CostTime = timeutil.Float64ToDurationpb(latency)
				apiAuditLog.StatusCode = trans.Ptr(statusCode)
				apiAuditLog.Reason = trans.Ptr(reason)
				apiAuditLog.Success = trans.Ptr(success)
			}
			if loginAuditLog != nil {
				loginAuditLog.StatusCode = trans.Ptr(statusCode)
				loginAuditLog.Reason = trans.Ptr(reason)
				loginAuditLog.Success = trans.Ptr(success)
			}

			if op.writeApiLogFunc != nil {
				_ = op.writeApiLogFunc(ctx, apiAuditLog)
			}
			if op.writeLoginLogFunc != nil {
				_ = op.writeLoginLogFunc(ctx, loginAuditLog)
			}

			return
		}
	}
}

// fillLoginLog 填充登录日志
func fillLoginLog(htr *http.Transport) *adminV1.LoginAuditLog {
	if htr.Operation() != adminV1.OperationAuthenticationServiceLogin {
		return nil
	}

	loginLogData := &adminV1.LoginAuditLog{}

	clientIp := getClientRealIP(htr.Request())

	loginLogData.LoginIp = trans.Ptr(clientIp)
	loginLogData.LoginTime = timeutil.TimeToTimestamppb(trans.Ptr(time.Now()))

	loginLogData.Location = trans.Ptr(clientIpToLocation(clientIp))

	if username, err := BindLoginRequest(htr.Request()); err == nil {
		loginLogData.Username = trans.Ptr(username)
	}

	// 获取客户端ID
	loginLogData.ClientId = trans.Ptr(getClientID(htr.Request(), nil))

	// 用户代理信息
	strUserAgent := htr.RequestHeader().Get(HeaderKeyUserAgent)
	ua := useragent.Parse(strUserAgent)

	var deviceName string
	if ua.Device != "" {
		deviceName = ua.Device
	} else {
		if ua.Desktop {
			deviceName = "PC"
		}
	}

	loginLogData.UserAgent = trans.Ptr(ua.String)
	loginLogData.BrowserVersion = trans.Ptr(ua.Version)
	loginLogData.BrowserName = trans.Ptr(ua.Name)
	loginLogData.OsName = trans.Ptr(ua.OS)
	loginLogData.OsVersion = trans.Ptr(ua.OSVersion)
	loginLogData.ClientName = trans.Ptr(deviceName)

	return loginLogData
}

// fillApiLog 填充API日志
func fillApiLog(htr *http.Transport) *adminV1.ApiAuditLog {
	if htr.Operation() == adminV1.OperationAuthenticationServiceLogin {
		return nil
	}

	apiAuditLog := &adminV1.ApiAuditLog{}

	clientIp := getClientRealIP(htr.Request())
	referer, _ := url.QueryUnescape(htr.RequestHeader().Get(HeaderKeyReferer))
	requestUri, _ := url.QueryUnescape(htr.Request().RequestURI)
	bodyBytes, _ := io.ReadAll(htr.Request().Body)

	apiAuditLog.Method = trans.Ptr(htr.Request().Method)
	apiAuditLog.Operation = trans.Ptr(htr.Operation())
	apiAuditLog.Path = trans.Ptr(htr.PathTemplate())
	apiAuditLog.Referer = trans.Ptr(referer)
	apiAuditLog.ClientIp = trans.Ptr(clientIp)
	apiAuditLog.RequestId = trans.Ptr(getRequestId(htr.Request()))
	apiAuditLog.RequestUri = trans.Ptr(requestUri)
	apiAuditLog.RequestBody = trans.Ptr(string(bodyBytes))
	apiAuditLog.Location = trans.Ptr(clientIpToLocation(clientIp))

	ut := extractAuthToken(htr)
	if ut != nil {
		apiAuditLog.UserId = trans.Ptr(ut.UserId)
		apiAuditLog.Username = ut.Username
	}

	// 获取客户端ID
	apiAuditLog.ClientId = trans.Ptr(getClientID(htr.Request(), ut))

	// 用户代理信息
	strUserAgent := htr.RequestHeader().Get(HeaderKeyUserAgent)
	ua := useragent.Parse(strUserAgent)

	var deviceName string
	if ua.Device != "" {
		deviceName = ua.Device
	} else {
		if ua.Desktop {
			deviceName = "PC"
		}
	}

	apiAuditLog.UserAgent = trans.Ptr(ua.String)
	apiAuditLog.BrowserVersion = trans.Ptr(ua.Version)
	apiAuditLog.BrowserName = trans.Ptr(ua.Name)
	apiAuditLog.OsName = trans.Ptr(ua.OS)
	apiAuditLog.OsVersion = trans.Ptr(ua.OSVersion)
	apiAuditLog.ClientName = trans.Ptr(deviceName)

	return apiAuditLog
}

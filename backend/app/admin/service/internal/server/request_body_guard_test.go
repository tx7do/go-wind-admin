package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-kratos/kratos/v2/middleware"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

type stubAccessTokenChecker struct {
	validToken string
}

func (s *stubAccessTokenChecker) IsValidAccessToken(_ context.Context, accessToken string, _ bool) (bool, *authenticationV1.UserTokenPayload) {
	return accessToken == s.validToken, nil
}

func (s *stubAccessTokenChecker) IsBlockedAccessToken(_ context.Context, _ string) bool {
	return false
}

// countingBody 记录是否被读过；用于断言"拦截发生在读取 body 之前"。
type countingBody struct {
	read bool
}

func (b *countingBody) Read([]byte) (int, error) {
	b.read = true
	return 0, io.EOF
}

func (b *countingBody) Close() error { return nil }

func TestUploadBodyGuardRejectsDeclaredOversizeWithoutReading(t *testing.T) {
	body := &countingBody{}
	nextCalled := false

	h := uploadBodyGuardWith(func(*nethttp.Request) int64 { return 16 })(nethttp.HandlerFunc(func(nethttp.ResponseWriter, *nethttp.Request) {
		nextCalled = true
	}))

	req := httptest.NewRequest(nethttp.MethodPost, "/admin/v1/file/upload", body)
	req.ContentLength = 17
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, nethttp.StatusRequestEntityTooLarge)
	}
	if nextCalled {
		t.Fatal("next handler should not be called for oversize body")
	}
	if body.read {
		t.Fatal("body must not be read when Content-Length already exceeds the limit")
	}
}

func TestUploadBodyGuardCapsUnknownLengthBody(t *testing.T) {
	var readErr error

	h := uploadBodyGuardWith(func(*nethttp.Request) int64 { return 16 })(nethttp.HandlerFunc(func(_ nethttp.ResponseWriter, r *nethttp.Request) {
		_, readErr = io.ReadAll(r.Body)
	}))

	req := httptest.NewRequest(nethttp.MethodPost, "/admin/v1/file/upload", strings.NewReader(strings.Repeat("a", 64)))
	req.ContentLength = -1
	h.ServeHTTP(httptest.NewRecorder(), req)

	var maxBytesErr *nethttp.MaxBytesError
	if !errors.As(readErr, &maxBytesErr) {
		t.Fatalf("read error = %v, want *http.MaxBytesError", readErr)
	}
}

func TestUploadBodyGuardSelectsLimitByContentType(t *testing.T) {
	multipartRequest := func() *nethttp.Request {
		req := httptest.NewRequest(nethttp.MethodPost, "/admin/v1/file/upload", &countingBody{})
		req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
		req.ContentLength = maxUploadBody + 1
		return req
	}

	rec := httptest.NewRecorder()
	uploadBodyGuard()(nethttp.HandlerFunc(func(nethttp.ResponseWriter, *nethttp.Request) {
		t.Fatal("oversize multipart body should be rejected before the handler")
	})).ServeHTTP(rec, multipartRequest())
	if rec.Code != nethttp.StatusRequestEntityTooLarge {
		t.Fatalf("multipart status = %d, want %d", rec.Code, nethttp.StatusRequestEntityTooLarge)
	}

	// JSON 形态（base64）上限更宽，同样长度应放行。
	passed := false
	jsonReq := multipartRequest()
	jsonReq.Header.Set("Content-Type", "application/json")
	uploadBodyGuard()(nethttp.HandlerFunc(func(nethttp.ResponseWriter, *nethttp.Request) {
		passed = true
	})).ServeHTTP(httptest.NewRecorder(), jsonReq)
	if !passed {
		t.Fatal("JSON base64 body within the global limit should pass the upload guard")
	}
}

func TestRequestBodyGuardAllowsSmallUnauthenticatedBody(t *testing.T) {
	nextCalled := false

	h := requestBodyGuard(&stubAccessTokenChecker{validToken: "good"})(nethttp.HandlerFunc(func(nethttp.ResponseWriter, *nethttp.Request) {
		nextCalled = true
	}))

	req := httptest.NewRequest(nethttp.MethodPost, "/admin/v1/auth/login", strings.NewReader(`{"username":"admin"}`))
	h.ServeHTTP(httptest.NewRecorder(), req)

	if !nextCalled {
		t.Fatal("small unauthenticated request should pass through")
	}
}

func TestRequestBodyGuardRejectsLargeUnauthenticatedBodyWithoutReading(t *testing.T) {
	body := &countingBody{}
	nextCalled := false

	h := requestBodyGuard(&stubAccessTokenChecker{validToken: "good"})(nethttp.HandlerFunc(func(nethttp.ResponseWriter, *nethttp.Request) {
		nextCalled = true
	}))

	req := httptest.NewRequest(nethttp.MethodPost, "/admin/v1/auth/login", body)
	req.ContentLength = maxUnauthenticatedBody + 1
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, nethttp.StatusUnauthorized)
	}
	if nextCalled {
		t.Fatal("next handler should not be called without a valid token")
	}
	if body.read {
		t.Fatal("body must not be read before authentication")
	}
}

func TestRequestBodyGuardRejectsInvalidTokenForLargeBody(t *testing.T) {
	h := requestBodyGuard(&stubAccessTokenChecker{validToken: "good"})(nethttp.HandlerFunc(func(nethttp.ResponseWriter, *nethttp.Request) {
		t.Fatal("next handler should not be called with an invalid token")
	}))

	req := httptest.NewRequest(nethttp.MethodPost, "/admin/v1/me/avatar", &countingBody{})
	req.ContentLength = maxUnauthenticatedBody + 1
	req.Header.Set("Authorization", "Bearer bad")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, nethttp.StatusUnauthorized)
	}
}

func TestRequestBodyGuardAllowsLargeAuthenticatedBody(t *testing.T) {
	nextCalled := false

	h := requestBodyGuard(&stubAccessTokenChecker{validToken: "good"})(nethttp.HandlerFunc(func(_ nethttp.ResponseWriter, r *nethttp.Request) {
		nextCalled = true
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			t.Errorf("read large body: %v", err)
		}
	}))

	payload := bytes.Repeat([]byte("a"), maxUnauthenticatedBody+1024)
	req := httptest.NewRequest(nethttp.MethodPost, "/admin/v1/me/avatar", bytes.NewReader(payload))
	req.ContentLength = int64(len(payload))
	req.Header.Set("Authorization", "Bearer good")

	h.ServeHTTP(httptest.NewRecorder(), req)

	if !nextCalled {
		t.Fatal("large request with a valid token should pass through")
	}
}

func TestRequestBodyGuardCapsChunkedUnauthenticatedBody(t *testing.T) {
	var readErr error

	h := requestBodyGuard(&stubAccessTokenChecker{validToken: "good"})(nethttp.HandlerFunc(func(_ nethttp.ResponseWriter, r *nethttp.Request) {
		_, readErr = io.ReadAll(r.Body)
	}))

	req := httptest.NewRequest(nethttp.MethodPost, "/admin/v1/auth/login", strings.NewReader(strings.Repeat("a", maxUnauthenticatedBody+1)))
	req.ContentLength = -1

	h.ServeHTTP(httptest.NewRecorder(), req)

	var maxBytesErr *nethttp.MaxBytesError
	if !errors.As(readErr, &maxBytesErr) {
		t.Fatalf("read error = %v, want *http.MaxBytesError", readErr)
	}
}

func TestRequestBodyGuardRejectsBodyBeyondGlobalLimit(t *testing.T) {
	body := &countingBody{}

	h := requestBodyGuard(&stubAccessTokenChecker{validToken: "good"})(nethttp.HandlerFunc(func(nethttp.ResponseWriter, *nethttp.Request) {
		t.Fatal("next handler should not be called for an oversize body")
	}))

	req := httptest.NewRequest(nethttp.MethodPost, "/admin/v1/file/upload", body)
	req.ContentLength = maxRequestBody + 1
	req.Header.Set("Authorization", "Bearer good")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, nethttp.StatusRequestEntityTooLarge)
	}
	if body.read {
		t.Fatal("body must not be read when Content-Length already exceeds the global limit")
	}
}

// TestUploadHandlerParsesBodyAfterMiddleware 守护本次修复的核心次序：
// 请求体必须在 ctx.Middleware 的回调内部解析，认证/鉴权中间件先于读 body 执行。
func TestUploadHandlerParsesBodyAfterMiddleware(t *testing.T) {
	var bodyReadBeforeMiddleware bool
	var readStarted atomic.Bool

	probe := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			bodyReadBeforeMiddleware = readStarted.Load()
			return next(ctx, req)
		}
	}

	srv := kratoshttp.NewServer()
	srv.Use(OperationFileTransferServicePostUploadFile, probe)
	srv.Route("/").POST(
		"/admin/v1/file/upload",
		_FileTransferService_PostUploadFile_HTTP_Handler(nil),
		uploadBodyGuardWith(func(*nethttp.Request) int64 { return 1024 }),
	)

	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	part, err := writer.CreateFormFile("file", "poc.bin")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err = part.Write(bytes.Repeat([]byte("a"), 4096)); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err = writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	body := &readNotifyBody{Reader: bytes.NewReader(payload.Bytes()), onRead: func() {
		readStarted.Store(true)
	}}
	req := httptest.NewRequest(nethttp.MethodPost, "/admin/v1/file/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.ContentLength = -1
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if bodyReadBeforeMiddleware {
		t.Fatal("request body was parsed before the middleware chain ran")
	}
	if !readStarted.Load() {
		t.Fatal("request body was never read; test did not exercise parsing")
	}
	if rec.Code != nethttp.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, nethttp.StatusRequestEntityTooLarge)
	}
}

type readNotifyBody struct {
	io.Reader
	onRead func()
}

func (b *readNotifyBody) Read(p []byte) (int, error) {
	b.onRead()
	return b.Reader.Read(p)
}

func (b *readNotifyBody) Close() error { return nil }

package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	nethttp "net/http"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/go-utils/ossutil"
	"github.com/tx7do/go-utils/trans"

	"github.com/go-kratos/kratos/v2/encoding"
	_ "github.com/go-kratos/kratos/v2/encoding/json"

	"go-wind-admin/app/admin/service/internal/service"

	storageV1 "go-wind-admin/api/gen/go/storage/service/v1"
)

var codec = encoding.GetCodec("json")

// maxUploadBody 上传路由的请求体上限：单文件上限 + multipart 编码余量。
const maxUploadBody = ossutil.MaxUploadSize + 1<<20

func registerFileTransferServiceHandler(srv *http.Server, svc *service.FileTransferService) {
	r := srv.Route("/")

	// 上传路由在 HTTP Filter 层单独封顶，而不是放在 kratos 中间件链里：
	// ctx.Middleware 由 handler 自己调用，中间件实际在请求体解析之后才执行，
	// 无法阻止 multipart 解码先把整包读进内存。
	uploadGuard := uploadBodyGuard()

	r.POST("admin/v1/file/upload", _FileTransferService_PostUploadFile_HTTP_Handler(svc), uploadGuard)
	r.PUT("admin/v1/file/upload", _FileTransferService_PutUploadFile_HTTP_Handler(svc), uploadGuard)

	r.GET("admin/v1/file/download", _FileTransferService_DownloadFile_HTTP_Handler(svc))

	// 签名图片代理（免鉴权：签名即凭证）。手动注册的路由未设置 Operation，
	// selector 按 Operation 匹配的 auth 中间件天然不应用；安全性由
	// HMAC 签名 + 过期时间在 handler 内校验保证。
	r.GET("admin/v1/file/image", func(ctx http.Context) error {
		return svc.ServeImageHandler(ctx.Response(), ctx.Request())
	})
}

const OperationFileTransferServicePostUploadFile = "/admin.service.v1.FileTransferService/PostUploadFile"
const OperationFileTransferServicePutUploadFile = "/admin.service.v1.FileTransferService/PutUploadFile"

const OperationFileTransferServiceDownloadFile = "/admin.service.v1.FileTransferService/DownloadFile"

// uploadBodyGuard 在 handler 之前给上传请求体封顶。
// multipart 的 body 约等于文件本身（50MiB + 编码余量）；JSON 形态走 base64
// （膨胀 4/3，react 端实际用这条路径），上限需容纳 50MiB 解码后的文件。
// Content-Length 已超限时直接 413、不读 body；长度未知（chunked）时用
// MaxBytesReader 兜住，读取超限会以 *http.MaxBytesError 中止解析。
func uploadBodyGuard() http.FilterFunc {
	return uploadBodyGuardWith(func(r *nethttp.Request) int64 {
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			return maxUploadBody
		}
		return maxRequestBody
	})
}

// uploadBodyGuardWith 按请求计算上限，便于测试用小阈值覆盖。
func uploadBodyGuardWith(limitFor func(*nethttp.Request) int64) http.FilterFunc {
	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			limit := limitFor(r)
			if r.ContentLength > limit {
				nethttp.Error(w, "request entity too large", nethttp.StatusRequestEntityTooLarge)
				return
			}
			r.Body = nethttp.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

func _FileTransferService_PostUploadFile_HTTP_Handler(svc *service.FileTransferService) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		http.SetOperation(ctx, OperationFileTransferServicePostUploadFile)
		return serveUploadFile(ctx, svc)
	}
}

func _FileTransferService_PutUploadFile_HTTP_Handler(svc *service.FileTransferService) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		http.SetOperation(ctx, OperationFileTransferServicePutUploadFile)
		return serveUploadFile(ctx, svc)
	}
}

// serveUploadFile POST/PUT 上传的共用实现。
// 关键点：请求体解析放在 ctx.Middleware 返回的 handler 内部。kratos 的中间件链
// 是在 h(ctx, &in) 被调用时才执行的，若在它之前解析 body，认证/鉴权与请求体
// 上限都会被推迟到整包读入内存之后（未认证即可打爆内存）。
func serveUploadFile(ctx http.Context, svc *service.FileTransferService) error {
	var in storageV1.UploadFileRequest

	h := ctx.Middleware(func(mctx context.Context, req interface{}) (interface{}, error) {
		aReq := req.(*storageV1.UploadFileRequest)

		if err := parseUploadFileRequest(ctx, aReq); err != nil {
			return nil, err
		}

		resp, err := svc.UploadFile(mctx, aReq)
		aReq.Source = nil

		return resp, err
	})

	// 逻辑处理，取数据
	out, err := h(ctx, &in)
	if err != nil {
		return err
	}

	reply, ok := out.(*storageV1.UploadFileResponse)
	if !ok {
		return storageV1.ErrorUploadFailed("unexpected response type %T", out)
	}

	return ctx.Result(200, reply)
}

// parseUploadFileRequest 解析 multipart 上传入参。读取受 uploadBodyGuard 安装的
// MaxBytesReader 约束，超限时映射为 413；文件内容只做一次完整拷贝。
func parseUploadFileRequest(ctx http.Context, in *storageV1.UploadFileRequest) error {
	file, header, err := ctx.Request().FormFile("file")
	if err != nil {
		if isMaxBytesError(err) {
			return storageV1.ErrorFileTooLarge("request body exceeds the upload body limit")
		}

		// 非 multipart 请求沿用原有行为：交给框架按 Content-Type 解码。
		if bindErr := ctx.Bind(in); bindErr != nil {
			log.Errorf("Bind upload file request error: %v", bindErr)
			return bindErr
		}
		return nil
	}
	defer file.Close()

	if header.Size > ossutil.MaxUploadSize {
		return storageV1.ErrorFileTooLarge("file size %d exceeds max upload size %d", header.Size, ossutil.MaxUploadSize)
	}

	// 此前 strings.Builder → []byte(b.String()) 会同时持有两份等大副本，大文件下
	// 内存直接翻倍；按已解析出的实际大小预分配，只保留一份。
	var buf bytes.Buffer
	if header.Size > 0 {
		buf.Grow(int(header.Size))
	}
	if _, err = io.Copy(&buf, file); err != nil {
		if isMaxBytesError(err) {
			return storageV1.ErrorFileTooLarge("request body exceeds the upload body limit")
		}
		return err
	}

	fileData := buf.Bytes()
	if fileData == nil {
		fileData = []byte{}
	}
	in.Source = &storageV1.UploadFileRequest_File{File: fileData}

	return applyUploadFileFormMeta(ctx, in, header)
}

// applyUploadFileFormMeta 读取 multipart 中与文件同行的表单元数据。
func applyUploadFileFormMeta(ctx http.Context, in *storageV1.UploadFileRequest, header *multipart.FileHeader) error {
	sourceFileName := ctx.Request().FormValue("sourceFileName")
	if sourceFileName != "" {
		in.SourceFileName = trans.Ptr(sourceFileName)
	} else {
		in.SourceFileName = trans.Ptr(header.Filename)
	}
	//log.Debugf("Upload file sourceFileName: %s", sourceFileName)

	mime := ctx.Request().FormValue("mime")
	if mime != "" {
		in.Mime = trans.Ptr(mime)
	} else {
		in.Mime = trans.Ptr(header.Header.Get("Content-Type"))
	}
	//log.Debugf("Upload file mime: %s", mime)

	size := ctx.Request().FormValue("size")
	if size != "" {
		if n, parseErr := strconv.ParseInt(size, 10, 64); parseErr == nil {
			in.Size = trans.Ptr(n)
		}
	} else {
		in.Size = trans.Ptr(header.Size)
	}
	//log.Debugf("Upload file size: %s", size)

	storageObject := ctx.Request().FormValue("storageObject")
	if storageObject != "" {
		in.StorageObject = &storageV1.StorageObject{}
		if err := codec.Unmarshal([]byte(storageObject), in.StorageObject); err != nil {
			log.Errorf("Unmarshal upload file storageObject error: %v", err)
			return err
		}
	}
	//log.Debugf("Upload file storageObject: %s", storageObject)

	return nil
}

// isMaxBytesError 判断错误链中是否包含请求体超限错误。
func isMaxBytesError(err error) bool {
	var maxBytesErr *nethttp.MaxBytesError
	return errors.As(err, &maxBytesErr)
}

func _FileTransferService_DownloadFile_HTTP_Handler(svc *service.FileTransferService) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		http.SetOperation(ctx, OperationFileTransferServiceDownloadFile)

		var in storageV1.DownloadFileRequest
		var err error

		if err = ctx.BindQuery(&in); err != nil {
			return err
		}

		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			aReq := req.(*storageV1.DownloadFileRequest)
			var resp *storageV1.DownloadFileResponse
			resp, err = svc.DownloadFile(ctx, aReq)
			return resp, err
		})

		// 逻辑处理，取数据
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}

		reply := out.(*storageV1.DownloadFileResponse)
		rw := ctx.Response()
		if rw == nil {
			return ctx.Result(500, "response writer not available")
		}

		data := reply.GetFile()
		if len(data) == 0 {
			// 若没有文件字节，交由框架默认处理（保持原行为）
			return ctx.Result(200, reply)
		}

		// 基本头部
		mime := reply.GetMime()
		if mime == "" {
			mime = "application/octet-stream"
		}
		rw.Header().Set("Content-Type", mime)

		filename := reply.GetSourceFileName()
		if filename == "" {
			filename = "file"
		}

		var disposition string
		if in.GetDisposition() != "" {
			disposition = in.GetDisposition()
		} else {
			disposition = "attachment; filename=\"" + filename + "\""
		}
		rw.Header().Set("Content-Disposition", disposition)
		rw.Header().Set("Accept-Ranges", "bytes")

		// 使用 bytes.Reader 以便支持高效的部分读取/流式写入
		reader := bytes.NewReader(data)
		total := int64(len(data))

		// 检查请求是否带 Range 头（仅支持单区间，格式 bytes=start-end 或 bytes=start-）
		rangeHeader := ctx.Request().Header.Get("Range")
		if strings.HasPrefix(rangeHeader, "bytes=") {
			r := strings.TrimPrefix(rangeHeader, "bytes=")
			parts := strings.SplitN(r, "-", 2)
			if len(parts) != 2 {
				return ctx.Result(400, "invalid Range header")
			}

			start, err1 := strconv.ParseInt(parts[0], 10, 64)
			if err1 != nil || start < 0 {
				return ctx.Result(400, "invalid Range start")
			}

			var end int64
			if parts[1] == "" {
				end = total - 1
			} else {
				end, err1 = strconv.ParseInt(parts[1], 10, 64)
				if err1 != nil || end < start {
					return ctx.Result(400, "invalid Range end")
				}
			}

			if start >= total {
				return ctx.Result(416, "requested range not satisfiable")
			}
			if end >= total {
				end = total - 1
			}

			length := end - start + 1
			// 设置部分响应头
			rw.Header().Set("Content-Range", "bytes "+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10)+"/"+strconv.FormatInt(total, 10))
			rw.Header().Set("Content-Length", strconv.FormatInt(length, 10))

			// 返回 206 并写入指定区间
			rw.WriteHeader(206)
			if _, err = reader.Seek(start, io.SeekStart); err != nil {
				return ctx.Result(500, err.Error())
			}
			if _, err = io.CopyN(rw, reader, length); err != nil {
				return ctx.Result(500, err.Error())
			}
			return nil
		}

		// 无 Range，返回完整内容（200）
		rw.Header().Set("Content-Length", strconv.FormatInt(total, 10))
		rw.WriteHeader(200)
		if _, err = io.Copy(rw, reader); err != nil {
			return ctx.Result(500, err.Error())
		}
		return nil
	}
}

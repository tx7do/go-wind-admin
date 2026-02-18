package server

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/go-utils/trans"

	"github.com/go-kratos/kratos/v2/encoding"
	_ "github.com/go-kratos/kratos/v2/encoding/json"

	"go-wind-admin/app/admin/service/internal/service"

	storageV1 "go-wind-admin/api/gen/go/storage/service/v1"
)

var codec = encoding.GetCodec("json")

func registerFileTransferServiceHandler(srv *http.Server, svc *service.FileTransferService) {
	r := srv.Route("/")

	r.POST("admin/v1/file/upload", _FileTransferService_PostUploadFile_HTTP_Handler(svc))
	r.PUT("admin/v1/file/upload", _FileTransferService_PutUploadFile_HTTP_Handler(svc))

	r.GET("admin/v1/file/download", _FileTransferService_DownloadFile_HTTP_Handler(svc))

	r.POST("admin/v1/ueditor", _FileTransferService_UEditorPostUploadFile_HTTP_Handler(svc))
	r.PUT("admin/v1/ueditor", _FileTransferService_UEditorPutUploadFile_HTTP_Handler(svc))
}

const OperationFileTransferServicePostUploadFile = "/admin.service.v1.FileTransferService/PostUploadFile"
const OperationFileTransferServicePutUploadFile = "/admin.service.v1.FileTransferService/PutUploadFile"

const OperationFileTransferServiceDownloadFile = "/admin.service.v1.FileTransferService/DownloadFile"

const OperationFileTransferServiceUEditorPostUploadFile = "/admin.service.v1.FileTransferService/UEditorPostUploadFile"
const OperationFileTransferServiceUEditorPutUploadFile = "/admin.service.v1.FileTransferService/UEditorPutUploadFile"

func _FileTransferService_PostUploadFile_HTTP_Handler(svc *service.FileTransferService) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		http.SetOperation(ctx, OperationFileTransferServicePostUploadFile)

		var in storageV1.UploadFileRequest
		var err error

		file, header, err := ctx.Request().FormFile("file")
		if err == nil {
			defer file.Close()

			b := new(strings.Builder)
			_, err = io.Copy(b, file)

			in.Source = &storageV1.UploadFileRequest_File{File: []byte(b.String())}

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
				var n int64
				n, err = strconv.ParseInt(size, 10, 64)
				if err == nil {
					in.Size = trans.Ptr(n)
				}
			} else {
				in.Size = trans.Ptr(header.Size)
			}
			//log.Debugf("Upload file size: %s", size)

			storageObject := ctx.Request().FormValue("storageObject")
			if storageObject != "" {
				in.StorageObject = &storageV1.StorageObject{}
				if err = codec.Unmarshal([]byte(storageObject), in.StorageObject); err != nil {
					log.Errorf("Unmarshal upload file storageObject error: %v", err)
					return err
				}
			}
			//log.Debugf("Upload file storageObject: %s", storageObject)
		} else {
			if err = ctx.Bind(&in); err != nil {
				log.Errorf("Bind upload file request error: %v", err)
				return err
			}
		}

		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			aReq := req.(*storageV1.UploadFileRequest)

			var resp *storageV1.UploadFileResponse
			resp, err = svc.UploadFile(ctx, aReq)
			in.Source = nil

			return resp, err
		})

		// 逻辑处理，取数据
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}

		reply := out.(*storageV1.UploadFileResponse)

		return ctx.Result(200, reply)
	}
}

func _FileTransferService_PutUploadFile_HTTP_Handler(svc *service.FileTransferService) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		http.SetOperation(ctx, OperationFileTransferServicePutUploadFile)

		var in storageV1.UploadFileRequest
		var err error

		file, header, err := ctx.Request().FormFile("file")
		if err == nil {
			defer file.Close()

			b := new(strings.Builder)
			_, err = io.Copy(b, file)

			in.Source = &storageV1.UploadFileRequest_File{File: []byte(b.String())}

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
				var n int64
				n, err = strconv.ParseInt(size, 10, 64)
				if err == nil {
					in.Size = trans.Ptr(n)
				}
			} else {
				in.Size = trans.Ptr(header.Size)
			}
			//log.Debugf("Upload file size: %s", size)

			storageObject := ctx.Request().FormValue("storageObject")
			if storageObject != "" {
				in.StorageObject = &storageV1.StorageObject{}
				if err = codec.Unmarshal([]byte(storageObject), in.StorageObject); err != nil {
					log.Errorf("Unmarshal upload file storageObject error: %v", err)
					return err
				}
			}
			//log.Debugf("Upload file storageObject: %s", storageObject)
		} else {
			if err = ctx.Bind(&in); err != nil {
				log.Errorf("Bind upload file request error: %v", err)
				return err
			}
		}

		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			aReq := req.(*storageV1.UploadFileRequest)

			var resp *storageV1.UploadFileResponse
			resp, err = svc.UploadFile(ctx, aReq)
			in.Source = nil

			return resp, err
		})

		// 逻辑处理，取数据
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}

		reply := out.(*storageV1.UploadFileResponse)

		return ctx.Result(200, reply)
	}
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

func _FileTransferService_UEditorPostUploadFile_HTTP_Handler(svc *service.FileTransferService) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		http.SetOperation(ctx, OperationFileTransferServiceUEditorPostUploadFile)

		var in storageV1.UEditorUploadRequest
		var err error

		file, header, err := ctx.Request().FormFile("file")
		if err == nil {
			defer file.Close()

			b := new(strings.Builder)
			_, err = io.Copy(b, file)

			in.SourceFileName = trans.Ptr(header.Filename)
			in.Mime = trans.Ptr(header.Header.Get("Content-Type"))
			in.File = []byte(b.String())
		}

		if err = ctx.BindQuery(&in); err != nil {
			return err
		}

		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			var resp *storageV1.UEditorUploadResponse

			resp, err = svc.UEditorUploadFile(ctx, req.(*storageV1.UEditorUploadRequest))
			in.File = nil

			return resp, err
		})

		// 逻辑处理，取数据
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}

		reply := out.(*storageV1.UEditorUploadResponse)

		return ctx.Result(200, reply)
	}
}

func _FileTransferService_UEditorPutUploadFile_HTTP_Handler(svc *service.FileTransferService) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		http.SetOperation(ctx, OperationFileTransferServiceUEditorPutUploadFile)

		var in storageV1.UEditorUploadRequest
		var err error

		file, header, err := ctx.Request().FormFile("file")
		if err == nil {
			defer file.Close()

			b := new(strings.Builder)
			_, err = io.Copy(b, file)

			in.SourceFileName = trans.Ptr(header.Filename)
			in.Mime = trans.Ptr(header.Header.Get("Content-Type"))
			in.File = []byte(b.String())
		}

		if err = ctx.BindQuery(&in); err != nil {
			return err
		}

		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			var resp *storageV1.UEditorUploadResponse

			resp, err = svc.UEditorUploadFile(ctx, req.(*storageV1.UEditorUploadRequest))
			in.File = nil

			return resp, err
		})

		// 逻辑处理，取数据
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}

		reply := out.(*storageV1.UEditorUploadResponse)

		return ctx.Result(200, reply)
	}
}

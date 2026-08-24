package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go-wind-admin/app/admin/service/internal/data"
	"io"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/minio/minio-go/v7"

	"github.com/tx7do/go-utils/id"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	storageV1 "go-wind-admin/api/gen/go/storage/service/v1"

	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/netutil"
	"go-wind-admin/pkg/oss"
)

type FileTransferService struct {
	adminV1.FileTransferServiceHTTPServer

	log *bLogger.Helper

	mc                *oss.MinIOClient
	fileServiceClient *data.FileRepo
}

func NewFileTransferService(
	ctx *bootstrap.Context,
	mc *oss.MinIOClient,
	fileServiceClient *data.FileRepo,
) *FileTransferService {
	return &FileTransferService{
		log:               ctx.NewLoggerHelper("file-transfer/service/app-service"),
		mc:                mc,
		fileServiceClient: fileServiceClient,
	}
}

// joinObjectPath 拼接 OSS object 的存储路径：目录与文件名之间补 "/"，
// 目录为空或文件名本身已含前导 "/" 时避免重复斜杠。供下载/删除等按元数据重建 key 时使用。
func joinObjectPath(directory, fileName string) string {
	directory = strings.Trim(directory, "/")
	fileName = strings.TrimLeft(fileName, "/")
	if directory == "" {
		return fileName
	}
	return directory + "/" + fileName
}

func parseKey(key string) (folder, filename, ext string) {
	if key == "" {
		return "", "", ""
	}

	// 统一去除前导斜杠，但保留中间路径
	key = strings.TrimPrefix(key, "/")

	// 如果以 '/' 结尾，则视为目录
	if strings.HasSuffix(key, "/") {
		f := strings.TrimSuffix(key, "/")
		return f, "", ""
	}

	// 目录部分
	dir := path.Dir(key)
	if dir == "." {
		dir = ""
	}

	base := path.Base(key)

	// 处理点文件（如 .env）：当且仅当只有一个前导点且没有其他点，视为无扩展名
	if strings.HasPrefix(base, ".") && strings.Count(base, ".") == 1 {
		return dir, base, ""
	}

	// 查找最后一个点作为扩展名分隔（点在开头不算）
	idx := strings.LastIndex(base, ".")
	if idx <= 0 {
		// 无扩展名或点在首位（已处理首位点情况）
		return dir, base, ""
	}

	name := base[:idx]
	ext = strings.ToLower(base[idx+1:])

	return dir, name, ext
}

// recordFile 记录文件元数据到数据库
func (s *FileTransferService) recordFile(
	ctx context.Context,
	tenantID, userID uint32,
	fileData []byte,
	sourceFileName string,
	info minio.UploadInfo,
	downloadUrl string,
) error {

	sum := sha256.Sum256(fileData)          // sha256.Sum256 返回 [32]byte
	sha256Hex := hex.EncodeToString(sum[:]) // 转为十六进制字符串

	dir, fileName, ext := parseKey(info.Key)
	//s.log.Debugf(context.Background(), "Parsed file - Dir: %s, FileName: %s, Ext: %s", dir, fileName, ext)

	if err := s.fileServiceClient.Create(ctx, &storageV1.CreateFileRequest{
		Data: &storageV1.File{
			Provider:      trans.Ptr(storageV1.OSSProvider_MINIO),
			BucketName:    trans.Ptr(info.Bucket),
			SaveFileName:  trans.Ptr(fileName + "." + ext),
			ContentHash:   trans.Ptr(sha256Hex),
			FileDirectory: trans.Ptr(dir),
			FileName:      trans.Ptr(sourceFileName),
			Extension:     trans.Ptr(ext),
			FileGuid:      trans.Ptr(id.NewGUIDv7(false)),
			Size:          trans.Ptr(uint64(info.Size)),
			LinkUrl:       trans.Ptr(downloadUrl),
			CreatedBy:     trans.Ptr(userID),
			TenantId:      trans.Ptr(tenantID),
		},
	}); err != nil {
		s.log.Errorf(context.Background(), "Failed to create file record: %v", err)
		return err
	}
	return nil
}

// directUploadFile 直接上传文件
func (s *FileTransferService) directUploadFile(ctx context.Context, req *storageV1.UploadFileRequest) (*storageV1.UploadFileResponse, error) {
	if req.StorageObject == nil {
		return nil, storageV1.ErrorUploadFailed("unknown storageObject")
	}

	if req.GetFile() == nil {
		return nil, storageV1.ErrorUploadFailed("unknown fileData")
	}

	if req.GetMime() == "" {
		return nil, storageV1.ErrorUploadFailed("unknown mime type")
	}

	if req.GetSourceFileName() == "" {
		return nil, storageV1.ErrorUploadFailed("unknown source file name")
	}

	// H3: 文件大小校验，防 DoS / 超大文件滥用
	if int64(len(req.GetFile())) > oss.MaxUploadSize {
		return nil, storageV1.ErrorFileTooLarge("file size %d exceeds max upload size %d", len(req.GetFile()), oss.MaxUploadSize)
	}

	// H3: 通过文件内容嗅探真实 MIME，避免信任客户端可伪造的 mime 字段
	// （防止把 text/html / 可执行文件伪装成 image/* 放进可被下载/渲染的 bucket）
	realMime, _ := oss.DetectFileType(req.GetFile())
	if !oss.IsAllowedMimeType(realMime) {
		return nil, storageV1.ErrorUnsupportedMediaType("file type %q is not allowed", realMime)
	}
	// 用真实嗅探出的类型覆盖客户端声明，防止 bucket 路由被绕过
	req.Mime = trans.Ptr(realMime)

	// H3: 校验客户端可控的 FileDirectory，拒绝 .. 等路径穿越/命名空间注入
	if !oss.IsFileDirectorySafe(req.GetStorageObject().GetFileDirectory()) {
		return nil, storageV1.ErrorBadRequest("invalid file directory")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.StorageObject.BucketName == nil {
		req.StorageObject.BucketName = trans.Ptr(oss.ContentTypeToBucketName(req.GetMime()))
	}

	if req.StorageObject.ObjectName == nil {
		req.StorageObject.ObjectName = trans.Ptr(
			oss.EnsureObjectName(
				req.GetStorageObject().GetFileDirectory(),
				req.GetSourceFileName(),
				req.GetMime(),
				req.GetFile(),
				oss.GenerateFileNameTypeUUID,
			),
		)
	}

	info, _, downloadUrl, err := s.mc.UploadFile(
		ctx,
		req.GetStorageObject().GetBucketName(),
		req.GetStorageObject().GetObjectName(),
		req.GetMime(),
		req.GetFile(),
	)
	if err != nil {
		return nil, err
	}

	if err = s.recordFile(
		ctx,
		operator.GetTenantId(), operator.GetUserId(),
		req.GetFile(),
		req.GetSourceFileName(),
		info, downloadUrl); err != nil {
		// H3: 元数据落库失败时记录告警。对象已入 OSS 但无 DB 记录（孤儿对象），
		// 这里不掩盖错误，让上层感知以便后续清理。
		s.log.Errorf(ctx, "upload succeeded but failed to record file metadata (orphan object %q in bucket %q): %v",
			info.Key, info.Bucket, err)
		return nil, err
	}

	return &storageV1.UploadFileResponse{
		ObjectName: trans.Ptr(downloadUrl),
	}, err
}

// presignedUploadFile 预签名上传文件
func (s *FileTransferService) presignedUploadFile(ctx context.Context, req *storageV1.UploadFileRequest) (*storageV1.UploadFileResponse, error) {
	// 预签名上传路径已禁用：该路径无法在服务端可靠记录文件元数据
	// （sourceFileName/tenantId/userId/sha256 无法从 MinIO 事件通知获得，
	// 且 x-amz-meta-* 客户端可伪造）。当前业务上传统一走 directUploadFile
	// （服务端中转，已正确落库）。待有预签名直传刚需时，需引入
	// MinIO 事件通知 + 待确认表 + 回调端点 + 定时清理的完整闭环。
	_ = req
	return nil, storageV1.ErrorUploadFailed("presigned upload is not implemented, use direct upload instead")
}

// UploadFile 上传文件
func (s *FileTransferService) UploadFile(ctx context.Context, req *storageV1.UploadFileRequest) (*storageV1.UploadFileResponse, error) {
	switch req.Source.(type) {
	case *storageV1.UploadFileRequest_File:
		return s.directUploadFile(ctx, req)

	case *storageV1.UploadFileRequest_Presign:
		return s.presignedUploadFile(ctx, req)

	default:
		return nil, storageV1.ErrorUploadFailed("unknown upload source")
	}
}

// downloadFileFromURL 从指定的 URL 下载文件内容。
// H2: 内置 SSRF 防护——scheme 白名单、解析后逐 IP 校验阻断内网、
// 自定义 DialContext 钉死 IP 防 DNS rebinding、CheckRedirect 二次校验、
// 限制响应体大小、超时。
func (s *FileTransferService) downloadFileFromURL(ctx context.Context, downloadUrl string) (*storageV1.DownloadFileResponse, error) {
	if downloadUrl == "" {
		return nil, storageV1.ErrorDownloadFailed("empty download url")
	}

	// 1. 静态校验 URL（scheme/host/userinfo）
	u, err := netutil.ValidateURL(downloadUrl)
	if err != nil {
		return nil, storageV1.ErrorDownloadFailed("invalid download url: %s", err.Error())
	}

	// 2. 解析主机名并校验所有解析到的 IP 不在内网
	ips, err := netutil.LookupAndCheckHost(ctx, u.Hostname())
	if err != nil {
		return nil, storageV1.ErrorForbidden("blocked download host: %s", err.Error())
	}
	// 选第一个合法 IP 作为拨号目标（pin IP，防 DNS rebinding）
	pinnedIP := ips[0].String()

	// 3. 构造自定义 http.Client：钉死 IP、限制重定向、超时
	pinnedPort := u.Port()
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		DialContext: func(dialCtx context.Context, network, addr string) (net.Conn, error) {
			// 忽略 addr 中的解析结果，强制使用前面已校验过的 pinnedIP
			var host string
			if pinnedPort != "" {
				host = net.JoinHostPort(pinnedIP, pinnedPort)
			} else {
				// 补默认端口
				if strings.ToLower(u.Scheme) == "https" {
					host = net.JoinHostPort(pinnedIP, "443")
				} else {
					host = net.JoinHostPort(pinnedIP, "80")
				}
			}
			return dialer.DialContext(dialCtx, network, host)
		},
		// 关闭 keepalive，避免长连接复用跨越不同 URL 的拨号上下文
		DisableKeepAlives: true,
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 每次重定向都重新校验目标 URL 与主机
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			if _, rerr := netutil.ValidateURL(req.URL.String()); rerr != nil {
				return fmt.Errorf("redirect to invalid url: %w", rerr)
			}
			if _, rerr := netutil.LookupAndCheckHost(req.Context(), req.URL.Hostname()); rerr != nil {
				return fmt.Errorf("redirect to blocked host: %w", rerr)
			}
			return nil
		},
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", downloadUrl, nil)
	if err != nil {
		return nil, storageV1.ErrorDownloadFailed("%s", err.Error())
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, storageV1.ErrorDownloadFailed("%s", err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, storageV1.ErrorDownloadFailed("%s", "unexpected status: "+resp.Status)
	}

	// 4. 限制响应体大小，防 DoS
	fileData, err := io.ReadAll(io.LimitReader(resp.Body, oss.MaxDownloadSize+1))
	if err != nil {
		return nil, storageV1.ErrorDownloadFailed("read body failed: %s", err.Error())
	}
	if int64(len(fileData)) > oss.MaxDownloadSize {
		return nil, storageV1.ErrorDownloadFailed("remote file exceeds max download size %d", oss.MaxDownloadSize)
	}

	return &storageV1.DownloadFileResponse{
		Content: &storageV1.DownloadFileResponse_File{
			File: fileData,
		},
	}, nil
}

// DownloadFile 下载文件
func (s *FileTransferService) DownloadFile(ctx context.Context, req *storageV1.DownloadFileRequest) (*storageV1.DownloadFileResponse, error) {
	switch req.Selector.(type) {
	case *storageV1.DownloadFileRequest_FileId:
		resp, err := s.fileServiceClient.Get(ctx, &storageV1.GetFileRequest{
			QueryBy: &storageV1.GetFileRequest_Id{Id: req.GetFileId()},
		})
		if err != nil {
			return nil, storageV1.ErrorDownloadFailed("file not found")
		}

		req.Selector = &storageV1.DownloadFileRequest_StorageObject{
			StorageObject: &storageV1.StorageObject{
				BucketName: resp.BucketName,
				// 此前缺少目录与文件名之间的 "/" 分隔，导致拼出的 object key 形如 "dirabc.jpg" 永远 404。
				// 与 file_service.go Delete 处的拼接逻辑保持一致：dir + "/" + name；目录为空时不加前缀。
				ObjectName: trans.Ptr(joinObjectPath(resp.GetFileDirectory(), resp.GetSaveFileName())),
			},
		}

		return s.mc.DownloadFile(ctx, req)

	case *storageV1.DownloadFileRequest_StorageObject:
		return s.mc.DownloadFile(ctx, req)

	case *storageV1.DownloadFileRequest_DownloadUrl:
		return s.downloadFileFromURL(ctx, req.GetDownloadUrl())

	default:
		return nil, storageV1.ErrorDownloadFailed("unknown download selector")
	}
}

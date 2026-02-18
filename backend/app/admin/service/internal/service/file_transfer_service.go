package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	storageV1 "go-wind-admin/api/gen/go/storage/service/v1"

	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/oss"
)

type FileTransferService struct {
	adminV1.FileTransferServiceHTTPServer

	log *log.Helper

	mc       *oss.MinIOClient
	fileRepo *data.FileRepo
}

func NewFileTransferService(
	ctx *bootstrap.Context,
	mc *oss.MinIOClient,
	fileRepo *data.FileRepo,
) *FileTransferService {
	return &FileTransferService{
		log:      ctx.NewLoggerHelper("file-transfer/service/admin-service"),
		mc:       mc,
		fileRepo: fileRepo,
	}
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
	//s.log.Debugf("Parsed file - Dir: %s, FileName: %s, Ext: %s", dir, fileName, ext)

	if err := s.fileRepo.Create(ctx, &storageV1.CreateFileRequest{
		Data: &storageV1.File{
			Provider:      trans.Ptr(storageV1.OSSProvider_MINIO),
			BucketName:    trans.Ptr(info.Bucket),
			SaveFileName:  trans.Ptr(fileName + "." + ext),
			ContentHash:   trans.Ptr(sha256Hex),
			FileDirectory: trans.Ptr(dir),
			FileName:      trans.Ptr(sourceFileName),
			Extension:     trans.Ptr(ext),
			FileGuid:      trans.Ptr(uuid.New().String()),
			Size:          trans.Ptr(uint64(info.Size)),
			LinkUrl:       trans.Ptr(downloadUrl),
			CreatedBy:     trans.Ptr(userID),
			TenantId:      trans.Ptr(tenantID),
		},
	}); err != nil {
		s.log.Errorf("Failed to create file record: %v", err)
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

	info, downloadUrl, err := s.mc.UploadFile(
		ctx,
		req.GetStorageObject().GetBucketName(),
		req.GetStorageObject().GetObjectName(),
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
	}

	return &storageV1.UploadFileResponse{
		ObjectName: trans.Ptr(downloadUrl),
	}, err
}

// presignedUploadFile 预签名上传文件
func (s *FileTransferService) presignedUploadFile(ctx context.Context, req *storageV1.UploadFileRequest) (*storageV1.UploadFileResponse, error) {
	if req.StorageObject == nil {
		return nil, storageV1.ErrorUploadFailed("unknown storageObject")
	}

	if req.GetPresign() == nil {
		return nil, storageV1.ErrorUploadFailed("unknown presign data")
	}

	var contentType string
	if req.GetPresign().GetContentType() != "" {
		contentType = req.GetPresign().GetContentType()
	} else if req.GetMime() != "" {
		contentType = req.GetMime()
	} else {
		return nil, storageV1.ErrorUploadFailed("unknown content type for presign")
	}

	if req.GetSourceFileName() == "" {
		return nil, storageV1.ErrorUploadFailed("unknown source file name")
	}

	if req.StorageObject.BucketName == nil {
		req.StorageObject.BucketName = trans.Ptr(oss.ContentTypeToBucketName(contentType))
	}
	if req.StorageObject.ObjectName == nil {
		req.StorageObject.ObjectName = trans.Ptr(
			oss.EnsureObjectName(
				req.GetStorageObject().GetFileDirectory(),
				req.GetSourceFileName(),
				contentType,
				req.GetFile(),
				oss.GenerateFileNameTypeUUID,
			),
		)
	}

	var method storageV1.GetUploadPresignedUrlRequest_Method
	switch strings.ToLower(req.GetPresign().GetMethod()) {
	case "put":
		method = storageV1.GetUploadPresignedUrlRequest_Put
	case "post":
		method = storageV1.GetUploadPresignedUrlRequest_Post
	default:
		method = storageV1.GetUploadPresignedUrlRequest_Post
	}

	resp, err := s.mc.GetUploadPresignedUrl(
		ctx,
		&storageV1.GetUploadPresignedUrlRequest{
			ContentType:   trans.Ptr(contentType),
			ExpireSeconds: req.GetPresign().ExpireSeconds,
			Method:        method,
			BucketName:    req.StorageObject.BucketName,
			FileDirectory: req.StorageObject.FileDirectory,
		})
	if err != nil {
		return nil, err
	}

	// TODO : 记录文件元数据到数据库（待上传完成后再记录更合适？）

	return &storageV1.UploadFileResponse{
		PresignedUrl: trans.Ptr(resp.UploadUrl),
	}, nil
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

// downloadFileFromURL 从指定的 URL 下载文件内容
func (s *FileTransferService) downloadFileFromURL(ctx context.Context, downloadUrl string) (*storageV1.DownloadFileResponse, error) {
	if downloadUrl == "" {
		return nil, storageV1.ErrorDownloadFailed("empty download url")
	}

	// 如果需要支持断点续传，可在此构造请求并设置 Range 头
	httpReq, err := http.NewRequestWithContext(ctx, "GET", downloadUrl, nil)
	if err != nil {
		return nil, storageV1.ErrorDownloadFailed(err.Error())
	}
	// 示例：如果你要设置 Range（可选）
	// httpReq.Header.Set("Range", "bytes=100-")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, storageV1.ErrorDownloadFailed(err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, storageV1.ErrorDownloadFailed("unexpected status: " + resp.Status)
	}

	fileData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, storageV1.ErrorDownloadFailed(err.Error())
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
		resp, err := s.fileRepo.Get(ctx, &storageV1.GetFileRequest{
			QueryBy: &storageV1.GetFileRequest_Id{Id: req.GetFileId()},
		})
		if err != nil {
			return nil, storageV1.ErrorDownloadFailed("file not found")
		}

		req.Selector = &storageV1.DownloadFileRequest_StorageObject{
			StorageObject: &storageV1.StorageObject{
				BucketName: resp.BucketName,
				ObjectName: trans.Ptr(resp.GetFileDirectory() + resp.GetSaveFileName()),
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

func (s *FileTransferService) UEditorUploadFile(ctx context.Context, req *storageV1.UEditorUploadRequest) (*storageV1.UEditorUploadResponse, error) {
	//s.log.Infof("上传文件： %s", req.GetFile())

	if req.File == nil {
		return nil, storageV1.ErrorUploadFailed("unknown file")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	var bucketName string
	switch req.GetAction() {
	default:
		fallthrough
	case storageV1.UEditorAction_uploadFile.String():
		bucketName = "files"
	case storageV1.UEditorAction_uploadImage.String(), storageV1.UEditorAction_uploadScrawl.String(), storageV1.UEditorAction_catchImage.String():
		bucketName = "images"
	case storageV1.UEditorAction_uploadVideo.String():
		bucketName = "videos"
	}

	info, downloadUrl, err := s.mc.UploadFile(ctx, bucketName, req.GetSourceFileName(), req.GetFile())
	if err != nil {
		return &storageV1.UEditorUploadResponse{
			State: trans.Ptr(err.Error()),
		}, err
	}

	if err = s.recordFile(
		ctx,
		operator.GetTenantId(), operator.GetUserId(),
		req.GetFile(),
		req.GetSourceFileName(),
		info, downloadUrl); err != nil {
	}

	return &storageV1.UEditorUploadResponse{
		State:    trans.Ptr(StateOK),
		Original: trans.Ptr(req.GetSourceFileName()),
		Title:    trans.Ptr(req.GetSourceFileName()),
		Url:      trans.Ptr(downloadUrl),
		Size:     trans.Ptr(int32(len(req.GetFile()))),
		Type:     trans.Ptr(path.Ext(req.GetSourceFileName())),
	}, nil
}

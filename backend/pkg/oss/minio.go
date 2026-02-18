package oss

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/minio/minio-go/v7"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"
	ossMinio "github.com/tx7do/kratos-bootstrap/oss/minio"

	storageV1 "go-wind-admin/api/gen/go/storage/service/v1"
)

const (
	defaultExpiryTime = time.Minute * 60 // 默认的预签名时间，默认为：1小时
)

// MinIOClient MinIO 客户端封装
type MinIOClient struct {
	mc         *minio.Client
	conf       *conf.OSS
	log        *log.Helper
	hmacSecret []byte
}

func NewMinIoClient(cfg *conf.Bootstrap, logger log.Logger) *MinIOClient {
	l := log.NewHelper(log.With(logger, "module", "minio/data/admin-service"))
	return &MinIOClient{
		log:        l,
		conf:       cfg.Oss,
		mc:         ossMinio.NewClient(cfg.Oss),
		hmacSecret: staticHMACSecret,
	}
}

// GetClient returns the underlying MinIO client
func (c *MinIOClient) GetClient() *minio.Client {
	return c.mc
}

// BucketExists Check if the specified bucket exists
func (c *MinIOClient) BucketExists(ctx context.Context, bucketName string) (exists bool, err error) {
	exists, err = c.mc.BucketExists(ctx, bucketName)
	if err != nil {
		c.log.Errorf("Failed to check bucket existence: %v", err)
		return false, storageV1.ErrorInternalServerError("failed to check bucket existence: %s", bucketName)
	}
	return exists, nil
}

// MakeBucket Create a new bucket
func (c *MinIOClient) MakeBucket(ctx context.Context, bucketName string) (err error) {
	err = c.mc.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		c.log.Errorf("Failed to create bucket: %v", err)
		return storageV1.ErrorInternalServerError("failed to create bucket: %s", bucketName)
	}
	c.log.Infof("Created bucket: %s", bucketName)

	return nil
}

// EnsureBucketExists Ensure that the specified bucket exists
func (c *MinIOClient) EnsureBucketExists(ctx context.Context, bucketName string) error {
	exists, err := c.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}

	if !exists {
		return c.MakeBucket(ctx, bucketName)
	}

	return nil
}

// GetUploadPresignedUrl 获取上传地址
func (c *MinIOClient) GetUploadPresignedUrl(ctx context.Context, req *storageV1.GetUploadPresignedUrlRequest) (*storageV1.GetUploadPresignedUrlResponse, error) {
	var bucketName string
	if req.BucketName != nil {
		bucketName = req.GetBucketName()
	} else {
		bucketName = ContentTypeToBucketName(req.GetContentType())
	}
	if bucketName == "" {
		bucketName = BucketFiles
	}

	objectName, _ := JoinObjectName(req.GetContentType(), req.FileDirectory, req.FileName)

	expiry := defaultExpiryTime
	if req.ExpireSeconds != nil {
		expiry = time.Second * time.Duration(req.GetExpireSeconds())
	}

	if err := c.EnsureBucketExists(ctx, bucketName); err != nil {
		return nil, err
	}

	var uploadUrl string
	var downloadUrl string
	var formData map[string]string

	var err error
	var presignedURL *url.URL

	switch req.GetMethod() {
	case storageV1.GetUploadPresignedUrlRequest_Put:
		presignedURL, err = c.mc.PresignedPutObject(ctx, bucketName, objectName, expiry)
		if err != nil {
			c.log.Errorf("Failed to generate presigned PUT policy: %v", err)
			return nil, storageV1.ErrorUploadFailed("failed to generate presigned PUT policy")
		}

		uploadUrl = presignedURL.String()
		uploadUrl = ReplaceEndpointHost(downloadUrl, c.conf.Minio.UploadHost, c.conf.Minio.Endpoint)

		downloadUrl = JoinObjectUrl(presignedURL.Host, bucketName, objectName)
		downloadUrl = ReplaceEndpointHost(downloadUrl, c.conf.Minio.DownloadHost, c.conf.Minio.Endpoint)
		if !strings.HasPrefix(downloadUrl, presignedURL.Scheme) {
			downloadUrl = presignedURL.Scheme + "://" + downloadUrl
		}

	case storageV1.GetUploadPresignedUrlRequest_Post:
		policy := minio.NewPostPolicy()
		_ = policy.SetBucket(bucketName)
		_ = policy.SetKey(objectName)
		_ = policy.SetExpires(time.Now().UTC().Add(expiry))
		_ = policy.SetContentType(req.GetContentType())

		presignedURL, formData, err = c.mc.PresignedPostPolicy(ctx, policy)
		if err != nil {
			c.log.Errorf("Failed to generate presigned POST policy: %v", err)
			return nil, storageV1.ErrorUploadFailed("failed to generate presigned POST policy")
		}

		uploadUrl = presignedURL.String()
		uploadUrl = ReplaceEndpointHost(downloadUrl, c.conf.Minio.UploadHost, c.conf.Minio.Endpoint)

		downloadUrl = JoinObjectUrl(presignedURL.Host, bucketName, objectName)
		uploadUrl = ReplaceEndpointHost(downloadUrl, c.conf.Minio.DownloadHost, c.conf.Minio.Endpoint)
		if !strings.HasPrefix(downloadUrl, presignedURL.Scheme) {
			downloadUrl = presignedURL.Scheme + "://" + downloadUrl
		}
	}

	return &storageV1.GetUploadPresignedUrlResponse{
		UploadUrl:   uploadUrl,
		DownloadUrl: downloadUrl,
		ObjectName:  objectName,
		BucketName:  trans.Ptr(bucketName),
		FormData:    formData,
	}, nil
}

// ListFile 获取文件夹下面的文件列表
func (c *MinIOClient) ListFile(ctx context.Context, req *storageV1.ListOssFileRequest) (*storageV1.ListOssFileResponse, error) {
	resp := &storageV1.ListOssFileResponse{
		Files: make([]string, 0),
	}
	for object := range c.mc.ListObjects(ctx,
		req.GetBucketName(),
		minio.ListObjectsOptions{
			Prefix:    req.GetFolder(),
			Recursive: req.GetRecursive(),
		},
	) {
		//fmt.Printf("%+v\n", object)
		resp.Files = append(resp.Files, object.Key)
	}
	return resp, nil
}

// ListFileForUEditor 获取文件夹下面的文件列表
func (c *MinIOClient) ListFileForUEditor(ctx context.Context, bucketName string, folder string) (*storageV1.UEditorResponse, error) {
	resp := &storageV1.UEditorResponse{
		State: trans.Ptr("SUCCESS"),
		List:  make([]*storageV1.UEditorResponse_Item, 0),
	}
	for object := range c.mc.ListObjects(ctx,
		bucketName,
		minio.ListObjectsOptions{
			Prefix:    folder,
			Recursive: true,
		},
	) {
		//fmt.Printf("%+v\n", object)
		resp.List = append(resp.List, &storageV1.UEditorResponse_Item{
			Url:   "/" + bucketName + "/" + folder + object.Key,
			Mtime: object.LastModified.Unix(),
		})
	}

	resp.Start = trans.Ptr(int32(0))
	resp.Total = trans.Ptr(int32(len(resp.List)))

	return resp, nil
}

// DeleteFile 删除一个文件
func (c *MinIOClient) DeleteFile(ctx context.Context, bucketName, objectName string) error {
	if bucketName == "" {
		return storageV1.ErrorBadRequest("bucket name is required")
	}
	if objectName == "" {
		return storageV1.ErrorBadRequest("object name is required")
	}

	err := c.mc.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		c.log.Errorf("Failed to delete file: %v", err)
		return storageV1.ErrorDeleteFailed("failed to delete file")
	}

	return nil
}

// UploadFile 上传文件
func (c *MinIOClient) UploadFile(ctx context.Context, bucketName string, objectName string, fileContent []byte) (minio.UploadInfo, string, error) {
	if len(fileContent) == 0 {
		c.log.Errorf("Empty fileContent data")
		return minio.UploadInfo{}, "", storageV1.ErrorUploadFailed("empty fileContent data")
	}

	if bucketName == "" {
		bucketName = BucketFiles
	}
	if objectName == "" {
		mimeType, ext := DetectFileType(fileContent)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		if ext == "" {
			ext = ".bin"
		}

		bucketName = ContentTypeToBucketName(mimeType)
		objectName = GenerateObjectName("", fileContent, ext, GenerateFileNameTypeUUID)
	}
	if err := c.EnsureBucketExists(ctx, bucketName); err != nil {
		return minio.UploadInfo{}, "", err
	}

	reader := bytes.NewReader(fileContent)
	if reader == nil {
		c.log.Errorf("Invalid fileContent data")
		return minio.UploadInfo{}, "", storageV1.ErrorUploadFailed("invalid fileContent data")
	}

	info, err := c.mc.PutObject(
		ctx,
		bucketName, objectName,
		reader, reader.Size(),
		minio.PutObjectOptions{},
	)
	if err != nil {
		c.log.Errorf("Failed to upload fileContent: %v", err)
		return info, "", storageV1.ErrorUploadFailed("failed to upload fileContent")
	}

	downloadUrl := JoinObjectUrl("", bucketName, objectName)

	return info, downloadUrl, nil
}

// getDownloadUrlWithStorageObjectDirect 直接获取文件内容
func (c *MinIOClient) getDownloadUrlWithStorageObjectDirect(ctx context.Context, req *storageV1.GetDownloadInfoRequest) (*storageV1.GetDownloadInfoResponse, error) {
	opts := minio.GetObjectOptions{}

	SetDownloadRange(&opts, req.RangeStart, req.RangeEnd)

	object, err := c.mc.GetObject(
		ctx,
		req.GetStorageObject().GetBucketName(),
		req.GetStorageObject().GetObjectName(),
		opts,
	)
	if err != nil {
		c.log.Errorf("Failed to get object: %v", err)
		return nil, storageV1.ErrorDownloadFailed("failed to get object")
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(object)
	if err != nil {
		c.log.Errorf("Failed to read object: %v", err)
		return nil, storageV1.ErrorDownloadFailed("failed to read object")
	}

	resp := &storageV1.GetDownloadInfoResponse{
		Content: &storageV1.GetDownloadInfoResponse_File{
			File: buf.Bytes(),
		},
	}

	st, err := object.Stat()
	if err != nil {
		c.log.Errorf("Failed to stat object: %v", err)
		return nil, storageV1.ErrorDownloadFailed("failed to stat object")
	}

	if req.GetAcceptMime() != "" {
		resp.Mime = req.GetAcceptMime()
	} else {
		resp.Mime = st.ContentType
	}
	if resp.GetMime() == "" {
		resp.Mime = "application/octet-stream"
	}

	resp.Checksum = st.ChecksumSHA256
	resp.SourceFileName = st.Key
	resp.Size = st.Size
	resp.UpdatedAt = timeutil.TimeToTimestamppb(&st.LastModified)

	return resp, nil
}

// getDownloadUrlWithStorageObjectPresigned 获取预签名下载地址
func (c *MinIOClient) getDownloadUrlWithStorageObjectPresigned(ctx context.Context, req *storageV1.GetDownloadInfoRequest) (*storageV1.GetDownloadInfoResponse, error) {
	expires := defaultExpiryTime
	if req.PresignExpireSeconds != nil {
		expires = time.Second * time.Duration(req.GetPresignExpireSeconds())
	}
	presignedURL, err := c.mc.PresignedGetObject(
		ctx,
		req.GetStorageObject().GetBucketName(),
		req.GetStorageObject().GetObjectName(),
		expires,
		nil,
	)
	if err != nil {
		c.log.Errorf("Failed to generate presigned URL: %v", err)
		return nil, storageV1.ErrorDownloadFailed("failed to generate presigned URL")
	}

	downloadUrl := presignedURL.String()
	downloadUrl = ReplaceEndpointHost(downloadUrl, c.conf.Minio.DownloadHost, c.conf.Minio.Endpoint)
	if !strings.HasPrefix(downloadUrl, presignedURL.Scheme) {
		downloadUrl = presignedURL.Scheme + "://" + downloadUrl
	}

	return &storageV1.GetDownloadInfoResponse{
		Content: &storageV1.GetDownloadInfoResponse_DownloadUrl{
			DownloadUrl: downloadUrl,
		},
	}, nil
}

// GetDownloadUrl 获取下载地址
func (c *MinIOClient) GetDownloadUrl(ctx context.Context, req *storageV1.GetDownloadInfoRequest) (*storageV1.GetDownloadInfoResponse, error) {
	switch req.Selector.(type) {
	case *storageV1.GetDownloadInfoRequest_StorageObject:
		if req.GetPreferPresignedUrl() {
			return c.getDownloadUrlWithStorageObjectPresigned(ctx, req)
		} else {
			return c.getDownloadUrlWithStorageObjectDirect(ctx, req)
		}

	case *storageV1.GetDownloadInfoRequest_FileId:
		return nil, storageV1.ErrorNotImplemented("not implemented yet")

	default:
		return nil, storageV1.ErrorBadRequest("invalid selector")
	}
}

// downloadFileWithStorageObjectDirect 直接获取文件内容
func (c *MinIOClient) downloadFileWithStorageObjectDirect(ctx context.Context, req *storageV1.DownloadFileRequest) (*storageV1.DownloadFileResponse, error) {
	opts := minio.GetObjectOptions{}

	SetDownloadRange(&opts, req.RangeStart, req.RangeEnd)

	object, err := c.mc.GetObject(
		ctx,
		req.GetStorageObject().GetBucketName(),
		req.GetStorageObject().GetObjectName(),
		opts,
	)
	if err != nil {
		c.log.Errorf("Failed to get object: %v", err)
		return nil, storageV1.ErrorDownloadFailed("failed to get object")
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(object)
	if err != nil {
		c.log.Errorf("Failed to read object: %v", err)
		return nil, storageV1.ErrorDownloadFailed("failed to read object")
	}

	resp := &storageV1.DownloadFileResponse{
		Content: &storageV1.DownloadFileResponse_File{
			File: buf.Bytes(),
		},
	}

	st, err := object.Stat()
	if err != nil {
		c.log.Errorf("Failed to stat object: %v", err)
		return nil, storageV1.ErrorDownloadFailed("failed to stat object")
	}

	if req.GetAcceptMime() != "" {
		resp.Mime = req.GetAcceptMime()
	} else {
		resp.Mime = st.ContentType
	}
	if resp.GetMime() == "" {
		resp.Mime = "application/octet-stream"
	}

	resp.Checksum = st.ChecksumSHA256
	resp.SourceFileName = st.Key
	resp.Size = st.Size
	resp.UpdatedAt = timeutil.TimeToTimestamppb(&st.LastModified)

	return resp, nil
}

// downloadFileWithStorageObjectPresigned 获取预签名下载地址
func (c *MinIOClient) downloadFileWithStorageObjectPresigned(ctx context.Context, req *storageV1.DownloadFileRequest) (*storageV1.DownloadFileResponse, error) {
	expires := defaultExpiryTime
	if req.PresignExpireSeconds != nil {
		expires = time.Second * time.Duration(req.GetPresignExpireSeconds())
	}
	presignedURL, err := c.mc.PresignedGetObject(
		ctx,
		req.GetStorageObject().GetBucketName(),
		req.GetStorageObject().GetObjectName(),
		expires,
		nil,
	)
	if err != nil {
		c.log.Errorf("Failed to generate presigned URL: %v", err)
		return nil, storageV1.ErrorDownloadFailed("failed to generate presigned URL")
	}

	downloadUrl := presignedURL.String()
	downloadUrl = ReplaceEndpointHost(downloadUrl, c.conf.Minio.DownloadHost, c.conf.Minio.Endpoint)
	if !strings.HasPrefix(downloadUrl, presignedURL.Scheme) {
		downloadUrl = presignedURL.Scheme + "://" + downloadUrl
	}

	return &storageV1.DownloadFileResponse{
		Content: &storageV1.DownloadFileResponse_DownloadUrl{
			DownloadUrl: downloadUrl,
		},
	}, nil
}

// DownloadFile 下载文件
func (c *MinIOClient) DownloadFile(ctx context.Context, req *storageV1.DownloadFileRequest) (*storageV1.DownloadFileResponse, error) {
	switch req.Selector.(type) {
	case *storageV1.DownloadFileRequest_StorageObject:
		if req.GetPreferPresignedUrl() {
			return c.downloadFileWithStorageObjectPresigned(ctx, req)
		} else {
			return c.downloadFileWithStorageObjectDirect(ctx, req)
		}

	case *storageV1.DownloadFileRequest_FileId:
		return nil, storageV1.ErrorNotImplemented("not implemented yet")

	default:
		return nil, storageV1.ErrorBadRequest("invalid selector")
	}
}

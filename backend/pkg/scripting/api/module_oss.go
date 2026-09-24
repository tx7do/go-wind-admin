package api

import (
	"context"
	"errors"
	"time"

	"github.com/minio/minio-go/v7"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/pkg/oss"
)

// errInvalid 构造脚本侧参数错误（goja 桥接后转 JS 异常）。
func errInvalid(msg string) error { return errors.New(msg) }

// ModuleOSS 构建语言无关的 oss 模块（JS 等基于 map[string]any 桥接的语言使用）。
// 与 Lua 版 LoaderOSS 功能对齐；client 为 nil 时返回不含函数的空模块。
func ModuleOSS(ossClient *oss.MinIOClient, logger *bLogger.Helper) ModuleDef {
	if ossClient == nil {
		return ModuleDef{Name: "oss", Funcs: map[string]any{}}
	}

	bg := context.Background()
	logError := func(op string, err error) {
		if logger != nil {
			logger.Errorf(bg, "oss.%s error: %v", op, err)
		}
	}

	return ModuleDef{
		Name: "oss",
		Funcs: map[string]any{
			// uploadUrl(options) → {upload_url, download_url, object_name, bucket_name}
			// options: {content_type, file_name?, file_path?, bucket_name?}
			"uploadUrl": func(options map[string]any) (map[string]any, error) {
				contentType, _ := options["content_type"].(string)
				if contentType == "" {
					return nil, errInvalid("content_type is required")
				}

				fileName, _ := options["file_name"].(string)
				filePath, _ := options["file_path"].(string)
				bucketName, _ := options["bucket_name"].(string)

				finalBucketName := bucketName
				if finalBucketName == "" {
					finalBucketName = oss.ContentTypeToBucketName(contentType)
				}

				// 与 Lua 版语义一致：空值传 nil，由 JoinObjectName 生成默认名
				var fileNamePtr, filePathPtr *string
				if fileName != "" {
					fileNamePtr = &fileName
				}
				if filePath != "" {
					filePathPtr = &filePath
				}
				objectName, _ := oss.JoinObjectName(contentType, filePathPtr, fileNamePtr)

				presignedURL, err := ossClient.GetClient().PresignedPutObject(bg, finalBucketName, objectName, time.Hour)
				if err != nil {
					logError("upload_url", err)
					return nil, err
				}

				return map[string]any{
					"upload_url":   presignedURL.String(),
					"download_url": "/" + finalBucketName + "/" + objectName,
					"object_name":  objectName,
					"bucket_name":  finalBucketName,
				}, nil
			},
			// listFiles(options) → [{key, size, last_modified, etag}]
			// options: {bucket_name, folder?, recursive?}
			"listFiles": func(options map[string]any) ([]map[string]any, error) {
				bucketName, _ := options["bucket_name"].(string)
				if bucketName == "" {
					return nil, errInvalid("bucket_name is required")
				}
				folder, _ := options["folder"].(string)
				recursive, _ := options["recursive"].(bool)

				objectCh := ossClient.GetClient().ListObjects(bg, bucketName, minio.ListObjectsOptions{
					Prefix:    folder,
					Recursive: recursive,
				})

				files := make([]map[string]any, 0)
				for object := range objectCh {
					if object.Err != nil {
						logError("list_files", object.Err)
						continue
					}
					files = append(files, map[string]any{
						"key":           object.Key,
						"size":          object.Size,
						"last_modified": object.LastModified.Unix(),
						"etag":          trimEtag(object.ETag),
					})
				}
				return files, nil
			},
			// deleteFile(bucketName, objectName) → bool
			"deleteFile": func(bucketName, objectName string) (bool, error) {
				if err := ossClient.GetClient().RemoveObject(bg, bucketName, objectName, minio.RemoveObjectOptions{}); err != nil {
					logError("delete_file", err)
					return false, err
				}
				return true, nil
			},
			// uploadFile(bucketName, objectName, content) → downloadURL
			"uploadFile": func(bucketName, objectName, content string) (string, error) {
				_, _, downloadURL, err := ossClient.UploadFile(bg, bucketName, objectName, "", []byte(content))
				if err != nil {
					logError("upload_file", err)
					return "", err
				}
				return downloadURL, nil
			},
			// ensureBucket(bucketName) → bool
			"ensureBucket": func(bucketName string) (bool, error) {
				if err := ossClient.EnsureBucketExists(bg, bucketName); err != nil {
					logError("ensure_bucket", err)
					return false, err
				}
				return true, nil
			},
			// getBucketForType(contentType) → string
			"getBucketForType": func(contentType string) string {
				return oss.ContentTypeToBucketName(contentType)
			},
		},
	}
}

func trimEtag(etag string) string {
	if len(etag) >= 2 && etag[0] == '"' && etag[len(etag)-1] == '"' {
		return etag[1 : len(etag)-1]
	}
	return etag
}

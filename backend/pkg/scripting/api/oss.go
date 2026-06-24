package api

import (
	"context"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/minio/minio-go/v7"
	lua "github.com/yuin/gopher-lua"

	"go-wind-admin/pkg/oss"
)

// RegisterOSS registers the OSS (Object Storage Service) API for Lua as a requireable module
func RegisterOSS(L *lua.LState, ossClient *oss.MinIOClient, logger *log.Helper) {
	// Register in package.preload so it can be required
	L.PreloadModule("kratos_oss", LoaderOSS(ossClient, logger))
}

// LoaderOSS 返回 oss 模块（kratos_oss）的 loader，供 go-scripts 引擎 RegisterModule 使用。
// ossClient 为 nil 时返回空模块。
func LoaderOSS(ossClient *oss.MinIOClient, logger *log.Helper) lua.LGFunction {
	return func(L *lua.LState) int {
		if ossClient == nil {
			L.Push(L.NewTable())
			return 1
		}
		// Create oss module
		ossModule := L.NewTable()

		// oss.upload_url(options)
		// Get presigned upload URL
		// options: {
		//   content_type = "image/jpeg",
		//   file_name = "photo.jpg",      -- optional
		//   file_path = "uploads/images",  -- optional
		//   bucket_name = "images",        -- optional
		//   method = "PUT"                 -- optional, "PUT" or "POST"
		// }
		// Returns: {upload_url, download_url, object_name, bucket_name}
		ossModule.RawSetString("upload_url", L.NewFunction(func(L *lua.LState) int {
			options := L.CheckTable(1)

			// Extract options
			contentType := options.RawGetString("content_type").String()
			if contentType == "" {
				L.RaiseError("content_type is required")
				return 0
			}

			var fileName *string
			if fn := options.RawGetString("file_name"); fn != lua.LNil {
				s := fn.String()
				fileName = &s
			}

			var filePath *string
			if fp := options.RawGetString("file_path"); fp != lua.LNil {
				s := fp.String()
				filePath = &s
			}

			var bucketName *string
			if bn := options.RawGetString("bucket_name"); bn != lua.LNil {
				s := bn.String()
				bucketName = &s
			}

			// Determine bucket name if not provided
			var finalBucketName string
			if bucketName != nil {
				finalBucketName = *bucketName
			} else {
				finalBucketName = oss.ContentTypeToBucketName(contentType)
			}

			// Generate object name
			objectName, _ := oss.JoinObjectName(contentType, filePath, fileName)

			// Get presigned URL using the underlying MinIO client
			ctx := context.Background()
			presignedURL, err := ossClient.GetClient().PresignedPutObject(ctx, finalBucketName, objectName, time.Hour)
			if err != nil {
				L.RaiseError("failed to get presigned URL: %v", err)
				return 0
			}

			// Build download URL
			downloadURL := "/" + finalBucketName + "/" + objectName

			// Create result table
			result := L.NewTable()
			result.RawSetString("upload_url", lua.LString(presignedURL.String()))
			result.RawSetString("download_url", lua.LString(downloadURL))
			result.RawSetString("object_name", lua.LString(objectName))
			result.RawSetString("bucket_name", lua.LString(finalBucketName))

			L.Push(result)
			return 1
		}))

		// oss.list_files(options)
		// List files in a bucket
		// options: {
		//   bucket_name = "images",
		//   folder = "uploads/",     -- optional
		//   recursive = true         -- optional, default false
		// }
		// Returns: array of file paths
		ossModule.RawSetString("list_files", L.NewFunction(func(L *lua.LState) int {
			options := L.CheckTable(1)

			bucketName := options.RawGetString("bucket_name").String()
			if bucketName == "" {
				L.RaiseError("bucket_name is required")
				return 0
			}

			folder := ""
			if f := options.RawGetString("folder"); f != lua.LNil {
				folder = f.String()
			}

			recursive := false
			if r := options.RawGetString("recursive"); r != lua.LNil {
				recursive = lua.LVAsBool(r)
			}

			ctx := context.Background()
			files := L.NewTable()
			idx := 1

			// Use MinIO's ListObjects with proper options
			objectCh := ossClient.GetClient().ListObjects(ctx, bucketName, minio.ListObjectsOptions{
				Prefix:    folder,
				Recursive: recursive,
			})

			for object := range objectCh {
				if object.Err != nil {
					logger.Errorf("Error listing objects: %v", object.Err)
					continue
				}

				// Create file info table
				fileInfo := L.NewTable()
				fileInfo.RawSetString("key", lua.LString(object.Key))
				fileInfo.RawSetString("size", lua.LNumber(object.Size))
				fileInfo.RawSetString("last_modified", lua.LNumber(object.LastModified.Unix()))
				fileInfo.RawSetString("etag", lua.LString(strings.Trim(object.ETag, "\"")))

				files.RawSetInt(idx, fileInfo)
				idx++
			}

			L.Push(files)
			return 1
		}))

		// oss.delete_file(bucket_name, object_name)
		// Delete a file from OSS
		ossModule.RawSetString("delete_file", L.NewFunction(func(L *lua.LState) int {
			bucketName := L.CheckString(1)
			objectName := L.CheckString(2)

			ctx := context.Background()
			err := ossClient.GetClient().RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
			if err != nil {
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LBool(true))
			return 1
		}))

		// oss.upload_file(bucket_name, object_name, content)
		// Upload file content to OSS
		// content should be a string (file bytes)
		ossModule.RawSetString("upload_file", L.NewFunction(func(L *lua.LState) int {
			bucketName := L.CheckString(1)
			objectName := L.CheckString(2)
			content := L.CheckString(3)

			ctx := context.Background()
			_, _, downloadURL, err := ossClient.UploadFile(ctx, bucketName, objectName, "", []byte(content))
			if err != nil {
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LBool(true))
			L.Push(lua.LString(downloadURL))
			return 2
		}))

		// oss.ensure_bucket(bucket_name)
		// Ensure a bucket exists, create if it doesn't
		ossModule.RawSetString("ensure_bucket", L.NewFunction(func(L *lua.LState) int {
			bucketName := L.CheckString(1)

			ctx := context.Background()
			err := ossClient.EnsureBucketExists(ctx, bucketName)
			if err != nil {
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LBool(true))
			return 1
		}))

		// oss.get_bucket_for_type(content_type)
		// Get the appropriate bucket name for a content type
		ossModule.RawSetString("get_bucket_for_type", L.NewFunction(func(L *lua.LState) int {
			contentType := L.CheckString(1)
			bucketName := oss.ContentTypeToBucketName(contentType)
			L.Push(lua.LString(bucketName))
			return 1
		}))

		L.Push(ossModule)
		return 1
	}
}

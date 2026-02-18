package oss

import (
	"fmt"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/tx7do/go-utils/trans"

	storageV1 "go-wind-admin/api/gen/go/storage/service/v1"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"
)

func createTestClient() *MinIOClient {
	return NewMinIoClient(&conf.Bootstrap{
		Oss: &conf.OSS{
			Minio: &conf.OSS_MinIO{
				Endpoint:     "127.0.0.1:9000",
				UploadHost:   "127.0.0.1:9000",
				DownloadHost: "127.0.0.1:9000",
				AccessKey:    "root",
				SecretKey:    "*Abcd123456",
			},
		},
	}, log.DefaultLogger)
}

func TestMinIoClient(t *testing.T) {
	cli := createTestClient()
	assert.NotNil(t, cli)

	resp, err := cli.GetUploadPresignedUrl(t.Context(), &storageV1.GetUploadPresignedUrlRequest{
		Method:        storageV1.GetUploadPresignedUrlRequest_Put,
		ContentType:   trans.String("image/jpeg"),
		BucketName:    trans.String("images"),
		FileDirectory: trans.String("20221010"),
	})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func TestListFile(t *testing.T) {
	cli := createTestClient()
	assert.NotNil(t, cli)

	req := &storageV1.ListOssFileRequest{
		BucketName: trans.Ptr("users"),
		Folder:     trans.Ptr("1"),
		Recursive:  trans.Ptr(true),
	}
	files, err := cli.ListFile(t.Context(), req)
	assert.Nil(t, err)
	fmt.Println(files)
}

func TestDownloadFile(t *testing.T) {
	cli := createTestClient()
	assert.NotNil(t, cli)

	resp, err := cli.DownloadFile(t.Context(), &storageV1.DownloadFileRequest{
		Selector: &storageV1.DownloadFileRequest_StorageObject{
			StorageObject: &storageV1.StorageObject{
				BucketName: trans.Ptr("images"),
				ObjectName: trans.Ptr("DateTimePicker.png"),
			},
		},
		PreferPresignedUrl: trans.Ptr(false),
	})
	if err != nil {
		t.Error(err)
	}
	fmt.Println(resp)
}

package models

import "github.com/tx7do/go-crud/gorm/mixin"

// File 对应表 files
type File struct {
	mixin.AutoIncrementID64

	Provider      *string `gorm:"column:provider;type:varchar(32);comment:OSS供应商"`
	BucketName    *string `gorm:"column:bucket_name;type:varchar(255);comment:存储桶名称"`
	FileDirectory *string `gorm:"column:file_directory;type:varchar(255);comment:文件目录"`
	FileGUID      *string `gorm:"column:file_guid;type:varchar(128);comment:文件Guid"`
	SaveFileName  *string `gorm:"column:save_file_name;type:varchar(255);comment:保存文件名"`
	FileName      *string `gorm:"column:file_name;type:varchar(255);comment:文件名"`
	Extension     *string `gorm:"column:extension;type:varchar(64);comment:文件扩展名"`
	Size          *uint64 `gorm:"column:size;type:bigint unsigned;comment:文件字节长度"`
	SizeFormat    *string `gorm:"column:size_format;type:varchar(64);comment:文件大小格式化"`
	LinkURL       *string `gorm:"column:link_url;type:varchar(1024);comment:链接地址"`
	ContentHash   *string `gorm:"column:content_hash;type:varchar(255);comment:文件内容哈希"`

	mixin.TimeAt
	mixin.OperatorID64
	mixin.Remark
	mixin.TenantID64
}

// TableName 指定表名
func (File) TableName() string {
	return "files"
}

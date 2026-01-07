package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// File holds the schema definition for the File entity.
type File struct {
	ent.Schema
}

func (File) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "files",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("文件表"),
	}
}

// Fields of the File.
func (File) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("provider").
			Comment("OSS供应商").
			NamedValues(
				"Unknown", "UNKNOWN",
				"MinIO", "MINIO",
				"Aliyun", "ALIYUN",
				"Qiniu", "QINIU",
				"Tencent", "TENCENT",
				"AWS", "AWS",
				"Google", "GOOGLE",
				"Azure", "AZURE",
				"Baidu", "BAIDU",
				"Huawei", "HUAWEI",
				"QCloud", "QCLOUD",
				"Local", "LOCAL",
			).
			Default("MINIO").
			Optional().
			Nillable(),

		field.String("bucket_name").
			Comment("存储桶名称").
			Optional().
			Nillable(),

		field.String("file_directory").
			Comment("文件目录").
			Optional().
			Nillable(),

		field.String("file_guid").
			Comment("文件Guid").
			Optional().
			Nillable(),

		field.String("save_file_name").
			Comment("保存文件名").
			Optional().
			Nillable(),

		field.String("file_name").
			Comment("文件名").
			Optional().
			Nillable(),

		field.String("extension").
			Comment("文件扩展名").
			Optional().
			Nillable(),

		field.Uint64("size").
			Comment("文件字节长度").
			Optional().
			Nillable(),

		field.String("size_format").
			Comment("文件大小格式化").
			Optional().
			Nillable(),

		field.String("link_url").
			Comment("链接地址").
			Optional().
			Nillable(),

		field.String("md5").
			Comment("md5码，防止上传重复文件").
			Optional().
			Nillable(),
	}
}

// Mixin of the File.
func (File) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID{},
	}
}

func (File) Indexes() []ent.Index {
	return []ent.Index{
		// 支持按租户快速筛选
		index.Fields("tenant_id").
			StorageKey("idx_files_tenant_id"),

		// 租户维度唯一：文件 GUID 在同一租户内唯一
		index.Fields("tenant_id", "file_guid").
			Unique().
			StorageKey("uix_files_tenant_file_guid"),

		// 租户维度的 md5 用于去重/定位（非强制唯一，视业务决定是否改为 Unique）
		index.Fields("tenant_id", "md5").
			StorageKey("idx_files_tenant_md5"),
		// 全局 md5 索引（模糊/跨租户场景）
		index.Fields("md5").
			StorageKey("idx_files_md5"),

		// 常用查询字段索引
		index.Fields("bucket_name").
			StorageKey("idx_files_bucket_name"),
		index.Fields("file_name").
			StorageKey("idx_files_file_name"),
		index.Fields("extension").
			StorageKey("idx_files_extension"),
		index.Fields("size").
			StorageKey("idx_files_size"),

		// 按创建时间查询/排序优化（假定 mixin 中为 created_at）
		index.Fields("created_at").
			StorageKey("idx_files_created_at"),
	}
}

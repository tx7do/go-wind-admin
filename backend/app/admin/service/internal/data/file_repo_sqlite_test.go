package data

import (
	"context"
	"fmt"
	"testing"

	"entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	storageV1 "go-wind-admin/api/gen/go/storage/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entFile "go-wind-admin/app/admin/service/internal/data/ent/file"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newFileRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的 FileRepo，
// 逐字段复刻 NewFileRepo 的 mapper/converter 初始化，再调用 init()。
func newFileRepoSqlite(t *testing.T) *FileRepo {
	t.Helper()
	repo := &FileRepo{
		entClient: enttest.NewEntClientForTest(t),
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[storageV1.File, ent.File](),
		providerConverter: mapper.NewEnumTypeConverter[storageV1.OSSProvider, entFile.Provider](
			storageV1.OSSProvider_name, storageV1.OSSProvider_value,
		),
	}
	repo.init()
	return repo
}

// TestFileRepoSqlite_Create 验证 Create 的字段落库与
// size → size_format 的格式化推导（512B / 2KB / 1MB / 0B 四个分支）。
func TestFileRepoSqlite_Create(t *testing.T) {
	repo := newFileRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &storageV1.CreateFileRequest{
		Data: &storageV1.File{
			Provider:      storageV1.OSSProvider_AWS.Enum(),
			BucketName:    trans.Ptr("bucket-alpha"),
			FileDirectory: trans.Ptr("dir/alpha"),
			FileGuid:      trans.Ptr("guid-sqlite-file-create-1"),
			SaveFileName:  trans.Ptr("saved_1.bin"),
			FileName:      trans.Ptr("报表.docx"),
			Extension:     trans.Ptr("docx"),
			Size:          trans.Ptr(uint64(512)),
			LinkUrl:       trans.Ptr("https://oss.example/bucket-alpha/saved_1.bin"),
			ContentHash:   trans.Ptr("sha256:abc"),
			CreatedBy:     trans.Ptr(uint32(21)),
		},
	})
	require.NoError(t, err, "repo.Create 写入 SQLite 应成功")

	rows, err := repo.entClient.Client().File.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "files 应有 1 条记录")
	row := rows[0]
	require.NotNil(t, row.Provider, "provider 枚举应经 converter 落库")
	require.Equal(t, entFile.ProviderAWS, *row.Provider, "proto AWS 应映射为 ent ProviderAws")
	require.Equal(t, "bucket-alpha", *row.BucketName, "bucket_name 应按请求落库")
	require.Equal(t, "dir/alpha", *row.FileDirectory, "file_directory 应按请求落库")
	require.Equal(t, "guid-sqlite-file-create-1", *row.FileGUID, "file_guid 应按请求落库")
	require.Equal(t, "saved_1.bin", *row.SaveFileName, "save_file_name 应按请求落库")
	require.Equal(t, "报表.docx", *row.FileName, "file_name 应按请求落库")
	require.Equal(t, "docx", *row.Extension, "extension 应按请求落库")
	require.Equal(t, uint64(512), *row.Size, "size 应按请求落库")
	require.Equal(t, "512B", *row.SizeFormat, "512 字节应格式化为 512B")
	require.Equal(t, "https://oss.example/bucket-alpha/saved_1.bin", *row.LinkURL, "link_url 应按请求落库")
	require.Equal(t, "sha256:abc", *row.ContentHash, "content_hash 应按请求落库")
	require.Equal(t, uint32(21), *row.CreatedBy, "created_by 应按请求落库")
	require.False(t, row.CreatedAt.IsZero(), "created_at 应由 repo 写入")
}

// TestFileRepoSqlite_SizeFormatBranches 验证 formatSize 的分支：
// 字节单位整数输出、跨单位换算后的两位小数去零、以及非正数输出 0B。
func TestFileRepoSqlite_SizeFormatBranches(t *testing.T) {
	repo := newFileRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	cases := []struct {
		size uint64
		want string
	}{
		{2048, "2KB"},
		{1048576, "1MB"},
		{0, "0B"},
	}
	for i, c := range cases {
		require.NoError(t, repo.Create(ctx, &storageV1.CreateFileRequest{
			Data: &storageV1.File{
				FileGuid: trans.Ptr(fmt.Sprintf("guid-sqlite-file-sizefmt-%d", i)),
				Size:     trans.Ptr(c.size),
			},
		}), "size=%d 建行应成功", c.size)
	}

	rows, err := repo.entClient.Client().File.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, len(cases))
	for _, row := range rows {
		var want string
		for _, c := range cases {
			if row.Size != nil && *row.Size == c.size {
				want = c.want
				break
			}
		}
		require.Equal(t, want, *row.SizeFormat, "size=%d 的 size_format 应为 %s", *row.Size, want)
	}
}

// TestFileRepoSqlite_ProviderEnumPairs 对 provider 枚举的全部取值逐一建行，
// 断言 proto → ent 与 ent → proto（List 路径）的双向映射逐对成立。
func TestFileRepoSqlite_ProviderEnumPairs(t *testing.T) {
	repo := newFileRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	expectedEnt := map[string]int{}
	expectedProto := map[int32]int{}
	serial := 0
	for value, name := range storageV1.OSSProvider_name {
		serial++
		expectedEnt[name] = 1
		expectedProto[value] = 1
		require.NoError(t, repo.Create(ctx, &storageV1.CreateFileRequest{
			Data: &storageV1.File{
				FileGuid: trans.Ptr(fmt.Sprintf("guid-sqlite-file-provider-%d", serial)),
				Provider: storageV1.OSSProvider(value).Enum(),
			},
		}), "provider=%s 建行应成功", name)
	}
	require.NotEmpty(t, expectedEnt, "应存在有效枚举值")

	entRows, err := repo.entClient.Client().File.Query().All(ctx)
	require.NoError(t, err)
	actualEnt := map[string]int{}
	for _, row := range entRows {
		require.NotNil(t, row.Provider, "provider 不应为空")
		actualEnt[string(*row.Provider)]++
	}
	require.Equal(t, expectedEnt, actualEnt, "ent 侧 provider 取值分布应与枚举名集合逐对一致")

	listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	actualProto := map[int32]int{}
	for _, item := range listed.Items {
		require.NotNil(t, item.Provider, "DTO 的 provider 应被 converter 回填")
		actualProto[int32(*item.Provider)]++
	}
	require.Equal(t, expectedProto, actualProto, "DTO 侧 provider 取值分布应与 proto 枚举值集合逐对一致")
}

// TestFileRepoSqlite_ListFilterAndPaging 验证 List 的无过滤全量、
// file_name 列 contains 模糊搜索、id 列等值过滤与分页语义。
func TestFileRepoSqlite_ListFilterAndPaging(t *testing.T) {
	repo := newFileRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	for i, marker := range []string{"MARKERNU", "MARKERXI"} {
		require.NoError(t, repo.Create(ctx, &storageV1.CreateFileRequest{
			Data: &storageV1.File{
				FileName: trans.Ptr(marker + ".bin"),
				FileGuid: trans.Ptr(fmt.Sprintf("guid-sqlite-file-list-%d", i)),
			},
		}), "写入第 %d 行应成功", i)
	}

	rows, err := repo.entClient.Client().File.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	firstID := rows[0].ID
	secondID := rows[1].ID

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应统计全部 2 条")
	require.Len(t, all.Items, 2)

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "file_name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "MARKERNU"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤应只统计命中行")
	require.Len(t, filtered.Items, 1, "contains 过滤应只返回命中行")
	require.Contains(t, filtered.Items[0].GetFileName(), "MARKERNU", "命中行应为含标记的那条")

	byID, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "id",
						Op:         paginationV1.Operator_EQ,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: fmt.Sprintf("%d", firstID)},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), byID.Total, "id 等值过滤应只统计目标行")
	require.Len(t, byID.Items, 1, "id 等值过滤应只返回目标行")
	require.Equal(t, firstID, byID.Items[0].GetId())

	page1, err := repo.List(ctx, &paginationV1.PagingRequest{
		Page:     trans.Ptr(uint32(1)),
		PageSize: trans.Ptr(uint32(1)),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(2), page1.Total, "分页时 Total 应仍为全量 2")
	require.Len(t, page1.Items, 1, "pageSize=1 第一页应只含 1 行")

	page2, err := repo.List(ctx, &paginationV1.PagingRequest{
		Page:     trans.Ptr(uint32(2)),
		PageSize: trans.Ptr(uint32(1)),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(2), page2.Total, "分页时 Total 应仍为全量 2")
	require.Len(t, page2.Items, 1, "pageSize=1 第二页应只含 1 行")
	require.ElementsMatch(t, []uint32{firstID, secondID},
		[]uint32{page1.Items[0].GetId(), page2.Items[0].GetId()}, "两页合并应覆盖全部行")
}

// TestFileRepoSqlite_Get 验证 Get 按主键的命中与未命中。
func TestFileRepoSqlite_Get(t *testing.T) {
	repo := newFileRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &storageV1.CreateFileRequest{
		Data: &storageV1.File{
			FileGuid:  trans.Ptr("guid-sqlite-file-get-1"),
			FileName:  trans.Ptr("命中检查.bin"),
			Extension: trans.Ptr("bin"),
		},
	}))

	rows, err := repo.entClient.Client().File.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	got, err := repo.Get(ctx, &storageV1.GetFileRequest{
		QueryBy: &storageV1.GetFileRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, got.GetId())
	require.Equal(t, "命中检查.bin", got.GetFileName(), "命中记录的 file_name 应与写入一致")
	require.Equal(t, "bin", got.GetExtension(), "命中记录的 extension 应与写入一致")

	_, err = repo.Get(ctx, &storageV1.GetFileRequest{
		QueryBy: &storageV1.GetFileRequest_Id{Id: 99999},
	})
	require.Error(t, err, "查询不存在的主键应返回错误")
}

// TestFileRepoSqlite_UpdateMaskAndAllowMissing 验证 Update 的单字段掩码更新、
// 掩码外字段保持原值、AllowMissing 对不存在 ID 走创建路径、参数校验分支。
func TestFileRepoSqlite_UpdateMaskAndAllowMissing(t *testing.T) {
	repo := newFileRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	require.NoError(t, repo.Create(ctx, &storageV1.CreateFileRequest{
		Data: &storageV1.File{
			FileGuid:  trans.Ptr("guid-sqlite-file-update-1"),
			FileName:  trans.Ptr("更新前文件名"),
			Extension: trans.Ptr("txt"),
		},
	}))
	rows, err := repo.entClient.Client().File.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	// 单字段掩码：file_name 更新、extension 保持
	err = repo.Update(ctx, &storageV1.UpdateFileRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"file_name"}},
		Data: &storageV1.File{
			FileName:  trans.Ptr("更新后文件名"),
			Extension: trans.Ptr("不应更新的扩展名"),
		},
	})
	require.NoError(t, err, "掩码内字段更新应成功")
	after, err := repo.entClient.Client().File.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.Equal(t, "更新后文件名", *after[0].FileName, "掩码内字段 file_name 应被更新")
	require.Equal(t, "txt", *after[0].Extension, "掩码外字段 extension 应保持原值")

	// 参数校验分支
	require.Error(t, repo.Update(ctx, &storageV1.UpdateFileRequest{
		Id:   0,
		Data: &storageV1.File{FileName: trans.Ptr("x")},
	}), "Id=0 应返回错误")
	require.Error(t, repo.Update(ctx, &storageV1.UpdateFileRequest{
		Id:   createdID,
		Data: nil,
	}), "Data=nil 应返回错误")

	// AllowMissing：不存在的 ID 走创建路径
	err = repo.Update(ctx, &storageV1.UpdateFileRequest{
		Id:           987654,
		AllowMissing: trans.Ptr(true),
		Data: &storageV1.File{
			FileGuid: trans.Ptr("guid-sqlite-file-allowmissing-1"),
			FileName: trans.Ptr("allowmissing创建的文件"),
		},
	})
	require.NoError(t, err, "AllowMissing 对不存在 ID 应走创建路径")
	after2, err := repo.entClient.Client().File.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, after2, 2, "AllowMissing 创建后应有 2 行")
	names := map[string]bool{}
	for _, r := range after2 {
		names[*r.FileName] = true
	}
	require.True(t, names["allowmissing创建的文件"], "新行应由 AllowMissing 创建")

	// 存在的 ID + AllowMissing：仍走更新路径（改名）
	err = repo.Update(ctx, &storageV1.UpdateFileRequest{
		Id:           createdID,
		AllowMissing: trans.Ptr(true),
		UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"file_name"}},
		Data: &storageV1.File{
			FileName: trans.Ptr("再次更新后的文件名"),
		},
	})
	require.NoError(t, err, "存在的 ID + AllowMissing 应走更新路径")
	after3, err := repo.entClient.Client().File.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, after3, 2)
	for _, r := range after3 {
		if r.ID == createdID {
			require.Equal(t, "再次更新后的文件名", *r.FileName, "存在的 ID 应被更新")
		}
	}
}

// TestFileRepoSqlite_DeleteAndUniqueGuid 验证 Delete 的删除与不存在报错、
// 以及 (tenant_id, file_guid) 唯一约束：同租户重复 guid 被拒、跨租户同 guid 允许。
func TestFileRepoSqlite_DeleteAndUniqueGuid(t *testing.T) {
	repo := newFileRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 同租户（tenant 0）同 guid：第二条被唯一索引拒绝
	require.NoError(t, repo.Create(ctx, &storageV1.CreateFileRequest{
		Data: &storageV1.File{
			FileGuid: trans.Ptr("guid-sqlite-file-uniq-1"),
			FileName: trans.Ptr("uniq-a.bin"),
		},
	}), "首条同 guid 行应创建成功")
	require.Error(t, repo.Create(ctx, &storageV1.CreateFileRequest{
		Data: &storageV1.File{
			FileGuid: trans.Ptr("guid-sqlite-file-uniq-1"),
			FileName: trans.Ptr("uniq-b.bin"),
		},
	}), "同租户重复 file_guid 应被唯一索引拒绝")

	// 跨租户同 guid：允许（唯一性按租户维度）
	require.NoError(t, repo.Create(ctx, &storageV1.CreateFileRequest{
		Data: &storageV1.File{
			TenantId: trans.Ptr(uint32(5)),
			FileGuid: trans.Ptr("guid-sqlite-file-uniq-1"),
			FileName: trans.Ptr("uniq-other-tenant.bin"),
		},
	}), "跨租户同 file_guid 应允许")

	rows, err := repo.entClient.Client().File.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2, "应有两条不同租户的行")
	delID := rows[0].ID

	// 删除：删除不存在的 ID 报 NotFound
	require.Error(t, repo.Delete(ctx, &storageV1.DeleteFileRequest{
		QueryBy: &storageV1.DeleteFileRequest_Id{Id: 999999},
	}), "删除不存在的 ID 应返回错误")

	// 删除：存在的 ID 删除成功、计数下降
	require.NoError(t, repo.Delete(ctx, &storageV1.DeleteFileRequest{
		QueryBy: &storageV1.DeleteFileRequest_Id{Id: delID},
	}), "删除已存在记录应成功")
	remain, err := repo.entClient.Client().File.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, remain, "删除一条后应剩 1 行")
}

// TestFileRepoSqlite_CountAndIsExist 验证 Count 的带谓词/无谓词语义
// 与 IsExist 的命中/未命中。
func TestFileRepoSqlite_CountAndIsExist(t *testing.T) {
	repo := newFileRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &storageV1.CreateFileRequest{
		Data: &storageV1.File{
			FileGuid:  trans.Ptr("guid-sqlite-file-count-1"),
			Extension: trans.Ptr("png"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &storageV1.CreateFileRequest{
		Data: &storageV1.File{
			FileGuid:  trans.Ptr("guid-sqlite-file-count-2"),
			Extension: trans.Ptr("jpg"),
		},
	}))

	rows, err := repo.entClient.Client().File.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	createdID := rows[0].ID

	total, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 2, total, "无谓词 Count 应为全量 2")

	byID, err := repo.Count(ctx, []func(s *sql.Selector){entFile.IDEQ(createdID)})
	require.NoError(t, err)
	require.Equal(t, 1, byID, "主键等值谓词应只命中 1 行")

	byExtension, err := repo.Count(ctx, []func(s *sql.Selector){entFile.ExtensionEQ("png")})
	require.NoError(t, err)
	require.Equal(t, 1, byExtension, "extension 等值谓词应只命中 1 行")

	exist, err := repo.IsExist(ctx, createdID)
	require.NoError(t, err)
	require.True(t, exist, "存在的主键 IsExist 应为 true")
	exist, err = repo.IsExist(ctx, 99999)
	require.NoError(t, err)
	require.False(t, exist, "不存在的主键 IsExist 应为 false")
}

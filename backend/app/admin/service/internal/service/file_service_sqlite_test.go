// FileService 的 SQLite 内存库测试（白盒，包内测试）。
//
// 元数据 CRUD 的盖章与派生口径：
//   - Create/Update 盖令牌操作人章；size 列在 repo 层派生 size_format 冗余列，
//     元数据经 service 走的也是同一条派生链（文件列表显示"1.5KB"就靠它）；
//   - Delete 的守卫分支：目标不存在时在查元数据一步就失败——MinIO 侧删除永远不会
//     在"对象不存在"时被触达。主路径（mc.DeleteFile）依赖真实 MinIO，打桩不到，
//     按任务约定不硬凑：文件行删除后 mc 未被调用这一行为无法在单测里断言，
//     只保证 DB 侧顺序（先查后删）与守卫。
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	storageV1 "go-wind-admin/api/gen/go/storage/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newFileServiceForTest 白盒装配 FileService：log 换 NopLogger、repo 走 SQLite 内存库、
// mc 置 nil（被测分支不触达 MinIO；触达即 panic，反而是守卫失效的警报）。
func newFileServiceForTest(t *testing.T) (*FileService, *entCrud.EntClient[*ent.Client]) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &FileService{
		log:      bLogger.NewHelper(bLogger.NopLogger()),
		fileRepo: data.NewFileRepoForTest(entClient),
	}, entClient
}

// TestFileServiceSqlite_CreateUpdateStampsOperatorAndSizeFormat 验证：
// created_by/updated_by 盖令牌操作人章；size 冗余列 size_format 派生落库；
// 掩码更新只动掩码内字段。
func TestFileServiceSqlite_CreateUpdateStampsOperatorAndSizeFormat(t *testing.T) {
	svc, entClient := newFileServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 61})

	// 1536 字节 → 1.5KB（repo 派生口径，元数据经 service 同链路）
	_, err := svc.Create(opCtx, &storageV1.CreateFileRequest{
		Data: &storageV1.File{
			FileName:     trans.Ptr("头像.png"),
			SaveFileName: trans.Ptr("uuid.png"),
			Size:         trans.Ptr(uint64(1536)),
			CreatedBy:    trans.Ptr(uint32(1)), // 客户端伪造，应被盖章覆盖
		},
	})
	require.NoError(t, err, "合法创建应成功")

	row, err := entClient.Client().File.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, uint32(61), *row.CreatedBy, "created_by 应取令牌操作人（客户端伪造无效）")
	require.NotNil(t, row.SizeFormat)
	require.Equal(t, "1.5KB", *row.SizeFormat, "size_format 应由 size 派生（列表直显口径）")

	// 掩码更新：只动 file_name，updated_by 盖章（service 追加掩码路径）。
	// 用另一位操作人更新：审计两列必须各归其位——created_by 留上传者 61，
	// updated_by 记修改人 62（此前 repo 误写 SetNillableCreatedBy(req.Data.UpdatedBy)，
	// 每次更新都会把"谁上传的"改写成"谁改的"）。
	updaterCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 62})
	_, err = svc.Update(updaterCtx, &storageV1.UpdateFileRequest{
		Id:         row.ID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"file_name"}},
		Data:       &storageV1.File{FileName: trans.Ptr("改名后的头像.png")},
	})
	require.NoError(t, err, "掩码更新应成功")
	row, err = entClient.Client().File.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "改名后的头像.png", *row.FileName)
	require.NotNil(t, row.UpdatedBy)
	require.Equal(t, uint32(62), *row.UpdatedBy, "updated_by 应记修改人")
	require.Equal(t, uint32(61), *row.CreatedBy, "created_by 必须保持上传者：更新改写它即审计错乱")

	// 入参守卫与缺操作人
	_, err = svc.Create(opCtx, nil)
	require.Error(t, err, "nil 请求应返回 BadRequest")
	_, err = svc.Create(opCtx, &storageV1.CreateFileRequest{})
	require.Error(t, err, "nil Data 应返回 BadRequest")
	_, err = svc.Create(ctx, &storageV1.CreateFileRequest{Data: &storageV1.File{}})
	require.Error(t, err, "缺操作人声明的创建应被拒绝")
	_, err = svc.Update(opCtx, nil)
	require.Error(t, err, "nil 更新请求应返回 BadRequest")
	_, err = svc.Update(opCtx, &storageV1.UpdateFileRequest{Id: row.ID})
	require.Error(t, err, "nil Data 应返回 BadRequest")
}

// TestFileServiceSqlite_ListGetDeleteGuards 验证 List/Get 透传与 Delete 守卫：
// 目标不存在时 Delete 在查元数据一步失败（MinIO 侧不会被触达）；nil 请求被拒。
func TestFileServiceSqlite_ListGetDeleteGuards(t *testing.T) {
	svc, entClient := newFileServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 62})

	_, err := svc.Create(opCtx, &storageV1.CreateFileRequest{
		Data: &storageV1.File{FileName: trans.Ptr("清单.csv")},
	})
	require.NoError(t, err)

	list, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(1), list.GetTotal())

	_, err = svc.List(ctx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")

	row, err := entClient.Client().File.Query().Only(ctx)
	require.NoError(t, err)
	got, err := svc.Get(ctx, &storageV1.GetFileRequest{
		QueryBy: &storageV1.GetFileRequest_Id{Id: row.ID},
	})
	require.NoError(t, err)
	require.Equal(t, "清单.csv", got.GetFileName())

	_, err = svc.Get(ctx, &storageV1.GetFileRequest{
		QueryBy: &storageV1.GetFileRequest_Id{Id: 987654},
	})
	require.Error(t, err, "不存在的文件应返回错误")

	// Delete 守卫：目标不存在 → 查元数据一步失败，mc（此处为 nil）绝不被触达
	_, err = svc.Delete(ctx, &storageV1.DeleteFileRequest{
		QueryBy: &storageV1.DeleteFileRequest_Id{Id: 987654},
	})
	require.Error(t, err, "删除不存在的文件应失败")
	require.Contains(t, err.Error(), "not found")
	cnt, err := entClient.Client().File.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "失败删除不影响行数")

	// 存在文件但 mc 为 nil：Delete 主路径需要真实 MinIO，本测试不覆盖；
	// 这里只验证 nil 请求在查元数据前就被 ID=0 的查不到挡下
	_, err = svc.Delete(ctx, nil)
	require.Error(t, err, "nil 删除请求应失败")
}

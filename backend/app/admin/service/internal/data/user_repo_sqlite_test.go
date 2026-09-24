package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entUser "go-wind-admin/app/admin/service/internal/data/ent/user"
	"go-wind-admin/app/admin/service/internal/data/ent/userorgunit"
	"go-wind-admin/app/admin/service/internal/data/ent/userposition"
	"go-wind-admin/app/admin/service/internal/data/ent/userrole"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newUserRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的 userRepo。
// 白盒构造逐字段复刻 NewUserRepo 的 mapper/converter 初始化，仅将 log 换为
// NopLogger、entClient 换为 SQLite 内存库测试 client。
// Get/List 途经的 ListUserRelationIDs 会触达三个关系子仓
// （userRoleRepo/userOrgUnitRepo/userPositionRepo），在同一 entclient 上
// 内联白盒构造（逐字段复刻各自 New* 构造器）；默认关联模式为 OneToOne，
// membershipRepo 在该模式下不被触及、保持 nil。
func newUserRepoSqlite(t *testing.T) *userRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	userRoleRepo := &UserRoleRepo{
		log:             bLogger.NewHelper(bLogger.NopLogger()),
		entClient:       entClient,
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.UserRole_Status, userrole.Status](permissionV1.UserRole_Status_name, permissionV1.UserRole_Status_value),
	}
	userOrgUnitRepo := &UserOrgUnitRepo{
		log:             bLogger.NewHelper(bLogger.NopLogger()),
		entClient:       entClient,
		statusConverter: mapper.NewEnumTypeConverter[identityV1.UserOrgUnit_Status, userorgunit.Status](identityV1.UserOrgUnit_Status_name, identityV1.UserOrgUnit_Status_value),
	}
	userPositionRepo := &UserPositionRepo{
		log:             bLogger.NewHelper(bLogger.NopLogger()),
		entClient:       entClient,
		statusConverter: mapper.NewEnumTypeConverter[identityV1.UserPosition_Status, userposition.Status](identityV1.UserPosition_Status_name, identityV1.UserPosition_Status_value),
	}
	repo := &userRepo{
		entClient:        entClient,
		log:              bLogger.NewHelper(bLogger.NopLogger()),
		mapper:           mapper.NewCopierMapper[identityV1.User, ent.User](),
		genderConverter:  mapper.NewEnumTypeConverter[identityV1.User_Gender, entUser.Gender](identityV1.User_Gender_name, identityV1.User_Gender_value),
		statusConverter:  mapper.NewEnumTypeConverter[identityV1.User_Status, entUser.Status](identityV1.User_Status_name, identityV1.User_Status_value),
		userRoleRepo:     userRoleRepo,
		userOrgUnitRepo:  userOrgUnitRepo,
		userPositionRepo: userPositionRepo,
	}
	repo.init()
	return repo
}

// TestUserRepoSqlite_Get 验证 userRepo.Get 按主键查询的命中与未命中，
// 以及 status/gender 枚举的落库与读视图回填：实体侧两者均为带列默认值的
// 可空指针枚举、DTO 侧为可选指针字段，mapper 的枚举转换对无法赋入而丢弃，
// 读路径经 queryEnumsAndBackfill 统一回填后如实呈现写入值。
func TestUserRepoSqlite_Get(t *testing.T) {
	repo := newUserRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	_, err := repo.Create(ctx, &identityV1.CreateUserRequest{
		Data: &identityV1.User{
			Username: trans.Ptr("sqlite_get_user"),
			Nickname: trans.Ptr("读视图用户"),
			Status:   identityV1.User_LOCKED.Enum(),
			Gender:   identityV1.User_MALE.Enum(),
		},
	})
	require.NoError(t, err, "通过 repo.Create 写入 SQLite 应成功")

	rows, err := repo.entClient.Client().User.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQLite 中应有 1 条 user 记录")
	require.Equal(t, entUser.StatusLocked, *rows[0].Status, "proto User_LOCKED 应映射为 ent StatusLocked")
	require.Equal(t, entUser.GenderMale, *rows[0].Gender, "proto User_MALE 应映射为 ent GenderMale")
	createdID := uint32(rows[0].ID)

	// 命中：按主键
	gotByID, err := repo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按主键查询已存在记录应命中")
	require.Equal(t, "sqlite_get_user", gotByID.GetUsername(), "命中记录的 username 应与写入一致")
	// 读视图：status/gender 均应经回填如实呈现写入值。
	require.Equal(t, identityV1.User_LOCKED, gotByID.GetStatus(), "读视图应回填 status")
	require.Equal(t, identityV1.User_MALE, gotByID.GetGender(), "读视图应回填 gender")

	// ListUsersByIds：实体已在手的路径同样经 backfillEnumsFrom 回填
	byIds, err := repo.ListUsersByIds(ctx, []uint32{createdID})
	require.NoError(t, err, "按主键列表查询应命中")
	require.Len(t, byIds, 1, "按主键列表应命中 1 条")
	require.Equal(t, identityV1.User_LOCKED, byIds[0].GetStatus(), "ListUsersByIds 读视图应回填 status")
	require.Equal(t, identityV1.User_MALE, byIds[0].GetGender(), "ListUsersByIds 读视图应回填 gender")

	// 未命中：不存在的主键
	_, err = repo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{Id: 99999},
	})
	require.Error(t, err, "查询不存在的主键应返回错误")
}

// TestUserRepoSqlite_List 验证 userRepo.List 的分页列表读视图：
// 各行携带不同显式枚举值，读视图按行内实际存储值各自如实回填。
func TestUserRepoSqlite_List(t *testing.T) {
	repo := newUserRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 两条带可区分标记、各自携带不同显式枚举值的记录
	_, err := repo.Create(ctx, &identityV1.CreateUserRequest{
		Data: &identityV1.User{
			Username: trans.Ptr("MARKERALPHA 用户"),
			Status:   identityV1.User_LOCKED.Enum(),
			Gender:   identityV1.User_MALE.Enum(),
		},
	})
	require.NoError(t, err)
	_, err = repo.Create(ctx, &identityV1.CreateUserRequest{
		Data: &identityV1.User{
			Username: trans.Ptr("MARKERBETA 用户"),
			Status:   identityV1.User_EXPIRED.Enum(),
			Gender:   identityV1.User_FEMALE.Enum(),
		},
	})
	require.NoError(t, err)

	// 无过滤：应返回全部 2 条
	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应统计全部 2 条")
	require.Len(t, all.Items, 2, "无过滤时应返回 2 条")
	for _, item := range all.Items {
		// 列表读视图：枚举按行内实际存储值回填，随标记行各自如实呈现。
		switch item.GetUsername() {
		case "MARKERALPHA 用户":
			require.Equal(t, identityV1.User_LOCKED, item.GetStatus(), "MARKERALPHA 行读视图应回填 status=LOCKED")
			require.Equal(t, identityV1.User_MALE, item.GetGender(), "MARKERALPHA 行读视图应回填 gender=MALE")
		case "MARKERBETA 用户":
			require.Equal(t, identityV1.User_EXPIRED, item.GetStatus(), "MARKERBETA 行读视图应回填 status=EXPIRED")
			require.Equal(t, identityV1.User_FEMALE, item.GetGender(), "MARKERBETA 行读视图应回填 gender=FEMALE")
		default:
			t.Fatalf("意外的 username: %q", item.GetUsername())
		}
	}
}

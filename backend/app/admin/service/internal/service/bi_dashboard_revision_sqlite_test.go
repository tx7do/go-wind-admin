package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	biV1 "go-wind-admin/api/gen/go/bi/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// TestBiDashboardRevision_Sqlite_SaveAndRestore 版本快照链路回归（docs/bi_dashboard.md
// 「已知边界/后续」的 revisions 表）：
// ① 更新保存成功后，保存前的 payload/name/version 留一行快照；
// ② 恢复把快照写回主表且版本 +1，恢复动作本身先以 RESTORE 来源留底当前版本（可逆）。
func TestBiDashboardRevision_Sqlite_SaveAndRestore(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	svc := &BiDashboardService{
		log:             bLogger.NewHelper(bLogger.NopLogger()),
		biDashboardRepo: data.NewBiDashboardRepoForTest(entClient),
		revisionRepo:    data.NewBiDashboardRevisionRepoForTest(entClient),
	}

	// 直插一行仪表盘（payload/name/version 基线，版本默认 1）
	row, err := entClient.Client().BiDashboard.Create().
		SetName("v1").
		SetNillablePayload(trans.Ptr(`{"charts":[1]}`)).
		Save(ctx)
	require.NoError(t, err)
	dashId := row.ID

	// 模拟登录态（快照记录触发者）
	ctx = auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	// ① 更新保存：v1 → v2
	_, err = svc.Update(ctx, &biV1.UpdateBiDashboardRequest{
		Id: dashId,
		Data: &biV1.BiDashboard{
			Name:    trans.Ptr("v2"),
			Payload: trans.Ptr(`{"charts":[1,2]}`),
			Version: trans.Ptr(uint32(1)),
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name", "payload", "version"}},
	})
	require.NoError(t, err, "更新保存应成功")

	after, err := entClient.Client().BiDashboard.Get(ctx, dashId)
	require.NoError(t, err)
	require.Equal(t, uint32(2), *after.Version, "版本号应 +1")
	require.Equal(t, "v2", *after.Name)

	revs, total, err := svc.revisionRepo.List(ctx, dashId, nil)
	require.NoError(t, err)
	require.Equal(t, uint32(1), total, "应有一行 v1 快照")
	require.Equal(t, uint32(1), revs[0].GetVersion(), "快照记的是保存前的旧版本")
	require.Equal(t, "v1", revs[0].GetName())
	require.JSONEq(t, `{"charts":[1]}`, revs[0].GetPayload())
	require.Equal(t, "SAVE", revs[0].GetSource())
	require.Equal(t, uint32(7), revs[0].GetCreatedBy())

	// ② 恢复到 v1 快照：当前 v2 先以 RESTORE 留底，再把 v1 写回
	_, err = svc.RestoreRevision(ctx, &biV1.RestoreBiDashboardRevisionRequest{Id: revs[0].GetId()})
	require.NoError(t, err, "恢复应成功")

	afterRestore, err := entClient.Client().BiDashboard.Get(ctx, dashId)
	require.NoError(t, err)
	require.Equal(t, "v1", *afterRestore.Name, "恢复后名称回到快照值")
	require.Equal(t, uint32(3), *afterRestore.Version, "恢复也是一次保存，版本 +1")
	require.JSONEq(t, `{"charts":[1]}`, *afterRestore.Payload)

	revs2, total2, err := svc.revisionRepo.List(ctx, dashId, nil)
	require.NoError(t, err)
	require.Equal(t, uint32(2), total2, "恢复前应先留底当前版本")
	sources := map[string]bool{}
	for _, r := range revs2 {
		sources[r.GetSource()] = true
	}
	require.True(t, sources["RESTORE"], "应有一行 RESTORE 来源的留底快照")
	require.True(t, sources["SAVE"], "原 SAVE 快照保留")
}

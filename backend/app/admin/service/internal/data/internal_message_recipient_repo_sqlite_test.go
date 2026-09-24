package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	crudViewer "github.com/tx7do/go-crud/viewer"

	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entInternalMessageRecipient "go-wind-admin/app/admin/service/internal/data/ent/internalmessagerecipient"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	appViewer "go-wind-admin/pkg/entgo/viewer"
)

// newInternalMessageRecipientRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的
// InternalMessageRecipientRepo，逐字段复刻 NewInternalMessageRecipientRepo 的
// mapper/converter 初始化，再调用 init()。
func newInternalMessageRecipientRepoSqlite(t *testing.T) *InternalMessageRecipientRepo {
	t.Helper()
	repo := &InternalMessageRecipientRepo{
		entClient: enttest.NewEntClientForTest(t),
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[internalMessageV1.InternalMessageRecipient, ent.InternalMessageRecipient](),
		statusConverter: mapper.NewEnumTypeConverter[internalMessageV1.InternalMessageRecipient_Status, entInternalMessageRecipient.Status](
			internalMessageV1.InternalMessageRecipient_Status_name, internalMessageV1.InternalMessageRecipient_Status_value,
		),
	}
	repo.init()
	return repo
}

// TestInternalMessageRecipientRepoSqlite_EnumReadback 对 status 枚举的全部 5 个取值
// 逐一显式建行，另建一行未指定 status 的行（该枚举无列默认，落 NULL）。断言：
//  1. ent 行侧：显式值经 converter 如实落库；未指定行的 status 为 NULL；
//  2. List 与 Get 读视图：显式行的 DTO status 如实呈现行内存储值（含显式指定的
//     零值 SENT——与"字段缺失"经 nil 语义区分）；未指定行的 DTO status 为 nil
//     （字段缺失如实缺失，不发生零值伪造）；
//  3. Create 返回的 DTO（同一 mapper）与上述读视图行为一致。
//
// 形态说明：实体侧 status 为可空指针枚举（schema Nillable），DTO 侧为可选指针
// 字段——两侧同为指针时，mapper 注册的枚举转换对（实体枚举名 → proto 枚举值）
// 在 copier 中逐字段命中并如实拷贝；行内 NULL 经 copier 的 nil 传播在 DTO 侧保持
// nil。本测试钉住该行为：若转换对被注销、或实体侧形态漂移为值型默认枚举
// （值↔指针不匹配会被 copier 丢弃并退化为零值——岗位仓 type 的历史缺陷形态），
// 读视图的退化（丢值或零值伪造）将被本测试捕获。
func TestInternalMessageRecipientRepoSqlite_EnumReadback(t *testing.T) {
	repo := newInternalMessageRecipientRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// marker 经 RecipientUserId 落库并读回，用于在 List/Get 结果中定位各行。
	type enumCase struct {
		marker    uint32
		status    *internalMessageV1.InternalMessageRecipient_Status // nil=未指定 → 落 NULL
		wantNil   bool                                               // 未指定行：ent/DTO 侧 status 堆 nil
		wantProto internalMessageV1.InternalMessageRecipient_Status  // 显式行：读回期望
		wantEnt   entInternalMessageRecipient.Status
	}
	cases := []enumCase{}
	for _, s := range []struct {
		proto   internalMessageV1.InternalMessageRecipient_Status
		entWant entInternalMessageRecipient.Status
	}{
		{internalMessageV1.InternalMessageRecipient_SENT, entInternalMessageRecipient.StatusSent},
		{internalMessageV1.InternalMessageRecipient_RECEIVED, entInternalMessageRecipient.StatusReceived},
		{internalMessageV1.InternalMessageRecipient_READ, entInternalMessageRecipient.StatusRead},
		{internalMessageV1.InternalMessageRecipient_REVOKED, entInternalMessageRecipient.StatusRevoked},
		{internalMessageV1.InternalMessageRecipient_DELETED, entInternalMessageRecipient.StatusDeleted},
	} {
		cases = append(cases, enumCase{
			marker:    9000 + uint32(s.proto),
			status:    s.proto.Enum(),
			wantNil:   false,
			wantProto: s.proto,
			wantEnt:   s.entWant,
		})
	}
	// 未指定行：无列默认，落 NULL。
	cases = append(cases, enumCase{marker: 9999, status: nil, wantNil: true})

	for _, c := range cases {
		created, err := repo.Create(ctx, &internalMessageV1.InternalMessageRecipient{
			RecipientUserId: trans.Ptr(c.marker),
			Status:          c.status,
		})
		require.NoError(t, err, "行 %d 建行应成功", c.marker)

		// Create 返回的 DTO 走同一 mapper：读视图行为与 List/Get 一致。
		if c.wantNil {
			require.Nil(t, created.Status,
				"行 %d 的 Create 返回 DTO 应保持 status nil（行内 NULL）", c.marker)
		} else {
			require.NotNil(t, created.Status, "行 %d 的 Create 返回 DTO 的 status 应为非 nil", c.marker)
			require.Equal(t, c.wantProto, created.GetStatus(),
				"行 %d 的 Create 返回 DTO 的 status 应如实呈现行内存储值", c.marker)
		}
	}

	// —— ent 行侧：显式值经 converter 落库；未指定行为 NULL ——
	rows, err := repo.entClient.Client().InternalMessageRecipient.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, len(cases), "应有 %d 行", len(cases))
	rowsByMarker := map[uint32]*ent.InternalMessageRecipient{}
	for _, row := range rows {
		rowsByMarker[*row.RecipientUserID] = row
	}
	ids := make(map[uint32]uint32, len(cases)) // marker → 行主键
	for _, c := range cases {
		row, ok := rowsByMarker[c.marker]
		require.True(t, ok, "行 %d 应存在", c.marker)
		ids[c.marker] = row.ID
		if c.wantNil {
			require.Nil(t, row.Status, "行 %d 未指定 status 应落 NULL", c.marker)
		} else {
			require.NotNil(t, row.Status, "行 %d 的 status 应经 converter 落库", c.marker)
			require.Equal(t, c.wantEnt, *row.Status, "行 %d 的 status 应经 converter 如实落库", c.marker)
		}
	}

	// —— List 读视图：显式行如实呈现、缺失行保持 nil ——
	listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Len(t, listed.Items, len(cases), "List 应返回全部 %d 行", len(cases))
	itemsByMarker := map[uint32]*internalMessageV1.InternalMessageRecipient{}
	for _, item := range listed.Items {
		itemsByMarker[item.GetRecipientUserId()] = item
	}
	for _, c := range cases {
		item, ok := itemsByMarker[c.marker]
		require.True(t, ok, "行 %d 应出现在 List 结果中", c.marker)
		if c.wantNil {
			require.Nil(t, item.Status,
				"行 %d 的 status 读视图应保持 nil（行内 NULL、字段缺失如实缺失）", c.marker)
		} else {
			require.NotNil(t, item.Status, "行 %d 的 status 读视图应为非 nil", c.marker)
			require.Equal(t, c.wantProto, item.GetStatus(),
				"行 %d 的 status 读视图应如实呈现行内存储值", c.marker)
		}
	}

	// —— Get 读视图：按主键逐行读取，行为与 List 一致 ——
	for _, c := range cases {
		got, err := repo.Get(ctx, &internalMessageV1.GetInternalMessageRecipientRequest{
			QueryBy: &internalMessageV1.GetInternalMessageRecipientRequest_Id{Id: ids[c.marker]},
		})
		require.NoError(t, err, "行 %d 按主键读取应命中", c.marker)
		if c.wantNil {
			require.Nil(t, got.Status,
				"行 %d 的 Get 读视图应保持 nil（行内 NULL）", c.marker)
		} else {
			require.NotNil(t, got.Status, "行 %d 的 Get 读视图应为非 nil", c.marker)
			require.Equal(t, c.wantProto, got.GetStatus(),
				"行 %d 的 Get 读视图应如实呈现行内存储值", c.marker)
		}
	}
}

// TestInternalMessageRecipientTenantSqlite 锁定收件行 tenant_id 的三条真实语义。
//
// 为什么要跑出来而不是读代码：repo 统一写 SetNillableTenantID(req.TenantId)，
// 但落库结果取决于 viewer 而不取决于调用方是否"记得传"——go-crud TenantPrivacy 在
// Create 上分平台/租户两套行为，留空时由 ent 的 DefaultTenantID=0 兜底。
// 全员广播在 asynq handler 里以 SystemViewer 运行，一旦不显式传收件用户的租户，
// 整批收件行会静默落到 0，收件人（租户用户）的查询被租户谓词过滤后一行也读不到——
// 不报错、不丢日志。本测试同时是 InternalMessageService 广播路径传租户的回归护栏。
func TestInternalMessageRecipientTenantSqlite(t *testing.T) {
	repo := newInternalMessageRecipientRepoSqlite(t)
	sysCtx := enttest.NewSystemViewerCtx(context.Background())
	tenantCtx := crudViewer.WithContext(context.Background(), appViewer.NewUserViewer(1, 5, 0, "", nil))

	newRecipient := func(messageID, recipientUserID uint32, tenantID *uint32) *internalMessageV1.InternalMessageRecipient {
		return &internalMessageV1.InternalMessageRecipient{
			TenantId:        tenantID,
			MessageId:       trans.Ptr(messageID),
			RecipientUserId: trans.Ptr(recipientUserID),
			Status:          internalMessageV1.InternalMessageRecipient_RECEIVED.Enum(),
		}
	}

	listAs := func(t *testing.T, uid uint32) map[uint32]bool {
		t.Helper()
		// 收件箱读取被服务端钉成"viewer 自己的收件行"（见 repo.List 的归属谓词），
		// 所以这里必须按要读的那个人构造 viewer，而不是只给租户。
		ctx := crudViewer.WithContext(context.Background(), appViewer.NewUserViewer(uint64(uid), 5, 0, "", nil))
		inbox, err := repo.List(ctx, &paginationV1.PagingRequest{})
		require.NoError(t, err)
		seen := make(map[uint32]bool, len(inbox.GetItems()))
		for _, item := range inbox.GetItems() {
			seen[item.GetRecipientUserId()] = true
		}
		return seen
	}

	t.Run("平台上下文留空落0且租户读者读不到", func(t *testing.T) {
		created, err := repo.Create(sysCtx, newRecipient(101, 201, nil))
		require.NoError(t, err)
		require.Equal(t, uint32(0), created.GetTenantId(), "SystemViewer 下未显式传租户 → 落 DefaultTenantID=0")
		require.False(t, listAs(t, 201)[201], "tenant_id=0 的收件行对租户 5 不可见（这就是广播丢投递的机理）")
	})

	t.Run("平台上下文显式传租户被尊重", func(t *testing.T) {
		created, err := repo.Create(sysCtx, newRecipient(102, 202, trans.Ptr(uint32(5))))
		require.NoError(t, err)
		require.Equal(t, uint32(5), created.GetTenantId(), "平台/系统上下文应尊重显式设置（广播修复依赖此行为）")
		require.True(t, listAs(t, 202)[202], "带正确租户的收件行必须对租户 5 的收件箱可见")
		require.False(t, listAs(t, 999)[202],
			"同租户、换一个收件人就读不到：归属谓词由服务端给出，不依赖调用方传 recipient_user_id")
	})

	t.Run("租户上下文强制覆盖为本租户", func(t *testing.T) {
		created, err := repo.Create(tenantCtx, newRecipient(103, 203, trans.Ptr(uint32(9))))
		require.NoError(t, err)
		require.Equal(t, uint32(5), created.GetTenantId(),
			"非平台上下文传他租户会被强制覆盖为当前租户——定向路径显式传值只是冗余，不是防线")
	})
}

// TestInternalMessageRecipientInboxWritesSqlite 钉住收件箱三个写口（标已读 / 改状态 / 从收件箱删除）
// 的归属：user_id 是别人的时候，别人的行碰不到。
//
// 为什么必须跑出来而不是读代码：读侧的同形洞（repo.List 的归属谓词）正是"看着有租户隔离、
// 其实同租户内换一个 id 就行"，而写侧后果更重——DeleteNotificationFromInbox 在 recipient_ids
// 为空时按用户维度**整箱删除**，换一个 id 就是清空别人的收件箱。
//
// 钉住之后有两种语义，都写死在下面，免得下次有人当成 bug 顺手"修"成另一种：
//   - 带 recipient_ids：条件缩成"我自己的行 ∩ 这些 id"= 空集，整个调用是空操作；
//   - recipient_ids 为空（"全部已读"/"清空"）：作用域缩回**调用者自己**的收件箱——越权意图
//     伤不到目标用户，但会作用在自己身上。选"缩回"而不是"报错"，是因为三端合法调用方传的
//     都是自己的 id，加一条错误码等于给三端新增文案与分支。
//
// 平台/系统上下文豁免同读侧：代客操作要按指定用户执行。
func TestInternalMessageRecipientInboxWritesSqlite(t *testing.T) {
	// 每个子测试起一个干净库：断言读的是"另一个人的行有没有被动过"，共享库会互相污染。
	newEnv := func(t *testing.T) (*InternalMessageRecipientRepo, uint32, uint32) {
		t.Helper()

		repo := newInternalMessageRecipientRepoSqlite(t)
		sysCtx := enttest.NewSystemViewerCtx(context.Background())

		var ownID, otherID uint32
		for _, uid := range []uint32{201, 202} {
			// 显式带 tenant_id=5：同租户两个用户才是本洞的前提（跨租户早有 TenantMutationGuard）。
			created, err := repo.Create(sysCtx, &internalMessageV1.InternalMessageRecipient{
				TenantId:        trans.Ptr(uint32(5)),
				MessageId:       trans.Ptr(uint32(101)),
				RecipientUserId: trans.Ptr(uid),
				Status:          internalMessageV1.InternalMessageRecipient_RECEIVED.Enum(),
			})
			require.NoError(t, err)
			if uid == 201 {
				ownID = created.GetId()
			} else {
				otherID = created.GetId()
			}
		}

		return repo, ownID, otherID
	}

	// statusOf 直读 ent 行，绕开三个被测写口自己。
	statusOf := func(t *testing.T, repo *InternalMessageRecipientRepo, recipientRowID uint32) (entInternalMessageRecipient.Status, bool) {
		t.Helper()

		sysCtx := enttest.NewSystemViewerCtx(context.Background())
		row, err := repo.entClient.Client().InternalMessageRecipient.Query().
			Where(entInternalMessageRecipient.IDEQ(recipientRowID)).Only(sysCtx)
		if ent.IsNotFound(err) {
			return "", false
		}
		require.NoError(t, err)
		require.NotNil(t, row.Status, "建行时显式写了 RECEIVED，读回应非空")

		return *row.Status, true
	}

	asUser := func(uid uint32) context.Context {
		return crudViewer.WithContext(context.Background(), appViewer.NewUserViewer(uint64(uid), 5, 0, "", nil))
	}

	t.Run("标已读碰不到别人的行", func(t *testing.T) {
		repo, ownID, otherID := newEnv(t)

		require.NoError(t, repo.MarkNotificationAsRead(asUser(201), &internalMessageV1.MarkNotificationAsReadRequest{
			UserId: 202, RecipientIds: []uint32{otherID},
		}))
		status, ok := statusOf(t, repo, otherID)
		require.True(t, ok, "别人的收件行不许被删，只许状态不变")
		require.Equal(t, entInternalMessageRecipient.StatusReceived, status,
			"带 recipient_ids 的跨用户标已读缩成空操作")
		own, _ := statusOf(t, repo, ownID)
		require.Equal(t, entInternalMessageRecipient.StatusReceived, own, "空操作不该顺手改自己的行")
	})

	t.Run("空 ids 的整箱操作缩回自己", func(t *testing.T) {
		repo, ownID, otherID := newEnv(t)

		require.NoError(t, repo.MarkNotificationAsRead(asUser(201), &internalMessageV1.MarkNotificationAsReadRequest{
			UserId: 202,
		}))
		other, _ := statusOf(t, repo, otherID)
		require.Equal(t, entInternalMessageRecipient.StatusReceived, other, "目标用户一行未动")
		own, _ := statusOf(t, repo, ownID)
		require.Equal(t, entInternalMessageRecipient.StatusRead, own,
			"作用域缩回调用者自己：这是钉定的后果，不是漏网")
	})

	t.Run("改状态碰不到别人的行", func(t *testing.T) {
		repo, _, otherID := newEnv(t)

		require.NoError(t, repo.MarkNotificationsStatus(asUser(201), &internalMessageV1.MarkNotificationsStatusRequest{
			UserId: 202, RecipientIds: []uint32{otherID},
			NewStatus: internalMessageV1.InternalMessageRecipient_READ,
		}))
		other, ok := statusOf(t, repo, otherID)
		require.True(t, ok)
		require.Equal(t, entInternalMessageRecipient.StatusReceived, other)
	})

	t.Run("删除碰不到别人的行", func(t *testing.T) {
		repo, _, otherID := newEnv(t)

		require.NoError(t, repo.DeleteNotificationFromInbox(asUser(201), &internalMessageV1.DeleteNotificationFromInboxRequest{
			UserId: 202, RecipientIds: []uint32{otherID},
		}))
		_, ok := statusOf(t, repo, otherID)
		require.True(t, ok, "带 id 的跨用户删除是空操作，行必须还在")
	})

	t.Run("空 ids 的清空删的是自己的箱", func(t *testing.T) {
		repo, ownID, otherID := newEnv(t)

		require.NoError(t, repo.DeleteNotificationFromInbox(asUser(201), &internalMessageV1.DeleteNotificationFromInboxRequest{
			UserId: 202,
		}))
		_, ok := statusOf(t, repo, otherID)
		require.True(t, ok, "想清空别人收件箱的调用一行都没删掉")
		_, ok = statusOf(t, repo, ownID)
		require.False(t, ok, "而「清空」被缩回成清空自己的收件箱")
	})

	t.Run("自己的行照常可写", func(t *testing.T) {
		repo, ownID, otherID := newEnv(t)

		require.NoError(t, repo.MarkNotificationAsRead(asUser(201), &internalMessageV1.MarkNotificationAsReadRequest{
			UserId: 201, RecipientIds: []uint32{ownID},
		}))
		own, _ := statusOf(t, repo, ownID)
		require.Equal(t, entInternalMessageRecipient.StatusRead, own, "合法调用方（传自己的 id）不受影响")
		other, _ := statusOf(t, repo, otherID)
		require.Equal(t, entInternalMessageRecipient.StatusReceived, other, "同租户另一个人的行仍不受影响")
	})

	t.Run("平台与系统上下文仍可代客", func(t *testing.T) {
		repo, _, otherID := newEnv(t)

		require.NoError(t, repo.MarkNotificationAsRead(
			crudViewer.WithContext(context.Background(), appViewer.NewUserViewer(1, 0, 0, "", nil)),
			&internalMessageV1.MarkNotificationAsReadRequest{UserId: 202, RecipientIds: []uint32{otherID}},
		))
		other, _ := statusOf(t, repo, otherID)
		require.Equal(t, entInternalMessageRecipient.StatusRead, other,
			"平台上下文（租户 0）按指定用户代客操作，与读侧豁免同形")
	})
}

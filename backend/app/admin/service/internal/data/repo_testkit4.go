// 跨包（service 层）测试装配：导出免 bootstrap.Context 的 repo 构造器，
// 字段初始化必须与生产 NewXxxRepo 逐字段一致，改生产构造器须同步此处。
//
// 说明：
//   - 本文件覆盖 repo_testkit.go / repo_testkit2.go 之外、且 repo_testkit5.go
//     未收录批次的纯 ent repo（access_key / notification_channel），供小型带
//     外部件依赖服务（批3a：user_profile / access_key / online_session /
//     notification_channel 的 service 层测试）装配使用。
//     各构造器与对应 *_repo.go 的生产构造器逐字段对齐（含 mapper/converter 初始化
//     与 init() 调用），唯一差异是 log 一律 bLogger.NewHelper(bLogger.NopLogger())；
//     entClient 由调用方传入（测试场景为 enttest.NewEntClientForTest 的 SQLite 内存库）。
//   - 生产构造器中经由 *bootstrap.Context 注入的仅是日志助手，无其他隐藏依赖。
//   - redis 依赖的构造器（UserTokenCache / LoginRateLimiter / ConfigRepo）与
//     UserCredentialRepo / Authenticator 的跨包构造器统一收录于 repo_testkit5.go
//     （注入式：redis 一律传 miniredis 假 client），此处不重复导出以免撞名。
package data

import (
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/accesskey"

	accesskeyV1 "go-wind-admin/api/gen/go/access_key/service/v1"
)

// NewAccessKeyRepoForTest 与生产 NewAccessKeyRepo 逐字段一致（log 换 NopLogger），并调用 init()。
func NewAccessKeyRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *AccessKeyRepo {
	repo := &AccessKeyRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[accesskeyV1.AccessKey, ent.AccessKey](),
		statusConverter: mapper.NewEnumTypeConverter[accesskeyV1.AccessKey_Status, accesskey.Status](
			accesskeyV1.AccessKey_Status_name,
			accesskeyV1.AccessKey_Status_value,
		),
	}

	repo.init()

	return repo
}

// NewNotificationChannelRepoForTest 与生产 NewNotificationChannelRepo 走同一条装配路径
// （log 换 NopLogger），converter 清单因此不会与生产漂移。
func NewNotificationChannelRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *NotificationChannelRepo {
	return newNotificationChannelRepo(bLogger.NewHelper(bLogger.NopLogger()), entClient)
}

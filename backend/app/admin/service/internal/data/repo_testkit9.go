// 跨包（service 层）测试装配：导出免 bootstrap.Context 的 repo 构造器，
// 字段初始化必须与生产 NewXxxRepo 逐字段一致，改生产构造器须同步此处。
//
// 说明：
//   - 本文件为批（BI 仪表盘 / 打印模板 / 权限-菜单关联）repo 层 SQLite 测试所需的构造器，
//     与各自 *_repo.go 的生产构造器逐字段对齐（含 mapper/converter 初始化与 init() 调用），
//     唯一差异是 log 一律 bLogger.NewHelper(bLogger.NopLogger())；entClient 由调用方传入
//     （测试场景为 enttest.NewEntClientForTest 的 SQLite 内存库）。
//   - PermissionMenuRepo 在 repo_testkit.go 内已有同款私有构造器 testkitPermissionMenuRepo
//     （供 RoleRepo 组装用），此处为测试可见的导出副本，二者字段一致。
//   - RoleFieldPermissionRepoForTest 已在 repo_testkit5.go 导出，不在此重复。
package data

import (
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/mapper"

	biV1 "go-wind-admin/api/gen/go/bi/service/v1"
	printV1 "go-wind-admin/api/gen/go/print/service/v1"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/bidashboard"
)

// NewBiDashboardRepoForTest 逐字段复刻 bi_dashboard_repo.go 的 NewBiDashboardRepo
// （log 换 NopLogger），并调用 init()。
func NewBiDashboardRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *BiDashboardRepo {
	repo := &BiDashboardRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[biV1.BiDashboard, ent.BiDashboard](),
		kindConverter: mapper.NewEnumTypeConverter[biV1.BiDashboardKind, bidashboard.Kind](
			biV1.BiDashboardKind_name,
			biV1.BiDashboardKind_value,
		),
	}

	repo.init()

	return repo
}

// NewPrintTemplateRepoForTest 逐字段复刻 print_template_repo.go 的 NewPrintTemplateRepo
// （log 换 NopLogger），并调用 init()。
func NewPrintTemplateRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *PrintTemplateRepo {
	repo := &PrintTemplateRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[printV1.PrintTemplate, ent.PrintTemplate](),
	}

	repo.init()

	return repo
}

// NewPermissionMenuRepoForTest 逐字段复刻 permission_menu_repo.go 的 NewPermissionMenuRepo
// （log 换 NopLogger；该 repo 无 mapper/converter）。
func NewPermissionMenuRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *PermissionMenuRepo {
	return &PermissionMenuRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}
}

// NewBiDashboardRevisionRepoForTest 版本快照薄仓储的测试构造（无 init 逻辑，直连 SQLite）。
func NewBiDashboardRevisionRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *BiDashboardRevisionRepo {
	return &BiDashboardRevisionRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}
}

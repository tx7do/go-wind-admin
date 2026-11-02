// 跨包（service 层）测试装配：导出免 bootstrap.Context 的 repo 构造器，
// 字段初始化必须与生产 NewXxxRepo 逐字段一致，改生产构造器须同步此处。
//
// 说明：
//   - 本文件为 service 层（dashboard / file / file_transfer）SQLite 测试所需的构造器，
//     与各自 *_repo.go 的生产构造器逐字段对齐（含 mapper/converter 初始化与 init() 调用），
//     唯一差异是 log 一律 bLogger.NewHelper(bLogger.NopLogger())；entClient 由调用方传入
//     （测试场景为 enttest.NewEntClientForTest 的 SQLite 内存库）。
package data

import (
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/file"

	storageV1 "go-wind-admin/api/gen/go/storage/service/v1"
)

// NewDashboardRepoForTest 逐字段复刻 dashboard_repo.go 的 NewDashboardRepo
// （log 换 NopLogger；该 repo 无 mapper/converter）。
func NewDashboardRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *DashboardRepo {
	return &DashboardRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}
}

// NewFileRepoForTest 逐字段复刻 file_repo.go 的 NewFileRepo（log 换 NopLogger），
// 并调用 init()（生产 init 内完成 repository 装配与 converter 追加）。
func NewFileRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *FileRepo {
	repo := &FileRepo{
		log:               bLogger.NewHelper(bLogger.NopLogger()),
		entClient:         entClient,
		mapper:            mapper.NewCopierMapper[storageV1.File, ent.File](),
		providerConverter: mapper.NewEnumTypeConverter[storageV1.OSSProvider, file.Provider](storageV1.OSSProvider_name, storageV1.OSSProvider_value),
	}

	repo.init()

	return repo
}

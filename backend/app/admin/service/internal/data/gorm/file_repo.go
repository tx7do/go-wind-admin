//go:build gorm_backend
// +build gorm_backend

// Package gorm 中的仓储是 ent 仓储的平行 gorm 镜像，作为"ent 为主力、gorm 为备选"脚手架的完整代码。
// 这些仓储为死代码：不接入 wire、不被 service 引用；采用者需要时自行装配。
//
// gorm 仓储不做租户隔离（ent 侧靠编译进生成代码的 privacy 策略自动注入，gorm 侧无此机制）。
// 直接切换 gorm 后端会有跨租户数据泄露风险，采用者须自行加 scope/plugin。
package gorm

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	gormDB "gorm.io/gorm"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	storageV1 "go-wind-admin/api/gen/go/storage/service/v1"
)

type FileRepo struct {
	client     *gormCrud.Client
	log        *log.Helper
	mapper     *mapper.CopierMapper[storageV1.File, models.File]
	repository *gormCrud.Repository[storageV1.File, models.File]

	providerConverter *mapper.EnumTypeConverter[storageV1.OSSProvider, string]
}

func NewFileRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *FileRepo {
	repo := &FileRepo{
		log:    ctx.NewLoggerHelper("file/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[storageV1.File, models.File](),

		providerConverter: mapper.NewEnumTypeConverter[storageV1.OSSProvider, string](storageV1.OSSProvider_name, storageV1.OSSProvider_value),
	}

	repo.init()

	return repo
}

func (r *FileRepo) init() {
	r.repository = gormCrud.NewRepository[storageV1.File, models.File](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.providerConverter.NewConverterPair())
}

// formatSize 返回格式化后的文本，例如 "512B", "1.5KB"。
// 对字节单位返回整数；对其它单位保留最多两位小数并去掉多余的 0。
func (r *FileRepo) formatSize(size int64) string {
	if size <= 0 {
		return "0B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	s := float64(size)
	i := 0
	for s >= 1024 && i < len(units)-1 {
		s /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d%s", size, units[i])
	}
	v := math.Round(s*100) / 100
	str := fmt.Sprintf("%.2f", v)
	str = strings.TrimRight(strings.TrimRight(str, "0"), ".")
	return str + units[i]
}

func (r *FileRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, storageV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *FileRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*storageV1.ListFileResponse, error) {
	if req == nil {
		return nil, storageV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, storageV1.ErrorInternalServerError("query list failed")
	}
	if ret == nil {
		return &storageV1.ListFileResponse{Total: 0, Items: nil}, nil
	}

	return &storageV1.ListFileResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *FileRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, storageV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *FileRepo) Get(ctx context.Context, req *storageV1.GetFileRequest) (*storageV1.File, error) {
	if req == nil {
		return nil, storageV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *storageV1.GetFileRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, storageV1.ErrorNotFound("file not found")
		}
		r.log.Errorf("query file failed: %s", err.Error())
		return nil, storageV1.ErrorInternalServerError("query file failed")
	}

	return dto, nil
}

func (r *FileRepo) Create(ctx context.Context, req *storageV1.CreateFileRequest) error {
	if req == nil || req.Data == nil {
		return storageV1.ErrorBadRequest("invalid parameter")
	}

	if req.Data.Size != nil {
		req.Data.SizeFormat = trans.Ptr(r.formatSize(int64(req.Data.GetSize())))
	}

	if _, err := r.repository.Create(ctx, r.client.DB, req.Data, nil); err != nil {
		r.log.Errorf("insert file failed: %s", err.Error())
		return storageV1.ErrorInternalServerError("insert file failed")
	}

	return nil
}

func (r *FileRepo) Update(ctx context.Context, req *storageV1.UpdateFileRequest) error {
	if req == nil || req.Data == nil {
		return storageV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return storageV1.ErrorBadRequest("id is required")
	}

	if req.Data.Size != nil {
		req.Data.SizeFormat = trans.Ptr(r.formatSize(int64(req.Data.GetSize())))
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &storageV1.CreateFileRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	if _, err := r.repository.UpdateWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}, req.Data, req.GetUpdateMask()); err != nil {
		r.log.Errorf("update file failed: %s", err.Error())
		return storageV1.ErrorInternalServerError("update file failed")
	}

	return nil
}

func (r *FileRepo) Delete(ctx context.Context, req *storageV1.DeleteFileRequest) error {
	if req == nil {
		return storageV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}); err != nil {
		r.log.Errorf("delete file failed: %s", err.Error())
		return storageV1.ErrorInternalServerError("delete file failed")
	}

	return nil
}

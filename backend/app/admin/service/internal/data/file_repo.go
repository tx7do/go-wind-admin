package data

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/file"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	fileV1 "go-wind-admin/api/gen/go/file/service/v1"
)

type FileRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper            *mapper.CopierMapper[fileV1.File, ent.File]
	providerConverter *mapper.EnumTypeConverter[fileV1.OSSProvider, file.Provider]

	repository *entCrud.Repository[
		ent.FileQuery, ent.FileSelect,
		ent.FileCreate, ent.FileCreateBulk,
		ent.FileUpdate, ent.FileUpdateOne,
		ent.FileDelete,
		predicate.File,
		fileV1.File, ent.File,
	]
}

func NewFileRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *FileRepo {
	repo := &FileRepo{
		log:               ctx.NewLoggerHelper("file/repo/admin-service"),
		entClient:         entClient,
		mapper:            mapper.NewCopierMapper[fileV1.File, ent.File](),
		providerConverter: mapper.NewEnumTypeConverter[fileV1.OSSProvider, file.Provider](fileV1.OSSProvider_name, fileV1.OSSProvider_value),
	}

	repo.init()

	return repo
}

func (r *FileRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.FileQuery, ent.FileSelect,
		ent.FileCreate, ent.FileCreateBulk,
		ent.FileUpdate, ent.FileUpdateOne,
		ent.FileDelete,
		predicate.File,
		fileV1.File, ent.File,
	](r.mapper)

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

func (r *FileRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().File.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, fileV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *FileRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*fileV1.ListFileResponse, error) {
	if req == nil {
		return nil, fileV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().File.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &fileV1.ListFileResponse{Total: 0, Items: nil}, nil
	}

	return &fileV1.ListFileResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *FileRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().File.Query().
		Where(file.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, fileV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *FileRepo) Get(ctx context.Context, req *fileV1.GetFileRequest) (*fileV1.File, error) {
	if req == nil {
		return nil, fileV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().File.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *fileV1.GetFileRequest_Id:
		whereCond = append(whereCond, file.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *FileRepo) Create(ctx context.Context, req *fileV1.CreateFileRequest) error {
	if req == nil || req.Data == nil {
		return fileV1.ErrorBadRequest("invalid parameter")
	}

	if req.Data.Size != nil {
		req.Data.SizeFormat = trans.Ptr(r.formatSize(int64(req.Data.GetSize())))
	}

	builder := r.entClient.Client().File.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableProvider(r.providerConverter.ToEntity(req.Data.Provider)).
		SetNillableBucketName(req.Data.BucketName).
		SetNillableFileDirectory(req.Data.FileDirectory).
		SetNillableFileGUID(req.Data.FileGuid).
		SetNillableSaveFileName(req.Data.SaveFileName).
		SetNillableFileName(req.Data.FileName).
		SetNillableExtension(req.Data.Extension).
		SetNillableSize(req.Data.Size).
		SetNillableSizeFormat(req.Data.SizeFormat).
		SetNillableLinkURL(req.Data.LinkUrl).
		SetNillableContentHash(req.Data.ContentHash).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("insert file failed: %s", err.Error())
		return fileV1.ErrorInternalServerError("insert file failed")
	}

	return nil
}

func (r *FileRepo) Update(ctx context.Context, req *fileV1.UpdateFileRequest) error {
	if req == nil || req.Data == nil {
		return fileV1.ErrorBadRequest("invalid parameter")
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
			createReq := &fileV1.CreateFileRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	builder := r.entClient.Client().Debug().File.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *fileV1.File) {
			builder.
				SetNillableProvider(r.providerConverter.ToEntity(req.Data.Provider)).
				SetNillableBucketName(req.Data.BucketName).
				SetNillableFileDirectory(req.Data.FileDirectory).
				SetNillableFileGUID(req.Data.FileGuid).
				SetNillableSaveFileName(req.Data.SaveFileName).
				SetNillableFileName(req.Data.FileName).
				SetNillableExtension(req.Data.Extension).
				SetNillableSize(req.Data.Size).
				SetNillableSizeFormat(req.Data.SizeFormat).
				SetNillableLinkURL(req.Data.LinkUrl).
				SetNillableContentHash(req.Data.ContentHash).
				SetNillableCreatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(file.FieldID, req.GetId()))
		},
	)

	return err
}

func (r *FileRepo) Delete(ctx context.Context, req *fileV1.DeleteFileRequest) error {
	if req == nil {
		return fileV1.ErrorBadRequest("invalid parameter")
	}

	if err := r.entClient.Client().File.DeleteOneID(req.GetId()).Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return fileV1.ErrorNotFound("file not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return fileV1.ErrorInternalServerError("delete failed")
	}

	return nil
}

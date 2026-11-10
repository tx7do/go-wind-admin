package data

import (
	"context"
	"time"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	scriptV1 "go-wind-admin/api/gen/go/script/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	"go-wind-admin/app/admin/service/internal/data/ent/scriptlog"
)

// ScriptLogRepo 脚本执行日志仓储：仅写入与简单查询，不走 proto/管理面。
// 高频写入路径，方法保持最小面。
type ScriptLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper *mapper.CopierMapper[scriptV1.ScriptLog, ent.ScriptLog]

	repository *entCrud.Repository[
		ent.ScriptLogQuery, ent.ScriptLogSelect,
		ent.ScriptLogCreate, ent.ScriptLogCreateBulk,
		ent.ScriptLogUpdate, ent.ScriptLogUpdateOne,
		ent.ScriptLogDelete,
		predicate.ScriptLog,
		scriptV1.ScriptLog, ent.ScriptLog,
	]
}

func NewScriptLogRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *ScriptLogRepo {
	repo := &ScriptLogRepo{
		entClient: entClient,
		log:       ctx.NewLoggerHelper("script-log/repo/admin-service"),
		mapper:    mapper.NewCopierMapper[scriptV1.ScriptLog, ent.ScriptLog](),
	}
	repo.repository = entCrud.NewRepository[
		ent.ScriptLogQuery, ent.ScriptLogSelect,
		ent.ScriptLogCreate, ent.ScriptLogCreateBulk,
		ent.ScriptLogUpdate, ent.ScriptLogUpdateOne,
		ent.ScriptLogDelete,
		predicate.ScriptLog,
		scriptV1.ScriptLog, ent.ScriptLog,
	](repo.mapper)

	repo.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	repo.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	return repo
}

// ScriptLogRecord 一次脚本执行的记录。
type ScriptLogRecord struct {
	ScriptID   int64
	ScriptName string
	Language   string
	Trigger    string // hook / task / test_run / manual
	HookPoint  string
	Version    uint32
	Success    bool
	DurationMS int64
	Error      string
}

// Record 落一条执行日志。失败只记运维日志，不影响业务调用方。
func (r *ScriptLogRepo) Record(ctx context.Context, rec ScriptLogRecord) {
	err := r.entClient.Client().ScriptLog.Create().
		SetNillableScriptID(nonZero(rec.ScriptID)).
		SetNillableScriptName(nonEmptyStr(rec.ScriptName)).
		SetNillableLanguage(nonEmptyStr(rec.Language)).
		SetNillableTriggerType(nonEmptyStr(rec.Trigger)).
		SetNillableHookPoint(nonEmptyStr(rec.HookPoint)).
		SetNillableVersion(nonZero(rec.Version)).
		SetSuccess(rec.Success).
		SetDurationMs(rec.DurationMS).
		SetNillableError(nonEmptyStr(rec.Error)).
		SetCreatedAt(time.Now()).
		Exec(ctx)
	if err != nil {
		r.log.Errorf(ctx, "record script log failed: %s", err.Error())
	}
}

// PurgeBefore 删除指定时间之前的日志（滚动清理），返回删除行数。
func (r *ScriptLogRepo) PurgeBefore(ctx context.Context, before time.Time) (int, error) {
	deleted, err := r.entClient.Client().ScriptLog.Delete().
		Where(scriptlog.CreatedAtLT(before)).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return deleted, nil
}

func nonZero[T ~uint32 | ~int64](v T) *T {
	if v == 0 {
		return nil
	}
	return &v
}

func nonEmptyStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// List 分页查询执行日志（repository 通用路径：query JSON 过滤 + created_at 倒序）。
func (r *ScriptLogRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*scriptV1.ListScriptLogsResponse, error) {
	if req == nil {
		return nil, scriptV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().ScriptLog.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &scriptV1.ListScriptLogsResponse{Total: 0, Items: nil}, nil
	}

	return &scriptV1.ListScriptLogsResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

// Count 统计日志数量。
func (r *ScriptLogRepo) CountLog(ctx context.Context) (*scriptV1.CountScriptLogsResponse, error) {
	count, err := r.entClient.Client().ScriptLog.Query().Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "count script logs failed: %s", err.Error())
		return nil, scriptV1.ErrorInternalServerError("count script logs failed")
	}
	return &scriptV1.CountScriptLogsResponse{Count: int64(count)}, nil
}

// Purge 清理指定时间之前的日志。
func (r *ScriptLogRepo) Purge(ctx context.Context, before time.Time) (int64, error) {
	deleted, err := r.PurgeBefore(ctx, before)
	if err != nil {
		r.log.Errorf(ctx, "purge script logs failed: %s", err.Error())
		return 0, scriptV1.ErrorInternalServerError("purge script logs failed")
	}
	return int64(deleted), nil
}

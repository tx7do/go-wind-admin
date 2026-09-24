package data

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"entgo.io/ent/dialect/sql"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"github.com/redis/go-redis/v9"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	configV1 "go-wind-admin/api/gen/go/config/service/v1"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	"go-wind-admin/app/admin/service/internal/data/ent/sysconfig"
)

// sysConfigCacheMaxEntries 参数缓存条目上限：合法的参数键是管理员配置的有限集合，
// 超限说明有调用在拿随机键打缓存，此时放弃缓存该键（每次查库兜底），防内存被刷爆。
const sysConfigCacheMaxEntries = 4096

// configInvalidateChannel 参数失效广播频道：写路径发布被失效的键，
// 所有实例订阅后清除各自进程内缓存——多实例部署下参数变更即时全局生效。
const configInvalidateChannel = "gowind:config:invalidate"

// sysConfigCacheEntry 参数缓存条目；found=false 表示“库里没有该键”的负缓存。
type sysConfigCacheEntry struct {
	found     bool
	value     string
	valueType sysconfig.ValueType
}

type ConfigRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	rdb       *redis.Client
	log       *bLogger.Helper

	mapper             *mapper.CopierMapper[configV1.Config, ent.SysConfig]
	valueTypeConverter *mapper.EnumTypeConverter[configV1.Config_ConfigValueType, sysconfig.ValueType]

	repository *entCrud.Repository[
		ent.SysConfigQuery, ent.SysConfigSelect,
		ent.SysConfigCreate, ent.SysConfigCreateBulk,
		ent.SysConfigUpdate, ent.SysConfigUpdateOne,
		ent.SysConfigDelete,
		predicate.SysConfig,
		configV1.Config, ent.SysConfig,
	]

	// 参数读取器（accessor）的进程内缓存：key → 条目，写路径（Create/Update/Delete）同步失效。
	// 依赖“全部写路径都经本 repo、单进程持有写权”的假设；多实例部署需改造为共享缓存。
	cacheMu sync.RWMutex
	cache   map[string]sysConfigCacheEntry
}

func NewConfigRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client], rdb *redis.Client) *ConfigRepo {
	repo := &ConfigRepo{
		log:       ctx.NewLoggerHelper("config/repo/admin-service"),
		entClient: entClient,
		rdb:       rdb,
		mapper:    mapper.NewCopierMapper[configV1.Config, ent.SysConfig](),
		valueTypeConverter: mapper.NewEnumTypeConverter[configV1.Config_ConfigValueType, sysconfig.ValueType](
			configV1.Config_ConfigValueType_name,
			configV1.Config_ConfigValueType_value,
		),
		cache: make(map[string]sysConfigCacheEntry),
	}

	repo.init()

	// 多实例失效广播订阅：收到其他实例的写失效通知后清除本进程缓存条目
	go repo.subscribeInvalidations(context.Background())

	return repo
}

func (r *ConfigRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.SysConfigQuery, ent.SysConfigSelect,
		ent.SysConfigCreate, ent.SysConfigCreateBulk,
		ent.SysConfigUpdate, ent.SysConfigUpdateOne,
		ent.SysConfigDelete,
		predicate.SysConfig,
		configV1.Config, ent.SysConfig,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.valueTypeConverter.NewConverterPair())
}

func (r *ConfigRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (*configV1.CountConfigResponse, error) {
	builder := r.entClient.Client().SysConfig.Query()

	whereSelectors, _, _ := r.repository.BuildListSelectorWithPaging(builder, req)
	if len(whereSelectors) != 0 {
		builder.Modify(whereSelectors...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query config count failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("query config count failed")
	}

	return &configV1.CountConfigResponse{
		Count: uint64(count),
	}, nil
}

func (r *ConfigRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*configV1.ListConfigResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().SysConfig.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &configV1.ListConfigResponse{Total: 0, Items: nil}, nil
	}

	return &configV1.ListConfigResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *ConfigRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().SysConfig.Query().
		Where(sysconfig.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *ConfigRepo) Get(ctx context.Context, req *configV1.GetConfigRequest) (*configV1.Config, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().SysConfig.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *configV1.GetConfigRequest_Id:
		whereCond = append(whereCond, sysconfig.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *ConfigRepo) Create(ctx context.Context, req *configV1.CreateConfigRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.newConfigCreate(req.Data)

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "insert config failed: %s", err.Error())
		if ent.IsConstraintError(err) {
			return adminV1.ErrorBadRequest("config key already exists")
		}
		return adminV1.ErrorInternalServerError("insert config failed")
	}

	r.invalidateCacheKey(ctx, req.Data.GetKey())

	return nil
}

func (r *ConfigRepo) newConfigCreate(cfg *configV1.Config) *ent.SysConfigCreate {
	builder := r.entClient.Client().SysConfig.Create().
		SetNillableName(cfg.Name).
		SetNillableKey(cfg.Key).
		SetNillableValue(cfg.Value).
		SetNillableIsBuiltIn(cfg.IsBuiltIn).
		SetNillableCreatedBy(cfg.CreatedBy).
		SetCreatedAt(time.Now())

	// value_type 为 proto 零值（CONFIG_VALUE_TYPE_INVALID）时跳过：ent schema 未声明该值，
	// 直传会触发 ValueTypeValidator 失败。未指定时落库走 schema 默认 STRING。
	if cfg.ValueType != nil && *cfg.ValueType != configV1.Config_CONFIG_VALUE_TYPE_INVALID {
		builder.SetNillableValueType(r.valueTypeConverter.ToEntity(cfg.ValueType))
	}

	if cfg.Id != nil {
		builder.SetID(cfg.GetId())
	}

	return builder
}

func (r *ConfigRepo) Update(ctx context.Context, req *configV1.UpdateConfigRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return adminV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &configV1.CreateConfigRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	// 缓存失效需要旧键：改键场景新旧两个键都要失效。
	// 必须在 UpdateX 前取旧键、在 UpdateX 前快照新键（FilterByFieldMask 会在调用中清掉不在掩码里的字段）。
	newKey := req.Data.GetKey()
	var oldKey string
	if old, err := r.entClient.Client().SysConfig.Query().
		Where(sysconfig.IDEQ(req.GetId())).
		Only(ctx); err == nil && old != nil {
		oldKey = derefStrP(old.Key)
	}

	builder := r.entClient.Client().SysConfig.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *configV1.Config) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableKey(req.Data.Key).
				SetNillableValue(req.Data.Value).
				SetNillableIsBuiltIn(req.Data.IsBuiltIn).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())

			// value_type 为 proto 零值（CONFIG_VALUE_TYPE_INVALID）时跳过：ent schema 未声明该值，
			// 直传会触发 ValueTypeValidator 失败。未指定即不更新该字段。
			if req.Data.ValueType != nil && *req.Data.ValueType != configV1.Config_CONFIG_VALUE_TYPE_INVALID {
				builder.SetNillableValueType(r.valueTypeConverter.ToEntity(req.Data.ValueType))
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(sysconfig.FieldID, req.GetId()))
		},
	)
	if err != nil {
		r.log.Errorf(ctx, "update config failed: %s", err.Error())
		if ent.IsConstraintError(err) {
			return adminV1.ErrorBadRequest("config key already exists")
		}
		return adminV1.ErrorInternalServerError("update config failed")
	}

	r.invalidateCacheKey(ctx, oldKey)
	r.invalidateCacheKey(ctx, newKey)

	return nil
}

func (r *ConfigRepo) Delete(ctx context.Context, req *configV1.DeleteConfigRequest) error {
	if req == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	// 删除前先取行：内置参数禁删守卫 + 取旧键做缓存失效
	entity, err := r.entClient.Client().SysConfig.Query().
		Where(sysconfig.IDEQ(req.GetId())).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// 幂等删除：目标不存在视为已删除
			return nil
		}
		r.log.Errorf(ctx, "query config before delete failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("query config failed")
	}
	if entity.IsBuiltIn != nil && *entity.IsBuiltIn {
		return adminV1.ErrorBadRequest("built-in config cannot be deleted")
	}

	builder := r.entClient.Client().SysConfig.Delete()

	_, err = r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.EQ(sysconfig.FieldID, req.GetId()))
	})
	if err != nil {
		r.log.Errorf(ctx, "delete config failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("delete config failed")
	}

	r.invalidateCacheKey(ctx, derefStrP(entity.Key))

	return nil
}

// SeedDefaults 按键缺一补一地播种内置平台参数（启动期）：
// 键不存在的按给定默认值创建，键已存在（含值被管理员改过）的跳过，不覆盖。
// 与其他默认数据的表级 count==0 守卫不同——管理员自建行不应阻断内置键补种，
// 且补种绝不能把管理员改过的阈值重置回默认。
func (r *ConfigRepo) SeedDefaults(ctx context.Context, defaults []*configV1.Config) error {
	for _, item := range defaults {
		if item == nil || item.GetKey() == "" {
			continue
		}
		exists, err := r.entClient.Client().SysConfig.Query().
			Where(sysconfig.KeyEQ(item.GetKey())).
			Exist(ctx)
		if err != nil {
			r.log.Errorf(ctx, "seed config %q: exists query failed: %s", item.GetKey(), err.Error())
			return err
		}
		if exists {
			continue
		}
		if err := r.newConfigCreate(item).Exec(ctx); err != nil {
			r.log.Errorf(ctx, "seed config %q: insert failed: %s", item.GetKey(), err.Error())
			return err
		}
		r.invalidateCacheKey(ctx, item.GetKey())
	}
	return nil
}

// —— 服务侧参数读取器（带缓存）——
//
// 其他服务通过 wiring 注入 *ConfigRepo 后按键读取运行时参数，例如：
//
//	showCaptcha := s.configRepo.GetConfigBool(ctx, "sys.login.captchaEnabled", true)
//
// 缓存语义：进程内 per-key 懒加载 + 负缓存（键不存在也缓存，防不存在的键反复打库）；
// Create/Update/Delete 同步失效受影响键。依赖“全部写路径都经本 repo、单进程持有写权”的
// 假设，多实例部署需改造为共享缓存（Redis 等）后再放开消费方。

// invalidateCacheKey 失效单个键的本地缓存并向 Redis 广播（空键 no-op）。
// 广播为尽力而为：Redis 不可用时仅告警，本地失效不受影响。
func (r *ConfigRepo) invalidateCacheKey(ctx context.Context, key string) {
	if key == "" {
		return
	}
	r.cacheMu.Lock()
	delete(r.cache, key)
	r.cacheMu.Unlock()

	if r.rdb != nil {
		if err := r.rdb.Publish(ctx, configInvalidateChannel, key).Err(); err != nil {
			r.log.Warnf(ctx, "publish config invalidate %q failed: %s", key, err.Error())
		}
	}
}

// subscribeInvalidations 订阅失效广播，清除本进程内对应缓存条目。
// 连接断开由 go-redis 自动重连；进程退出时随连接一起消亡。
func (r *ConfigRepo) subscribeInvalidations(ctx context.Context) {
	if r.rdb == nil {
		return
	}
	sub := r.rdb.Subscribe(ctx, configInvalidateChannel)
	defer sub.Close()

	for msg := range sub.Channel() {
		r.cacheMu.Lock()
		delete(r.cache, msg.Payload)
		r.cacheMu.Unlock()
	}
}

// getCachedEntry 取参数条目。第二个返回值表示库里确实存在该键；
// 缓存命中（含负缓存）不查库，查库异常不写缓存以便下次重试。
func (r *ConfigRepo) getCachedEntry(ctx context.Context, key string) (sysConfigCacheEntry, bool) {
	if key == "" {
		return sysConfigCacheEntry{}, false
	}

	r.cacheMu.RLock()
	e, ok := r.cache[key]
	r.cacheMu.RUnlock()
	if ok {
		return e, e.found
	}

	e = sysConfigCacheEntry{}
	entity, err := r.entClient.Client().SysConfig.Query().
		Where(sysconfig.KeyEQ(key)).
		Only(ctx)
	switch {
	case err == nil:
		e.found = true
		e.value = derefStrP(entity.Value)
		if entity.ValueType != nil {
			e.valueType = *entity.ValueType
		}
	case ent.IsNotFound(err):
		// 负缓存：键不存在也记录
	default:
		r.log.Errorf(ctx, "query config by key failed: %s", err.Error())
		return e, false
	}

	r.cacheMu.Lock()
	if len(r.cache) < sysConfigCacheMaxEntries {
		r.cache[key] = e
	}
	r.cacheMu.Unlock()

	return e, e.found
}

// GetConfigBool 读布尔参数；键不存在、声明类型不符或值解析失败时返回 def。
func (r *ConfigRepo) GetConfigBool(ctx context.Context, key string, def bool) bool {
	e, ok := r.getCachedEntry(ctx, key)
	if !ok {
		return def
	}
	if e.valueType != sysconfig.ValueTypeBool {
		r.log.Warnf(ctx, "config %q declared as %s, read as bool: fallback to default", key, e.valueType)
		return def
	}
	v, err := strconv.ParseBool(strings.TrimSpace(e.value))
	if err != nil {
		r.log.Warnf(ctx, "config %q value %q not parseable as bool: %s", key, e.value, err.Error())
		return def
	}
	return v
}

// GetConfigInt 读整数参数；键不存在、声明类型不符或值解析失败时返回 def。
func (r *ConfigRepo) GetConfigInt(ctx context.Context, key string, def int) int {
	e, ok := r.getCachedEntry(ctx, key)
	if !ok {
		return def
	}
	if e.valueType != sysconfig.ValueTypeInt {
		r.log.Warnf(ctx, "config %q declared as %s, read as int: fallback to default", key, e.valueType)
		return def
	}
	v, err := strconv.Atoi(strings.TrimSpace(e.value))
	if err != nil {
		r.log.Warnf(ctx, "config %q value %q not parseable as int: %s", key, e.value, err.Error())
		return def
	}
	return v
}

// GetConfigString 读字符串参数；键不存在或声明类型不符时返回 def。
func (r *ConfigRepo) GetConfigString(ctx context.Context, key string, def string) string {
	e, ok := r.getCachedEntry(ctx, key)
	if !ok {
		return def
	}
	if e.valueType != sysconfig.ValueTypeString {
		r.log.Warnf(ctx, "config %q declared as %s, read as string: fallback to default", key, e.valueType)
		return def
	}
	return e.value
}

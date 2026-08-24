package data

import (
	"context"
	"strings"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	redisCacheV1 "go-wind-admin/api/gen/go/redis_cache/service/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// RedisCacheMonitorRepo 提供 Redis 运行时信息的只读聚合视图。
// 数据来源为 Redis 的 INFO / DBSIZE / SLOWLOG GET 命令，全部为只读查询，不做任何写操作。
// 仿照 LoginRateLimiter 的封装风格：直接持有 *redis.Client，在 data 层完成 Redis 访问。
type RedisCacheMonitorRepo struct {
	log *bLogger.Helper
	rdb *redis.Client
}

// NewRedisCacheMonitorRepo 构造 Redis 缓存监控仓库。
func NewRedisCacheMonitorRepo(ctx *bootstrap.Context, rdb *redis.Client) *RedisCacheMonitorRepo {
	return &RedisCacheMonitorRepo{
		rdb: rdb,
		log: ctx.NewLoggerHelper("redis-cache-monitor/repo/admin-service"),
	}
}

// slowLogFetchLimit SLOWLOG GET 抓取的最近条目数上限。
const slowLogFetchLimit = 10

// GetInfo 聚合返回 Redis 的 INFO（按 section 拆分）、DBSIZE、SLOWLOG 三类只读运维指标。
// 任一命令失败时仅记录告警并将对应字段置空，不阻断其余字段的返回——
// 对齐 LoginRateLimiter 的 fail-soft 风格，避免单点故障导致整个监控视图不可用。
func (r *RedisCacheMonitorRepo) GetInfo(ctx context.Context) (*redisCacheV1.RedisCacheMonitorInfo, error) {
	info := &redisCacheV1.RedisCacheMonitorInfo{}

	if r == nil || r.rdb == nil {
		// Redis 不可用时返回空视图，而非错误。
		return info, nil
	}

	// INFO：原始串按 section/entry 拆分后落入泛型结构。
	if raw, err := r.rdb.Info(ctx).Result(); err != nil {
		r.log.Errorf(ctx, "redis INFO failed: %s", err.Error())
	} else {
		info.Sections = parseInfoSections(raw)
	}

	// DBSIZE：当前库 key 总数。
	if n, err := r.rdb.DBSize(ctx).Result(); err != nil {
		r.log.Errorf(ctx, "redis DBSIZE failed: %s", err.Error())
	} else if n >= 0 {
		info.DbSize = uint64(n)
	}

	// SLOWLOG GET：最近慢日志条目。
	if entries, err := r.rdb.SlowLogGet(ctx, slowLogFetchLimit).Result(); err != nil {
		r.log.Errorf(ctx, "redis SLOWLOG GET failed: %s", err.Error())
	} else {
		info.Slowlog = mapSlowLogEntries(entries)
	}

	return info, nil
}

// mapSlowLogEntries 将 redis.SlowLog 列表映射为 proto 消息列表。
// 字段一一对应：ID→Id、Time→CreatedAt（timestamppb）、Duration→DurationUsec（微秒）、
// Args→Args、ClientAddr→ClientAddr、ClientName→ClientName。
func mapSlowLogEntries(in []redis.SlowLog) []*redisCacheV1.SlowLogEntry {
	if len(in) == 0 {
		return nil
	}
	out := make([]*redisCacheV1.SlowLogEntry, 0, len(in))
	for i := range in {
		s := &in[i]
		out = append(out, &redisCacheV1.SlowLogEntry{
			Id:            s.ID,
			CreatedAt:     timestamppb.New(s.Time),
			DurationUsec:  s.Duration.Microseconds(),
			Args:          sanitizeSlowLogArgs(s.Args),
			ClientAddr:    s.ClientAddr,
			ClientName:    strings.ToValidUTF8(s.ClientName, "\uFFFD"),
		})
	}
	return out
}

// sanitizeSlowLogArgs 将 slowlog 命令参数净化为合法 UTF-8。
// Redis 命令参数二进制安全（如二进制序列化的 key/value），而 proto string 字段
// 强制 UTF-8——未净化直接透传会让响应在 HTTP codec 序列化阶段整体失败，
// 表现为与本 repo fail-soft 设计相悖的 500（message: invalid UTF-8）。
func sanitizeSlowLogArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, 0, len(args))
	for _, a := range args {
		out = append(out, strings.ToValidUTF8(a, "\uFFFD"))
	}
	return out
}

// parseInfoSections 解析 Redis INFO 原始串为 section/entry 树。
// INFO 输出格式：以 `# <SectionName>` 行标记新 section，其后 `key:value` 行为该 section 的条目，
// 空行与无法识别的行被忽略。Keyspace section 的 `db0:keys=...` 同样按 key/value 落入此结构，
// 前端按通用 kv 表渲染，无需特殊列。
func parseInfoSections(raw string) []*redisCacheV1.InfoSection {
	var sections []*redisCacheV1.InfoSection
	var current *redisCacheV1.InfoSection

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "# ") {
			name := strings.TrimSpace(line[2:])
			if name == "" {
				// 畸形 section 头：停止向其挂条目，避免误归属。
				current = nil
				continue
			}
			current = &redisCacheV1.InfoSection{Name: name}
			sections = append(sections, current)
			continue
		}

		idx := strings.IndexByte(line, ':')
		if idx < 0 {
			continue
		}
		if current == nil {
			// 无归属 section 的孤立条目，忽略。
			continue
		}
		current.Entries = append(current.Entries, &redisCacheV1.InfoEntry{
			Key:   line[:idx],
			Value: line[idx+1:],
		})
	}

	return sections
}
